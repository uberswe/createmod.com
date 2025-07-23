package aidescription

import (
	"createmod/internal/openai"
	"encoding/base64"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem/blob"
	"io"
	"net/http"
	"os"
	"time"
)

// Service represents the AI description service
type Service struct {
	openaiClient *openai.Client
	testMode     bool
	stopChan     chan struct{} // Channel to signal the background goroutine to stop
}

// New creates a new AI description service
func New(apiKey string, logger openai.Logger) *Service {
	return &Service{
		openaiClient: openai.NewClient(apiKey, logger),
		testMode:     false,
		stopChan:     make(chan struct{}),
	}
}

// StartScheduler starts a background goroutine that runs the service every 30 minutes
func (s *Service) StartScheduler(app *pocketbase.PocketBase) {
	go func() {
		// Run immediately on start
		s.ProcessSchematics(app)

		// Then run every 30 minutes
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.ProcessSchematics(app)
			case <-s.stopChan:
				app.Logger().Info("AI description scheduler stopped")
				return
			}
		}
	}()

	app.Logger().Info("AI description scheduler started (polling every 30 minutes)")
}

// Stop stops the background goroutine
func (s *Service) Stop() {
	close(s.stopChan)
}

// ProcessSchematics processes schematics without AI descriptions
func (s *Service) ProcessSchematics(app *pocketbase.PocketBase) {
	app.Logger().Info("AI description generation started")

	// Find schematics with empty ai_description (limit to 100)
	schematics, err := app.FindRecordsByFilter(
		"schematics",
		"(ai_description = '' || ai_description = null) && moderated = 1 && deleted = null",
		"-created",
		100,
		0,
	)
	if err != nil {
		app.Logger().Error("Failed to find schematics without AI descriptions", "error", err)
		return
	}

	app.Logger().Info("Found schematics without AI descriptions", "count", len(schematics))

	// Process only one schematic in test mode
	if s.testMode && len(schematics) > 0 {
		app.Logger().Info("Test mode: processing only one schematic")
		s.processSchematic(app, schematics[0])
		return
	}

	// Process schematics with rate limiting (1 request per second)
	for i, schematic := range schematics {
		s.processSchematic(app, schematic)

		// Rate limiting: wait 1 second between requests
		if i < len(schematics)-1 {
			time.Sleep(time.Second)
		}
	}

	app.Logger().Info("AI description generation completed")
}

// processSchematic processes a single schematic
func (s *Service) processSchematic(app *pocketbase.PocketBase, schematic *core.Record) {
	schematicID := schematic.Id
	app.Logger().Info("Processing schematic", "id", schematicID, "title", schematic.GetString("title"))

	// construct the full file key by concatenating the record storage path with the specific filename
	imgPath := schematic.BaseFilesPath() + "/" + schematic.GetString("featured_image")

	// initialize the filesystem
	fsys, err := app.NewFilesystem()
	if err != nil {
		app.Logger().Error("Failed to make new filesystem",
			"error", err,
			"id", schematicID)
		return
	}
	defer fsys.Close()

	// retrieve a file reader for the avatar key
	r, err := fsys.GetReader(imgPath)
	if err != nil {
		app.Logger().Error("Failed to get reader",
			"error", err,
			"id", schematicID)
		return
	}
	defer r.Close()
	// Generate description for the featured image
	description, err := s.generateDescription(app, r)
	if err != nil {
		app.Logger().Error("Failed to generate description for featured image",
			"error", err,
			"id", schematicID)
		return
	}

	// Tried looping all gallery images here but in 90% of cases the featured image will be sufficient

	// Update the schematic with the generated description and
	schematic.Set("ai_description", description)

	if err := app.Save(schematic); err != nil {
		app.Logger().Error("Failed to save schematic with AI description",
			"error", err,
			"id", schematicID)
		return
	}

	app.Logger().Info("Successfully updated schematic with AI description",
		"id", schematicID,
		"description_length", len(description))
}

// generateDescription generates a description for an image
func (s *Service) generateDescription(app *pocketbase.PocketBase, r *blob.Reader) (string, error) {
	app.Logger().Debug("Generating description for image")

	// Read the image data
	imageData, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Encode the image data as base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Generate description using OpenAI with base64-encoded image
	description, err := s.openaiClient.GenerateImageDescriptionFromBase64(base64Image)
	if err != nil {
		return "", fmt.Errorf("failed to generate description: %w", err)
	}

	return description, nil
}

// DownloadAndProcessImage downloads an image and processes it
func (s *Service) DownloadAndProcessImage(app *pocketbase.PocketBase, imageURL string, tempFilePath string) (string, error) {
	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	// Create a temporary file
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer file.Close()

	// Copy the image data to the file
	_, err = file.ReadFrom(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write image data to file: %w", err)
	}

	// Generate description from the file
	description, err := s.openaiClient.GenerateImageDescriptionFromFile(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to generate description from file: %w", err)
	}

	// Clean up the temporary file
	os.Remove(tempFilePath)

	return description, nil
}
