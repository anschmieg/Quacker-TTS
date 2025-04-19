package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"easy-tts/internal/util" // Corrected import path
)

const (
	openAIAPIURL = "https://api.openai.com/v1/audio/speech"
)

// Client handles communication with the OpenAI TTS API.
type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new TTS client.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// Request represents the parameters for a TTS request.
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

// GenerateSpeech sends a request to the OpenAI TTS API and returns the audio data.
func (c *Client) GenerateSpeech(reqData Request) ([]byte, error) {
	// always split large input into chunks (splitText handles encoding fallbacks)
	parts := splitText(reqData.Input, reqData.Model, DefaultTokenLimit)
	if len(parts) > 1 {
		return c.GenerateSpeechChunks(reqData)
	}

	if c.APIKey == "" {
		return nil, fmt.Errorf("API key is not configured")
	}

	// Prepare payload - clean the input text for JSON
	payload := map[string]any{
		"model":           reqData.Model,
		"voice":           reqData.Voice,
		"speed":           reqData.Speed,
		"input":           util.CleanJSONString(reqData.Input),
		"response_format": reqData.ResponseFormat,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest("POST", openAIAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("API error (status %d):", resp.StatusCode)
		if len(respBody) > 0 {
			// Try to pretty-print JSON error, otherwise show raw body
			var prettyJSON bytes.Buffer
			if json.Indent(&prettyJSON, respBody, "", "  ") == nil {
				errMsg += "\n" + prettyJSON.String()
			} else {
				errMsg += "\n" + string(respBody)
			}
		} else {
			errMsg += " " + resp.Status
		}
		return nil, fmt.Errorf(errMsg)
	}

	return respBody, nil
}
