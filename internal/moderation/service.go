package moderation

import (
	"createmod/internal/openai"
	"createmod/internal/slowlog"
	"encoding/base64"
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

	if !response.IsFlagged() {
		return &ModerationResult{Approved: true}, nil
	}

	// Allow violence for schematics: Create builds are frequently weapons and
	// military hardware (TNT cannons, tanks, nukes) that the classifier reports
	// as violence. Only non-violence categories can hold a schematic, matching
	// the deliberate violence-allow already used for user images. (#1646)
	categories := blockingSchematicCategories(response.GetFlaggedCategories())
	if len(categories) == 0 {
		return &ModerationResult{Approved: true}, nil
	}

	// Context-aware second pass to clear figurative/gaming-slang false positives
	// (the same Minecraft-aware review used for comments). On error it upholds
	// the flag so a human still reviews it.
	uphold, reviewErr := s.openaiClient.ReviewModerationFlag(textContent, categories)
	if reviewErr != nil && s.logger != nil {
		s.logger.Warn("schematic moderation second-pass review failed, upholding flag",
			"error", reviewErr, "categories", strings.Join(categories, ", "))
	}
	if !uphold {
		if s.logger != nil {
			s.logger.Debug("schematic moderation flag cleared by second-pass review",
				"categories", strings.Join(categories, ", "))
		}
		return &ModerationResult{Approved: true}, nil
	}

	reason := fmt.Sprintf("Content violates policy: %s", strings.Join(categories, ", "))
	return &ModerationResult{Approved: false, Reason: reason}, nil
}

// blockingSchematicCategories drops violence categories from a flagged-category
// list. Create builds are legitimately weapons/military, so violence alone must
// not hold a schematic; the remaining categories are the ones that can. (#1646)
func blockingSchematicCategories(categories []string) []string {
	out := make([]string, 0, len(categories))
	for _, c := range categories {
		switch c {
		case "violence", "graphic violence":
			continue
		}
		out = append(out, c)
	}
	return out
}

// CheckContent is a generic function to check any text content. When the
// OpenAI Moderation API flags the text, a context-aware second pass
// (ReviewModerationFlag) re-evaluates it to clear common false positives such
// as gaming slang ("this build is the bomb"). The moderation endpoint itself
// has no prompt, so this second pass is where nuance is applied.
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

		// Second-pass, context-aware review to reduce false positives. On
		// error this returns true (uphold), so a human still reviews it.
		uphold, reviewErr := s.openaiClient.ReviewModerationFlag(content, categories)
		if reviewErr != nil && s.logger != nil {
			s.logger.Warn("moderation second-pass review failed, upholding flag",
				"error", reviewErr, "categories", strings.Join(categories, ", "))
		}
		if !uphold {
			if s.logger != nil {
				s.logger.Debug("moderation flag cleared by second-pass review",
					"categories", strings.Join(categories, ", "))
			}
			return &ModerationResult{Approved: true}, nil
		}

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

// CheckSchematicQuality checks if a schematic is low-effort spam or an actual
// schematic. blocksSummary, when non-empty, is a compact air-excluded list of
// the structure's blocks, passed as positive evidence of a genuine build so a
// real build with a thin description isn't mistaken for spam. Pass "" when the
// block palette is unavailable.
func (s *Service) CheckSchematicQuality(title, description, blocksSummary string) (*ModerationResult, error) {
	// Log that we're checking the schematic quality
	if s.logger != nil {
		s.logger.Debug("Checking schematic quality", "title", title)
	}

	// Send the request to OpenAI
	isValid, reason, err := s.openaiClient.CheckSchematicQuality(title, description, blocksSummary)
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

// CheckImage runs the OpenAI omni-moderation model against an image URL to
// detect policy-violating content (NSFW, violence, etc.). It does NOT check
// whether the image depicts a Minecraft build – use CheckImageQuality for that.
func (s *Service) CheckImage(imageURL string) (*ModerationResult, error) {
	if !s.isValidURL(imageURL) {
		if s.logger != nil {
			s.logger.Debug("Skipping image moderation check - invalid URL", "url", imageURL)
		}
		return &ModerationResult{Approved: true}, nil
	}
	if s.logger != nil {
		s.logger.Debug("Running image moderation check", "url", imageURL)
	}
	response, err := s.openaiClient.ModerateTextAndImage("", imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to moderate image: %w", err)
	}
	if response.IsFlagged() {
		categories := response.GetFlaggedCategories()
		reason := fmt.Sprintf("Image violates policy: %s", strings.Join(categories, ", "))
		if s.logger != nil {
			s.logger.Debug("Image moderation check failed", "url", imageURL, "reason", reason)
		}
		return &ModerationResult{Approved: false, Reason: reason}, nil
	}
	if s.logger != nil {
		s.logger.Debug("Image moderation check passed", "url", imageURL)
	}
	return &ModerationResult{Approved: true}, nil
}

// CheckUserImageContent moderates a user-uploaded image (raw bytes) with a
// Minecraft-aware policy: it flags nudity/sexual, hate, harassment and self-harm
// content but deliberately ALLOWS violence (violence, violence/graphic), since
// Create-mod builds are frequently weapons and military vehicles — tanks,
// cannons, nukes — which the model reports as violent. It moderates the bytes as
// a base64 data URI (no public fetch), so a temp image can be checked while it
// is still gated from public serving. (#1646)
func (s *Service) CheckUserImageContent(imageData []byte, mimeType string) (*ModerationResult, error) {
	if len(imageData) == 0 {
		return &ModerationResult{Approved: true}, nil
	}
	if mimeType == "" {
		mimeType = "image/webp"
	}
	dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imageData)
	response, err := s.openaiClient.ModerateTextAndImage("", dataURI)
	if err != nil {
		return nil, fmt.Errorf("failed to moderate image: %w", err)
	}
	if len(response.Results) == 0 {
		return &ModerationResult{Approved: true}, nil
	}
	c := response.Results[0].Categories
	var flagged []string
	if c.Sexual {
		flagged = append(flagged, "sexual content")
	}
	if c.SexualMinors {
		flagged = append(flagged, "sexual content involving minors")
	}
	if c.Hate || c.HateThreatening {
		flagged = append(flagged, "hate")
	}
	if c.Harassment || c.HarassmentThreatening {
		flagged = append(flagged, "harassment")
	}
	if c.SelfHarm || c.SelfHarmIntent || c.SelfHarmInstructions {
		flagged = append(flagged, "self-harm")
	}
	// Violence and violence/graphic are intentionally NOT flagged.
	if len(flagged) > 0 {
		reason := "Image violates policy: " + strings.Join(flagged, ", ")
		if s.logger != nil {
			s.logger.Debug("User image content check failed", "reason", reason)
		}
		return &ModerationResult{Approved: false, Reason: reason}, nil
	}
	return &ModerationResult{Approved: true}, nil
}

// CheckImageQuality checks if an image shows an actual Minecraft build
func (s *Service) CheckImageQuality(featuredImagePath string) (*ModerationResult, error) {
	// Check if the featured image path is a valid URL
	if !s.isValidURL(featuredImagePath) {
		// If the image path is not a valid URL, we can't check it
		if s.logger != nil {
			s.logger.Debug("Skipping image quality check - invalid URL", "path", featuredImagePath)
		}
		return &ModerationResult{
			Approved: true,
			Reason:   "",
		}, nil
	}

	// Log that we're checking the image quality
	if s.logger != nil {
		s.logger.Debug("Checking image quality for Minecraft build", "url", featuredImagePath)
	}

	// Send the request to OpenAI
	isValid, reason, err := s.openaiClient.CheckMinecraftBuildImage(featuredImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check image quality: %w", err)
	}

	// Log the result
	if s.logger != nil {
		if isValid {
			s.logger.Debug("Image quality check passed")
		} else {
			s.logger.Debug("Image quality check failed", "reason", reason)
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
		Timeout:   5 * time.Second,
		Transport: &slowlog.SlowHTTPTransport{Base: http.DefaultTransport, Subsystem: "moderation"},
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
