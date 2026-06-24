package openai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"createmod/internal/slowlog"
)

// Logger is an interface that defines the logging methods we need
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

const (
	// ModerationEndpoint is the OpenAI moderation API endpoint
	ModerationEndpoint = "https://api.openai.com/v1/moderations"
	// ChatCompletionEndpoint is the OpenAI chat completion API endpoint
	ChatCompletionEndpoint = "https://api.openai.com/v1/chat/completions"
	// ResponsesEndpoint is the OpenAI responses API endpoint for image analysis
	ResponsesEndpoint = "https://api.openai.com/v1/responses"
)

// Client represents an OpenAI API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	logger     Logger
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string, logger Logger) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: &slowlog.SlowHTTPTransport{Base: http.DefaultTransport, Subsystem: "openai"},
		},
		logger: logger,
	}
}

// ImageInput represents an image input for moderation
type ImageInput struct {
	Type     string   `json:"type"`
	ImageURL ImageURL `json:"image_url"`
}

// ImageURL contains either a URL of the image or base64 encoded image data
type ImageURL struct {
	URL string `json:"url"`
}

// TextInput represents a text input for moderation
type TextInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ModerationResponse represents a response from the moderation API
type ModerationResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Results []ModerationResult `json:"results"`
}

// ModerationResult represents a single result from the moderation API
type ModerationResult struct {
	Flagged        bool                     `json:"flagged"`
	Categories     ModerationCategories     `json:"categories"`
	CategoryScores ModerationCategoryScores `json:"category_scores"`
}

// ModerationCategories represents the categories that can be flagged
type ModerationCategories struct {
	Sexual                bool `json:"sexual"`
	Hate                  bool `json:"hate"`
	Harassment            bool `json:"harassment"`
	SelfHarm              bool `json:"self-harm"`
	SexualMinors          bool `json:"sexual/minors"`
	HateThreatening       bool `json:"hate/threatening"`
	ViolenceGraphic       bool `json:"violence/graphic"`
	SelfHarmIntent        bool `json:"self-harm/intent"`
	SelfHarmInstructions  bool `json:"self-harm/instructions"`
	HarassmentThreatening bool `json:"harassment/threatening"`
	Violence              bool `json:"violence"`
}

// ModerationCategoryScores represents the scores for each category
type ModerationCategoryScores struct {
	Sexual                float64 `json:"sexual"`
	Hate                  float64 `json:"hate"`
	Harassment            float64 `json:"harassment"`
	SelfHarm              float64 `json:"self-harm"`
	SexualMinors          float64 `json:"sexual/minors"`
	HateThreatening       float64 `json:"hate/threatening"`
	ViolenceGraphic       float64 `json:"violence/graphic"`
	SelfHarmIntent        float64 `json:"self-harm/intent"`
	SelfHarmInstructions  float64 `json:"self-harm/instructions"`
	HarassmentThreatening float64 `json:"harassment/threatening"`
	Violence              float64 `json:"violence"`
}

// ModerateContent sends a request to the OpenAI moderation API to check if content violates policies
// This is a backward-compatible method that accepts a string input
func (c *Client) ModerateContent(content string) (*ModerationResponse, error) {
	textInput := TextInput{
		Type: "text",
		Text: content,
	}

	request := ModerationRequest{
		Model: "omni-moderation-latest",
		Input: []interface{}{textInput},
	}

	return c.sendModerationRequest(request)
}

// ModerateContentArray sends a request to the OpenAI moderation API with an array of strings
func (c *Client) ModerateContentArray(contents []string) (*ModerationResponse, error) {
	var inputs []interface{}

	for _, content := range contents {
		textInput := TextInput{
			Type: "text",
			Text: content,
		}
		inputs = append(inputs, textInput)
	}

	request := ModerationRequest{
		Model: "omni-moderation-latest",
		Input: inputs,
	}

	return c.sendModerationRequest(request)
}

// ModerationRequest represents a request to the moderation API
type ModerationRequest struct {
	Model string        `json:"model"`
	Input []interface{} `json:"input"`
}

// ModerateTextAndImage sends a request to the OpenAI moderation API with text and image inputs
func (c *Client) ModerateTextAndImage(text, imageURL string) (*ModerationResponse, error) {
	textInput := TextInput{
		Type: "text",
		Text: text,
	}

	imageInput := ImageInput{
		Type: "image_url",
		ImageURL: ImageURL{
			URL: imageURL,
		},
	}

	input := ModerationRequest{
		Model: "omni-moderation-latest",
		Input: []interface{}{textInput, imageInput},
	}

	return c.sendModerationRequest(input)
}

// sendModerationRequest handles sending the request to the OpenAI API with any supported input type
func (c *Client) sendModerationRequest(input interface{}) (*ModerationResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request at info level, but don't log potentially large image data
	if c.logger != nil {
		// Create a sanitized version of the request for logging
		var sanitizedInput interface{}
		if err := json.Unmarshal(jsonData, &sanitizedInput); err == nil {
			// For moderation requests, we can't easily sanitize the input without knowing its exact structure,
			// so we'll just log that a request was made and its size
			c.logger.Info("OpenAI moderation request", "endpoint", ModerationEndpoint, "request_body_size", len(jsonData))
		} else {
			// Fallback if unmarshaling fails
			c.logger.Info("OpenAI moderation request", "endpoint", ModerationEndpoint, "request_body_error", "unable to sanitize request body")
		}
	}

	req, err := http.NewRequest("POST", ModerationEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI moderation request", "error", err.Error())
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI moderation response", "error", err.Error())
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response at info level
	if c.logger != nil {
		// For successful responses, log only the status code and not the potentially large response body
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI moderation response", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			// For error responses, log the full response as it's useful for debugging
			c.logger.Info("OpenAI moderation response", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			return nil, fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}

	var moderationResponse ModerationResponse
	if err := json.Unmarshal(respBody, &moderationResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &moderationResponse, nil
}

// IsFlagged checks if any of the results in the moderation response are flagged
func (r *ModerationResponse) IsFlagged() bool {
	for _, result := range r.Results {
		if result.Flagged {
			return true
		}
	}
	return false
}

// GetFlaggedCategories returns a list of flagged categories from the moderation response
func (r *ModerationResponse) GetFlaggedCategories() []string {
	if len(r.Results) == 0 {
		return nil
	}

	var flaggedCategories []string
	categories := r.Results[0].Categories

	if categories.Sexual {
		flaggedCategories = append(flaggedCategories, "sexual content")
	}
	if categories.Hate {
		flaggedCategories = append(flaggedCategories, "hate speech")
	}
	if categories.Harassment {
		flaggedCategories = append(flaggedCategories, "harassment")
	}
	if categories.SelfHarm {
		flaggedCategories = append(flaggedCategories, "self-harm")
	}
	if categories.SexualMinors {
		flaggedCategories = append(flaggedCategories, "sexual content involving minors")
	}
	if categories.HateThreatening {
		flaggedCategories = append(flaggedCategories, "threatening hate speech")
	}
	if categories.ViolenceGraphic {
		flaggedCategories = append(flaggedCategories, "graphic violence")
	}
	if categories.SelfHarmIntent {
		flaggedCategories = append(flaggedCategories, "self-harm intent")
	}
	if categories.SelfHarmInstructions {
		flaggedCategories = append(flaggedCategories, "self-harm instructions")
	}
	if categories.HarassmentThreatening {
		flaggedCategories = append(flaggedCategories, "threatening harassment")
	}
	if categories.Violence {
		flaggedCategories = append(flaggedCategories, "violence")
	}

	return flaggedCategories
}

// ChatMessage represents a message in a chat completion request
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatCompletionResponse represents a response from the chat completion API
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ResponseContent represents the content of a response message
type ResponseContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// ResponseMessage represents a message in a response request
type ResponseMessage struct {
	Role    string            `json:"role"`
	Content []ResponseContent `json:"content"`
}

// ResponseRequest represents a request to the responses API
type ResponseRequest struct {
	Model string            `json:"model"`
	Input []ResponseMessage `json:"input"`
}

// ResponseOutputContent represents the content of a response output message
type ResponseOutputContent struct {
	Type        string        `json:"type"`
	Annotations []interface{} `json:"annotations"`
	Logprobs    []interface{} `json:"logprobs"`
	Text        string        `json:"text"`
}

// ResponseOutputMessage represents a message in a response output
type ResponseOutputMessage struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Status  string                  `json:"status"`
	Content []ResponseOutputContent `json:"content"`
	Role    string                  `json:"role"`
}

// ResponseResponse represents a response from the responses API
type ResponseResponse struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created_at"`
	Status  string                  `json:"status"`
	Model   string                  `json:"model"`
	Output  []ResponseOutputMessage `json:"output"`
}

// EncodeImageToBase64 encodes an image file to base64
func (c *Client) EncodeImageToBase64(imagePath string) (string, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
}

// GenerateImageDescription generates a description for an image using the OpenAI API
func (c *Client) GenerateImageDescription(imageURL string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is required")
	}

	// Create the request
	request := ResponseRequest{
		Model: "gpt-4.1",
		Input: []ResponseMessage{
			{
				Role: "user",
				Content: []ResponseContent{
					{
						Type: "input_text",
						Text: "A user uploaded this image to represent a create mod schematic file. Your job is to make a very short description that will be used for site search.",
					},
					{
						Type:     "input_image",
						ImageURL: imageURL,
					},
				},
			},
		},
	}

	// Send the request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a sanitized version of the request for logging (without base64 image data)
	var sanitizedRequest ResponseRequest
	if err := json.Unmarshal(jsonData, &sanitizedRequest); err == nil {
		// Sanitize the request by replacing base64 image data with a placeholder
		for i := range sanitizedRequest.Input {
			for j := range sanitizedRequest.Input[i].Content {
				if sanitizedRequest.Input[i].Content[j].Type == "input_image" &&
					strings.HasPrefix(sanitizedRequest.Input[i].Content[j].ImageURL, "data:image") {
					sanitizedRequest.Input[i].Content[j].ImageURL = "[BASE64_IMAGE_DATA_REDACTED]"
				}
			}
		}

		// Marshal the sanitized request
		sanitizedJSON, err := json.Marshal(sanitizedRequest)
		if err == nil && c.logger != nil {
			c.logger.Info("OpenAI responses request", "endpoint", ResponsesEndpoint, "request_body", string(sanitizedJSON))
		}
	} else if c.logger != nil {
		// Fallback if unmarshaling fails
		c.logger.Info("OpenAI responses request", "endpoint", ResponsesEndpoint, "request_body_error", "unable to sanitize request body")
	}

	req, err := http.NewRequest("POST", ResponsesEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI responses request", "error", err.Error())
		}
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI responses response", "error", err.Error())
		}
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response at info level
	if c.logger != nil {
		// For successful responses, log only the status code and not the potentially large response body
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI responses response", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			// For error responses, log the full response as it's useful for debugging
			c.logger.Info("OpenAI responses response", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			return "", fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
		}
		return "", fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}

	var responseResponse ResponseResponse
	if err := json.Unmarshal(respBody, &responseResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract the description from the response
	if len(responseResponse.Output) == 0 {
		return "", fmt.Errorf("no output in OpenAI response")
	}

	// Get the response text from the first message
	for _, message := range responseResponse.Output {
		if message.Role == "assistant" && len(message.Content) > 0 {
			for _, content := range message.Content {
				if content.Type == "output_text" {
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no text content in OpenAI response")
}

// GenerateImageDescriptionFromBase64 generates a description for a base64-encoded image using the OpenAI API
func (c *Client) GenerateImageDescriptionFromBase64(base64Image string) (string, error) {
	// Format the base64 image as a data URL
	imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
	return c.GenerateImageDescription(imageURL)
}

// GenerateImageDescriptionFromFile generates a description for an image file using the OpenAI API
func (c *Client) GenerateImageDescriptionFromFile(imagePath string) (string, error) {
	base64Image, err := c.EncodeImageToBase64(imagePath)
	if err != nil {
		return "", err
	}
	return c.GenerateImageDescriptionFromBase64(base64Image)
}

// CheckSchematicQuality sends a request to the OpenAI chat completion API to check if a schematic is low-effort spam
func (c *Client) CheckSchematicQuality(title, description string) (bool, string, error) {
	if c.apiKey == "" {
		return false, "", fmt.Errorf("OpenAI API key is required")
	}

	// Format the prompt as specified
	prompt := fmt.Sprintf("Title: %s\nDescription: %s\n\nThis text accompanies a shared Minecraft Create-mod schematic. Approve it unless it is obvious spam such as gibberish, advertising, links to unrelated sites, or clearly abusive content. A short, simple, or low-detail title and description is fine and must be approved — builders often write very little. Respond with exactly 'true' if it should be approved, otherwise a brief reason why it is spam.", title, description)

	// Create the request
	request := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a lenient content classifier for CreateMod.com, a Minecraft Create-mod schematic community. You decide only whether a submission is obvious low-effort spam, not whether it is high quality. Err strongly on the side of approval.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Send the request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request at info level
	if c.logger != nil {
		// For chat completion requests, we can log the full request as it doesn't contain large binary data
		// But we'll still follow the same pattern for consistency
		c.logger.Info("OpenAI chat completion request", "endpoint", ChatCompletionEndpoint, "request_body_size", len(jsonData))
	}

	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI chat completion request", "error", err.Error())
		}
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI chat completion response", "error", err.Error())
		}
		return false, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response at info level
	if c.logger != nil {
		// For successful responses, log only the status code and not the potentially large response body
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI chat completion response", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			// For error responses, log the full response as it's useful for debugging
			c.logger.Info("OpenAI chat completion response", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			return false, "", fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
		}
		return false, "", fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}

	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return false, "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if there are any choices in the response
	if len(completionResponse.Choices) == 0 {
		return false, "", fmt.Errorf("no choices in OpenAI response")
	}

	// Get the response content
	responseContent := completionResponse.Choices[0].Message.Content
	responseContent = strings.TrimSpace(responseContent)

	// Check for truthy responses robustly
	if isAffirmativeTrue(responseContent) {
		return true, "", nil
	}

	// Otherwise, return the reason
	return false, responseContent, nil
}

func (c *Client) HasApiKey() bool {
	return c.apiKey != ""
}

// TranslateToEnglish uses Chat Completions to translate arbitrary text to natural English.
// It returns the original text if the API key is missing or an error occurs.
func (c *Client) TranslateToEnglish(text string) (string, error) {
	if c.apiKey == "" {
		return text, fmt.Errorf("OpenAI API key is required")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text, nil
	}
	prompt := "Translate the following text to natural English suitable for a website.\nReturn only the translated English text without quotes or extra commentary.\n\n" + trimmed
	request := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a professional translator that outputs only natural English translations."},
			{Role: "user", Content: prompt},
		},
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return text, fmt.Errorf("failed to marshal request: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("OpenAI chat completion request (translate)", "endpoint", ChatCompletionEndpoint, "request_body_size", len(jsonData))
	}
	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return text, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI chat completion request (translate)", "error", err.Error())
		}
		return text, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI chat completion response (translate)", "error", err.Error())
		}
		return text, fmt.Errorf("failed to read response body: %w", err)
	}
	if c.logger != nil {
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI chat completion response (translate)", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			c.logger.Info("OpenAI chat completion response (translate)", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}
	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}
	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return text, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return text, fmt.Errorf("no choices in OpenAI response")
	}
	out := strings.TrimSpace(completionResponse.Choices[0].Message.Content)
	if out == "" {
		return text, fmt.Errorf("empty translation")
	}
	return out, nil
}

// TranslateText uses Chat Completions to translate arbitrary text to the given target language.
// It returns the original text if the API key is missing or an error occurs.
func (c *Client) TranslateText(text, targetLang string) (string, error) {
	if c.apiKey == "" {
		return text, fmt.Errorf("OpenAI API key is required")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text, nil
	}
	prompt := fmt.Sprintf("Translate the following text to %s suitable for a website.\nReturn only the translated text without quotes or extra commentary.\n\n%s", targetLang, trimmed)
	request := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: fmt.Sprintf("You are a professional translator that outputs only natural %s translations.", targetLang)},
			{Role: "user", Content: prompt},
		},
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return text, fmt.Errorf("failed to marshal request: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("OpenAI chat completion request (translate)", "endpoint", ChatCompletionEndpoint, "target_lang", targetLang, "request_body_size", len(jsonData))
	}
	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return text, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI chat completion request (translate)", "error", err.Error())
		}
		return text, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI chat completion response (translate)", "error", err.Error())
		}
		return text, fmt.Errorf("failed to read response body: %w", err)
	}
	if c.logger != nil {
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI chat completion response (translate)", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			c.logger.Info("OpenAI chat completion response (translate)", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}
	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}
	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return text, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return text, fmt.Errorf("no choices in OpenAI response")
	}
	out := strings.TrimSpace(completionResponse.Choices[0].Message.Content)
	if out == "" {
		return text, fmt.Errorf("empty translation")
	}
	return out, nil
}

// TranslatedFields holds the result of a multi-field translation.
type TranslatedFields struct {
	Title       string
	Description string
	Content     string
}

// TranslateFields translates title, description, and content in a single API call.
// Empty input fields are returned as empty strings without being sent to the API.
func (c *Client) TranslateFields(title, description, content, targetLang string) (*TranslatedFields, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	content = strings.TrimSpace(content)
	if title == "" && description == "" && content == "" {
		return &TranslatedFields{}, nil
	}

	// Build the input with delimiters
	var sb strings.Builder
	sb.WriteString("Translate the following fields to " + targetLang + ". ")
	sb.WriteString("Return each translated field wrapped in exactly the same tags. ")
	sb.WriteString("If a field is empty, return it empty. Return only the tags and translated text, nothing else.\n\n")
	sb.WriteString("[TITLE]" + title + "[/TITLE]\n")
	sb.WriteString("[DESCRIPTION]" + description + "[/DESCRIPTION]\n")
	sb.WriteString("[CONTENT]" + content + "[/CONTENT]")

	request := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: fmt.Sprintf("You are a professional translator. Output only the translated fields wrapped in [TITLE][/TITLE], [DESCRIPTION][/DESCRIPTION], and [CONTENT][/CONTENT] tags. Produce natural %s translations suitable for a website.", targetLang)},
			{Role: "user", Content: sb.String()},
		},
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("OpenAI chat completion request (translateFields)", "endpoint", ChatCompletionEndpoint, "target_lang", targetLang, "request_body_size", len(jsonData))
	}
	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI request (translateFields)", "error", err.Error())
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if c.logger != nil {
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI response (translateFields)", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			c.logger.Info("OpenAI response (translateFields)", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}
	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}
	raw := completionResponse.Choices[0].Message.Content

	// Parse tagged fields from the response
	extract := func(tag, s string) string {
		open := "[" + tag + "]"
		close := "[/" + tag + "]"
		start := strings.Index(s, open)
		end := strings.Index(s, close)
		if start == -1 || end == -1 || end <= start {
			return ""
		}
		return strings.TrimSpace(s[start+len(open) : end])
	}

	return &TranslatedFields{
		Title:       extract("TITLE", raw),
		Description: extract("DESCRIPTION", raw),
		Content:     extract("CONTENT", raw),
	}, nil
}

// DetectLanguage asks the model to identify which supported language the text is
// written in. It returns one of the supported language codes (en, fr, pt-BR,
// pt-PT, es, de, pl, ru, zh-Hans). On failure or for English text it returns "en".
func (c *Client) DetectLanguage(text string) (string, error) {
	if c.apiKey == "" {
		return "en", fmt.Errorf("OpenAI API key is required")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "en", nil
	}

	prompt := "Identify the language of the following text. Return exactly one of these codes: en, fr, pt-BR, pt-PT, es, de, pl, ru, zh-Hans\nReturn only the code, nothing else.\n\n" + trimmed
	request := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a language detection assistant. You return only a single language code from the allowed list."},
			{Role: "user", Content: prompt},
		},
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "en", fmt.Errorf("failed to marshal request: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("OpenAI chat completion request (detectLanguage)", "endpoint", ChatCompletionEndpoint, "request_body_size", len(jsonData))
	}
	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "en", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI chat completion request (detectLanguage)", "error", err.Error())
		}
		return "en", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "en", fmt.Errorf("failed to read response body: %w", err)
	}
	if c.logger != nil {
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI chat completion response (detectLanguage)", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			c.logger.Info("OpenAI chat completion response (detectLanguage)", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}
	if resp.StatusCode != http.StatusOK {
		return "en", fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}
	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return "en", fmt.Errorf("failed to decode response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return "en", fmt.Errorf("no choices in OpenAI response")
	}
	code := strings.TrimSpace(completionResponse.Choices[0].Message.Content)
	// Validate the returned code is one of our supported languages
	supported := map[string]bool{
		"en": true, "fr": true, "pt-BR": true, "pt-PT": true,
		"es": true, "de": true, "pl": true, "ru": true, "zh-Hans": true,
	}
	if !supported[code] {
		return "en", nil
	}
	return code, nil
}

// CheckMinecraftBuildImage checks if an image shows an actual Minecraft build
// Returns true if it's a valid build, false with a reason if it's spam/random blocks
func (c *Client) CheckMinecraftBuildImage(imageURL string) (bool, string, error) {
	if c.apiKey == "" {
		return false, "", fmt.Errorf("OpenAI API key is required")
	}

	// Log that we're checking the image
	if c.logger != nil {
		c.logger.Debug("Checking if image shows a Minecraft build")
	}

	// Create the request using chat completion with vision
	request := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a careful content classifier for CreateMod.com, a community where players share Minecraft 'Create' mod schematics and builds. You decide whether an uploaded image plausibly belongs on the site. Err strongly on the side of approval: only reject images that are clearly unrelated to Minecraft or the Create mod.",
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "This image was uploaded to represent a Minecraft Create-mod schematic. Approve it if it plausibly relates to Minecraft or the Create mod in ANY form, including: in-game builds, structures, machines, contraptions, trains, factories and redstone; close-ups or wide shots; in-progress or unfinished builds; schematic holograms, previews or renders; and IN-GAME USER INTERFACE OR GUI SCREENSHOTS such as inventories, the Schematicannon, the schematic table, Ponder, JEI, or config and menu screens. UI and GUI screenshots from the game or its mods are valid and must be approved. Only reject if the image is clearly NOT related to Minecraft or the Create mod, for example a real-life photo, an unrelated anime or cartoon character, a meme, or an advertisement. Respond with exactly 'true' if it should be approved, otherwise a brief reason (a few words) why it is unrelated to Minecraft or the Create mod.",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": imageURL,
						},
					},
				},
			},
		},
	}

	// Send the request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request at info level (without image data)
	if c.logger != nil {
		c.logger.Info("OpenAI Minecraft build image check request", "endpoint", ChatCompletionEndpoint, "request_body_size", len(jsonData))
	}

	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI Minecraft build image check request", "error", err.Error())
		}
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read OpenAI Minecraft build image check response", "error", err.Error())
		}
		return false, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response
	if c.logger != nil {
		if resp.StatusCode == http.StatusOK {
			c.logger.Info("OpenAI Minecraft build image check response", "status_code", resp.StatusCode, "response_body_size", len(respBody))
		} else {
			c.logger.Info("OpenAI Minecraft build image check response", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			return false, "", fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
		}
		return false, "", fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}

	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return false, "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if there are any choices in the response
	if len(completionResponse.Choices) == 0 {
		return false, "", fmt.Errorf("no choices in OpenAI response")
	}

	// Get the response content
	responseContent := completionResponse.Choices[0].Message.Content
	responseContent = strings.TrimSpace(responseContent)

	// Check for truthy responses robustly
	if isAffirmativeTrue(responseContent) {
		if c.logger != nil {
			c.logger.Debug("Minecraft build image check passed")
		}
		return true, "", nil
	}

	// Otherwise, return the reason
	if c.logger != nil {
		c.logger.Debug("Minecraft build image check failed", "reason", responseContent)
	}
	return false, responseContent, nil
}

// ModerationReviewSystemPrompt is the system prompt for the context-aware
// second-pass moderation review (ReviewModerationFlag). It is intentionally
// kept as an exported constant so it is easy to tune over time. Edit this to
// adjust how lenient the false-positive review is.
const ModerationReviewSystemPrompt = "You are a content-moderation reviewer for CreateMod.com, a friendly community where players share Minecraft 'Create' mod builds. An automated classifier has flagged a user's text. Decide whether it is a GENUINE violation of content policy (real sexual content, real hate speech, real harassment of a person or group, credible threats or incitement of real-world violence, or encouragement of self-harm) or a FALSE POSITIVE. Casual gaming and English slang is NOT a violation: figurative or enthusiastic phrases such as 'this is the bomb', 'this is fire', 'this build slaps', 'killer base', 'sick build', 'this blew my mind', 'I'm gonna destroy this boss', or 'this kills me' are positive or figurative and must be cleared. Hyperbole, excitement, and praise are allowed. Only uphold the flag when a reasonable human moderator would agree the text genuinely violates policy. Respond with exactly 'violation' to uphold the flag, or 'ok' to clear it."

// ReviewModerationFlag is a context-aware second pass that reduces false
// positives from the OpenAI Moderation API. The moderation endpoint is a fixed
// classifier with no prompt, so harmless gaming slang ("this build is the
// bomb", "this is fire", "killer base") is sometimes flagged for
// violence/harassment. This method re-reads the text in the context of a
// friendly Minecraft building community and decides whether the flag is a real
// policy violation.
//
// It returns true to UPHOLD the flag (treat as a genuine violation) and false
// to clear it (false positive). On a missing API key or any API error it
// returns true so the flag is preserved and a human reviews the content
// (fail safe).
func (c *Client) ReviewModerationFlag(content string, categories []string) (bool, error) {
	if c.apiKey == "" {
		return true, fmt.Errorf("OpenAI API key is required")
	}

	userPrompt := fmt.Sprintf("Flagged categories: %s\n\nText:\n%s\n\nIs this a genuine policy violation or a false positive? Respond with exactly 'violation' or 'ok'.", strings.Join(categories, ", "), content)

	request := ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []ChatMessage{
			{Role: "system", Content: ModerationReviewSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return true, fmt.Errorf("failed to marshal request: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("OpenAI moderation review request", "endpoint", ChatCompletionEndpoint, "categories", strings.Join(categories, ", "))
	}

	req, err := http.NewRequest("POST", ChatCompletionEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return true, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to send OpenAI moderation review request", "error", err.Error())
		}
		return true, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if c.logger != nil {
			c.logger.Info("OpenAI moderation review response", "status_code", resp.StatusCode, "response_body", string(respBody))
		}
		return true, fmt.Errorf("OpenAI API returned status code %d", resp.StatusCode)
	}

	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResponse); err != nil {
		return true, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return true, fmt.Errorf("no choices in OpenAI response")
	}

	return upholdFromReviewAnswer(completionResponse.Choices[0].Message.Content), nil
}

// upholdFromReviewAnswer interprets the second-pass reviewer's answer. The flag
// is cleared (returns false) only on an explicit "ok"; anything else — an
// explicit "violation", or any unexpected/empty answer — preserves the flag
// (returns true) so a human reviews the content.
func upholdFromReviewAnswer(raw string) bool {
	answer := strings.ToLower(strings.Trim(strings.TrimSpace(raw), "\"'`.! "))
	if answer == "ok" || strings.HasPrefix(answer, "ok ") || strings.HasPrefix(answer, "ok,") {
		return false
	}
	return true
}

// isAffirmativeTrue determines if the OpenAI response should be treated as approval (true)
// It handles variants like "True", "true.", and sentences that clearly include the token 'true'.
// It also guards against obvious negatives like "not true" or explicit "false".
func isAffirmativeTrue(s string) bool {
	if s == "" {
		return false
	}
	t := strings.ToLower(strings.TrimSpace(s))
	// quick exact and punctuation-trimmed checks
	t = strings.Trim(t, "\"'` ")
	trimmed := strings.TrimRight(t, ".!?) ")
	if trimmed == "true" {
		return true
	}
	// guard common negatives
	if strings.Contains(t, "not true") || strings.Contains(t, "false") {
		return false
	}
	// look for standalone word true
	re := regexp.MustCompile(`\btrue\b`)
	return re.FindStringIndex(t) != nil
}
