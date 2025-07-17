package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"easy-tts/internal/util"
)

// generateSpeechInternal is the internal implementation for generating speech.
func (p *OpenAIProvider) generateSpeechInternal(reqData Request) ([]byte, error) {
	// always split large input into chunks (splitText handles encoding fallbacks)
	parts := splitText(reqData.Input, reqData.Model, DefaultTokenLimit)
	if len(parts) > 1 {
		return p.GenerateSpeechChunks(reqData)
	}

	if p.APIKey == "" {
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
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(req)
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
