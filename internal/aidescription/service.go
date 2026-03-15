package aidescription

import (
	"createmod/internal/openai"
	"createmod/internal/storage"
	"createmod/internal/store"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
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

// StartScheduler starts a background goroutine that runs the service every 30 minutes.
func (s *Service) StartScheduler(storageSvc *storage.Service, appStore *store.Store) {
	go func() {
		// Run immediately on start
		s.ProcessSchematics(storageSvc, appStore)

		// Then run every 30 minutes
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.ProcessSchematics(storageSvc, appStore)
			case <-s.stopChan:
				slog.Info("AI description scheduler stopped")
				return
			}
		}
	}()

	slog.Info("AI description scheduler started (polling every 30 minutes)")
}

// Stop stops the background goroutine
func (s *Service) Stop() {
	close(s.stopChan)
}

// ProcessSchematics processes schematics without AI descriptions.
func (s *Service) ProcessSchematics(storageSvc *storage.Service, appStore *store.Store) {
	slog.Info("AI description generation started")

	if !s.openaiClient.HasApiKey() {
		slog.Error("OpenAI API key is required")
		return
	}

	ctx := context.Background()

	// Find schematics with empty ai_description (limit to 100)
	schematics, err := appStore.Schematics.ListAllForIndex(ctx)
	if err != nil {
		slog.Error("Failed to find schematics for AI descriptions", "error", err)
		return
	}

	// Filter to those without AI descriptions, moderated, and not deleted.
	// Limit to 10 per run to control OpenAI costs (uses gpt-4.1 with image input).
	var pending []store.Schematic
	for _, sc := range schematics {
		if sc.AIDescription == "" && sc.Moderated && sc.Deleted == nil {
			pending = append(pending, sc)
		}
		if len(pending) >= 10 {
			break
		}
	}

	slog.Info("Found schematics without AI descriptions", "count", len(pending))

	// Process only one schematic in test mode
	if s.testMode && len(pending) > 0 {
		slog.Info("Test mode: processing only one schematic")
		s.processSchematic(storageSvc, appStore, &pending[0])
		return
	}

	// Process schematics with rate limiting (1 request per second)
	for i := range pending {
		s.processSchematic(storageSvc, appStore, &pending[i])

		// Rate limiting: wait 1 second between requests
		if i < len(pending)-1 {
			time.Sleep(time.Second)
		}
	}

	slog.Info("AI description generation completed")
}

// processSchematic processes a single schematic.
func (s *Service) processSchematic(storageSvc *storage.Service, appStore *store.Store, schematic *store.Schematic) {
	slog.Info("Processing schematic", "id", schematic.ID, "title", schematic.Title)

	if storageSvc == nil || schematic.FeaturedImage == "" {
		slog.Warn("Skipping schematic: no storage or no featured image", "id", schematic.ID)
		return
	}

	// Use legacy PB collection ID prefix for S3 key lookup
	collPrefix := storage.CollectionPrefix("schematics")

	ctx := context.Background()
	r, err := storageSvc.Download(ctx, collPrefix, schematic.ID, schematic.FeaturedImage)
	if err != nil {
		slog.Error("Failed to download featured image",
			"error", err,
			"id", schematic.ID)
		return
	}
	defer r.Close()

	// Generate description for the featured image
	description, err := s.generateDescription(r)
	if err != nil {
		slog.Error("Failed to generate description for featured image",
			"error", err,
			"id", schematic.ID)
		return
	}

	// Update the schematic with the generated description
	schematic.AIDescription = description
	if err := appStore.Schematics.Update(ctx, schematic); err != nil {
		slog.Error("Failed to save schematic with AI description",
			"error", err,
			"id", schematic.ID)
		return
	}

	slog.Info("Successfully updated schematic with AI description",
		"id", schematic.ID,
		"description_length", len(description))
}

// generateDescription generates a description for an image.
func (s *Service) generateDescription(r io.Reader) (string, error) {
	slog.Debug("Generating description for image")

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

// TranslateToEnglish is a thin wrapper around the OpenAI client to translate text to English.
// On any error or when the API key is missing, it returns the original text unchanged.
func (s *Service) TranslateToEnglish(text string) (string, error) {
	if s == nil || s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return text, fmt.Errorf("openai not configured")
	}
	out, err := s.openaiClient.TranslateToEnglish(text)
	if err != nil {
		return text, err
	}
	return out, nil
}

// BackfillTranslations is deprecated and has been replaced by the translation.Service scheduler.
// This method is kept as a no-op for backwards compatibility.
func (s *Service) BackfillTranslations() {
	slog.Info("BackfillTranslations is deprecated; use translation.Service instead")
}
