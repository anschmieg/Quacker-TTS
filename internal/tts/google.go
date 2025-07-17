package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"easy-tts/internal/util"
)

const (
	googleTTSAPIURL = "https://texttospeech.googleapis.com/v1/text:synthesize"
	// Google TTS has a limit of 5000 characters per request
	googleMaxCharsPerRequest = 5000
)

// GoogleProvider handles communication with the Google Cloud TTS API.
type GoogleProvider struct {
	ProjectID    string
	HTTPClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
	tokenMutex   sync.RWMutex
}

// NewGoogleProvider creates a new Google TTS provider.
func NewGoogleProvider(projectID string) *GoogleProvider {
	return &GoogleProvider{
		ProjectID:  projectID,
		HTTPClient: &http.Client{},
	}
}

// GetName returns the provider's name.
func (g *GoogleProvider) GetName() string {
	return "google"
}

// GetDefaultVoice returns the provider's default voice.
func (g *GoogleProvider) GetDefaultVoice() string {
	return "de-DE-Chirp3-HD-Kore"
}

// GetSupportedFormats returns the audio formats supported by this provider.
func (g *GoogleProvider) GetSupportedFormats() []string {
	return []string{"mp3", "linear16", "ogg-opus", "mulaw", "alaw"}
}

// ValidateConfig validates the provider's configuration.
func (g *GoogleProvider) ValidateConfig() error {
	if g.ProjectID == "" {
		return fmt.Errorf("Google Cloud project ID is required")
	}

	// Try to get access token to validate authentication
	_, err := g.getAccessToken()
	if err != nil {
		return fmt.Errorf("failed to authenticate with Google Cloud: %w", err)
	}

	return nil
}

// GetMaxTokensPerChunk returns the maximum tokens per request for this provider.
func (g *GoogleProvider) GetMaxTokensPerChunk() int {
	// Google TTS uses character limit, not token limit
	// We'll estimate roughly 3 characters per token for compatibility
	return googleMaxCharsPerRequest / 3
}

// getAccessToken retrieves an access token using gcloud auth.
func (g *GoogleProvider) getAccessToken() (string, error) {
	g.tokenMutex.Lock()
	defer g.tokenMutex.Unlock()

	// Check if we have a valid cached token
	if g.accessToken != "" && time.Now().Before(g.tokenExpiry) {
		return g.accessToken, nil
	}

	// Get new token using gcloud
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token from gcloud: %w. Make sure you're authenticated with 'gcloud auth login'", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("received empty access token from gcloud")
	}

	// Cache the token (Google Cloud tokens typically last 1 hour, we'll refresh after 50 minutes)
	g.accessToken = token
	g.tokenExpiry = time.Now().Add(50 * time.Minute)

	return token, nil
}

// googleTTSRequest represents the request structure for Google TTS API.
type googleTTSRequest struct {
	Input       googleTTSInput       `json:"input"`
	Voice       googleTTSVoice       `json:"voice"`
	AudioConfig googleTTSAudioConfig `json:"audioConfig"`
}

type googleTTSInput struct {
	Text string `json:"text"`
}

type googleTTSVoice struct {
	LanguageCode string                 `json:"languageCode"`
	Name         string                 `json:"name"`
	VoiceClone   map[string]interface{} `json:"voiceClone,omitempty"`
}

type googleTTSAudioConfig struct {
	AudioEncoding string  `json:"audioEncoding"`
	SpeakingRate  float64 `json:"speakingRate"`
}

// googleTTSResponse represents the response structure from Google TTS API.
type googleTTSResponse struct {
	AudioContent string `json:"audioContent"`
}

// GenerateSpeech generates speech using the unified request format.
func (g *GoogleProvider) GenerateSpeech(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	if err := g.ValidateConfig(); err != nil {
		return nil, err
	}

	// Handle large texts by splitting into chunks
	textLength := len([]rune(req.Text))
	if textLength > googleMaxCharsPerRequest {
		return g.generateSpeechChunks(ctx, req)
	}

	return g.generateSpeechSingle(ctx, req)
}

// generateSpeechSingle generates speech for a single chunk.
func (g *GoogleProvider) generateSpeechSingle(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	token, err := g.getAccessToken()
	if err != nil {
		return nil, err
	}

	// Parse language code and voice name
	languageCode, voiceName := g.parseVoice(req.Voice)

	// Convert format
	audioEncoding := g.convertFormat(req.Format)

	// Prepare the request payload
	payload := googleTTSRequest{
		Input: googleTTSInput{
			Text: util.CleanJSONString(req.Text),
		},
		Voice: googleTTSVoice{
			LanguageCode: languageCode,
			Name:         voiceName,
			VoiceClone:   map[string]interface{}{},
		},
		AudioConfig: googleTTSAudioConfig{
			AudioEncoding: audioEncoding,
			SpeakingRate:  req.Speed,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", googleTTSAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	if g.ProjectID != "" {
		httpReq.Header.Set("X-Goog-User-Project", g.ProjectID)
	}

	// Send request
	resp, err := g.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Google TTS API error (status %d):", resp.StatusCode)
		if len(respBody) > 0 {
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

	// Parse response
	var ttsResp googleTTSResponse
	if err := json.Unmarshal(respBody, &ttsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Decode base64 audio content
	audioData, err := base64.StdEncoding.DecodeString(ttsResp.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio content: %w", err)
	}

	return audioData, nil
}

// generateSpeechChunks handles large texts by splitting them into chunks.
func (g *GoogleProvider) generateSpeechChunks(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	// Split text into chunks based on character limit
	chunks := g.splitTextByChars(req.Text, googleMaxCharsPerRequest)
	results := make([][]byte, len(chunks))
	errs := make([]error, len(chunks))

	var wg sync.WaitGroup
	ticker := time.NewTicker(200 * time.Millisecond) // 5 requests per second limit
	defer ticker.Stop()

	for i, chunk := range chunks {
		wg.Add(1)
		go func(i int, textChunk string) {
			defer wg.Done()
			<-ticker.C

			chunkReq := *req
			chunkReq.Text = textChunk

			data, err := g.generateSpeechSingle(ctx, &chunkReq)
			results[i] = data
			errs[i] = err
		}(i, chunk)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	// Concatenate audio data
	var combined []byte
	for _, blob := range results {
		combined = append(combined, blob...)
	}

	return combined, nil
}

// splitTextByChars splits text by character count while trying to preserve word boundaries.
func (g *GoogleProvider) splitTextByChars(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	runes := []rune(text)
	if len(runes) <= maxChars {
		return []string{text}
	}

	var chunks []string
	var current strings.Builder

	words := strings.Fields(text)
	for _, word := range words {
		// Check if adding this word would exceed the limit
		testLen := current.Len()
		if testLen > 0 {
			testLen++ // for space
		}
		testLen += len([]rune(word))

		if testLen > maxChars && current.Len() > 0 {
			// Finalize current chunk
			chunks = append(chunks, current.String())
			current.Reset()
		}

		// Add word to current chunk
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(word)
	}

	// Add the last chunk
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// parseVoice extracts language code and voice name from the voice string.
func (g *GoogleProvider) parseVoice(voice string) (languageCode, voiceName string) {
	// Default values
	languageCode = "de-DE"
	voiceName = voice

	// If voice contains a language code pattern (e.g., "de-DE-Chirp3-HD-Kore")
	parts := strings.Split(voice, "-")
	if len(parts) >= 2 {
		// Try to detect language code pattern
		if len(parts[0]) == 2 && len(parts[1]) == 2 {
			languageCode = parts[0] + "-" + parts[1]
		}
	}

	return languageCode, voiceName
}

// convertFormat converts unified format to Google TTS audio encoding.
func (g *GoogleProvider) convertFormat(format string) string {
	switch strings.ToLower(format) {
	case "mp3":
		return "MP3"
	case "linear16":
		return "LINEAR16"
	case "ogg-opus":
		return "OGG_OPUS"
	case "mulaw":
		return "MULAW"
	case "alaw":
		return "ALAW"
	default:
		return "MP3" // Default to MP3
	}
}
