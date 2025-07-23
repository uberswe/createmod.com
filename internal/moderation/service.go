package moderation

import (
	"createmod/internal/openai"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Service represents a moderation service
type Service struct {
	openaiClient *openai.Client
	logger       openai.Logger
}

// NewService creates a new moderation service
func NewService(apiKey string, logger openai.Logger) *Service {
	return &Service{
		openaiClient: openai.NewClient(apiKey, logger),
		logger:       logger,
	}
}

// ModerationResult represents the result of a moderation check
type ModerationResult struct {
	Approved bool
	Reason   string
}

// CheckSchematic checks if a schematic's content violates content policies
func (s *Service) CheckSchematic(title, description, featuredImagePath string) (*ModerationResult, error) {
	// Combine title and description for text content
	textContent := fmt.Sprintf("Title: %s\nDescription: %s", title, description)

	// Send content to OpenAI moderation API
	var response *openai.ModerationResponse
	var err error

	// Check if the featured image path is a valid URL
	if s.isValidURL(featuredImagePath) {
		// Use the multi-modal moderation with both text and image
		response, err = s.openaiClient.ModerateTextAndImage(textContent, featuredImagePath)
	} else {
		// If the image path is not a valid URL, fall back to text-only moderation
		response, err = s.openaiClient.ModerateContent(textContent)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to moderate content: %w", err)
	}

	// Check if content is flagged
	if response.IsFlagged() {
		// Get flagged categories
		categories := response.GetFlaggedCategories()
		reason := fmt.Sprintf("Content violates policy: %s", strings.Join(categories, ", "))

		return &ModerationResult{
			Approved: false,
			Reason:   reason,
		}, nil
	}

	// Content is approved
	return &ModerationResult{
		Approved: true,
		Reason:   "",
	}, nil
}

// CheckContent is a generic function to check any content
func (s *Service) CheckContent(content string) (*ModerationResult, error) {
	// Send content to OpenAI moderation API
	response, err := s.openaiClient.ModerateContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to moderate content: %w", err)
	}

	// Check if content is flagged
	if response.IsFlagged() {
		// Get flagged categories
		categories := response.GetFlaggedCategories()
		reason := fmt.Sprintf("Content violates policy: %s", strings.Join(categories, ", "))

		return &ModerationResult{
			Approved: false,
			Reason:   reason,
		}, nil
	}

	// Content is approved
	return &ModerationResult{
		Approved: true,
		Reason:   "",
	}, nil
}

// CheckSchematicQuality checks if a schematic is low-effort spam or an actual schematic
func (s *Service) CheckSchematicQuality(title, description string) (*ModerationResult, error) {
	// Log that we're checking the schematic quality
	if s.logger != nil {
		s.logger.Debug("Checking schematic quality", "title", title)
	}

	// Send the request to OpenAI
	isValid, reason, err := s.openaiClient.CheckSchematicQuality(title, description)
	if err != nil {
		return nil, fmt.Errorf("failed to check schematic quality: %w", err)
	}

	// Log the result
	if s.logger != nil {
		if isValid {
			s.logger.Debug("Schematic quality check passed", "title", title)
		} else {
			s.logger.Debug("Schematic quality check failed", "title", title, "reason", reason)
		}
	}

	// Return the result
	return &ModerationResult{
		Approved: isValid,
		Reason:   reason,
	}, nil
}

// isValidURL checks if the provided string is a valid URL and if it resolves
// by making a HEAD request without downloading the full content
func (s *Service) isValidURL(urlString string) bool {
	if urlString == "" {
		if s.logger != nil {
			s.logger.Debug("URL validation failed: empty URL string")
		}
		return false
	}

	// Parse the URL to check if it's syntactically valid
	u, err := url.Parse(urlString)
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("URL validation failed: invalid URL format", "url", urlString, "error", err.Error())
		}
		return false
	}

	// Check if the URL has a scheme and host
	if u.Scheme == "" || u.Host == "" {
		if s.logger != nil {
			s.logger.Debug("URL validation failed: missing scheme or host", "url", urlString)
		}
		return false
	}

	// Create an HTTP client with a reasonable timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create a HEAD request to check if the URL resolves without downloading the content
	req, err := http.NewRequest(http.MethodHead, urlString, nil)
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("URL validation failed: could not create request", "url", urlString, "error", err.Error())
		}
		return false
	}

	// Add a user agent to be more respectful to servers
	req.Header.Set("User-Agent", "CreateMod-Validator/1.0")

	// Send the request
	if s.logger != nil {
		s.logger.Debug("Validating URL with HEAD request", "url", urlString)
	}

	resp, err := client.Do(req)
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("URL validation failed: request error", "url", urlString, "error", err.Error())
		}
		return false
	}
	defer resp.Body.Close()

	// Check if the response status code indicates success (2xx)
	isValid := resp.StatusCode >= 200 && resp.StatusCode < 300

	if s.logger != nil {
		if isValid {
			s.logger.Debug("URL validation succeeded", "url", urlString, "status", resp.StatusCode)
		} else {
			s.logger.Debug("URL validation failed: non-success status code", "url", urlString, "status", resp.StatusCode)
		}
	}

	return isValid
}
