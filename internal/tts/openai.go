package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const openAIAPIURL = "https://api.openai.com/v1/audio/speech"

// OpenAIProvider handles communication with the OpenAI TTS API.
type OpenAIProvider struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI TTS provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// GetName returns the provider's name.
func (p *OpenAIProvider) GetName() string {
	return "openai"
}

// GetDefaultVoice returns the provider's default voice.
func (p *OpenAIProvider) GetDefaultVoice() string {
	return "shimmer"
}

// GetSupportedFormats returns the audio formats supported by this provider.
func (p *OpenAIProvider) GetSupportedFormats() []string {
	return []string{"mp3", "opus", "aac", "flac"}
}

// ValidateConfig validates the provider's configuration.
func (p *OpenAIProvider) ValidateConfig() error {
	if p.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	return nil
}

// GetMaxTokensPerChunk returns the maximum tokens per request for this provider.
func (p *OpenAIProvider) GetMaxTokensPerChunk() int {
	return DefaultTokenLimit
}

// CheckAuth verifies that the OpenAI API key is valid by making a lightweight request.
func (p *OpenAIProvider) CheckAuth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("OpenAI auth failed with status: %s", resp.Status)
}

// GenerateSpeech generates speech for a single, pre-chunked piece of text.
func (p *OpenAIProvider) GenerateSpeech(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("API key is not configured")
	}

	payload := map[string]any{
		"model":           req.Model,
		"voice":           req.Voice,
		"speed":           req.Speed,
		"input":           req.Text,
		"response_format": req.Format,
	}
	if payload["model"] == "" {
		payload["model"] = "gpt-4o-mini-tts"
	}
	if payload["response_format"] == "" {
		payload["response_format"] = "mp3"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("API error (status %d): %s", resp.StatusCode, resp.Status)
		if len(respBody) > 0 {
			var prettyJSON bytes.Buffer
			if json.Indent(&prettyJSON, respBody, "", "  ") == nil {
				errMsg += "\n" + prettyJSON.String()
			} else {
				errMsg += "\n" + string(respBody)
			}
		}
		return nil, fmt.Errorf(errMsg)
	}

	return respBody, nil
}
