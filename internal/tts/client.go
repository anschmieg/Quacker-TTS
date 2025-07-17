package tts

import (
	"context"
	"fmt"
	"net/http"
	// Corrected import path
)

const (
	openAIAPIURL = "https://api.openai.com/v1/audio/speech"
)

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

// Client wraps OpenAIProvider for backward compatibility.
type Client struct {
	*OpenAIProvider
}

// NewClient creates a new OpenAI TTS client (backward compatibility).
func NewClient(apiKey string) *Client {
	return &Client{OpenAIProvider: NewOpenAIProvider(apiKey)}
}

// Request represents the parameters for a TTS request (backward compatibility).
type Request struct {
	Model          string  `json:"model"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	Input          string  `json:"input"`
	ResponseFormat string  `json:"response_format"`
	// Instructions are not directly part of the API payload for TTS,
	// but might be used for future prompt engineering if needed.
	// Instructions string
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

// GenerateSpeech generates speech using the unified request format.
func (p *OpenAIProvider) GenerateSpeech(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	// Convert unified request to OpenAI format
	openAIReq := Request{
		Model:          req.Model,
		Voice:          req.Voice,
		Speed:          req.Speed,
		Input:          req.Text,
		ResponseFormat: req.Format,
	}

	// Set defaults if not provided
	if openAIReq.Model == "" {
		openAIReq.Model = "gpt-4o-mini-tts"
	}
	if openAIReq.ResponseFormat == "" {
		openAIReq.ResponseFormat = "mp3"
	}

	return p.generateSpeechInternal(openAIReq)
}

// GenerateSpeech sends a request to the OpenAI TTS API and returns the audio data (backward compatibility).
func (c *Client) GenerateSpeech(reqData Request) ([]byte, error) {
	return c.OpenAIProvider.generateSpeechInternal(reqData)
}
