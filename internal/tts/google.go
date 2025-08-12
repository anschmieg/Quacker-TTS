package tts

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"google.golang.org/api/option"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

// GoogleProvider handles communication with the Google Cloud TTS API using the Go SDK.
type GoogleProvider struct {
	ProjectID  string
	APIKey     string
	AuthMethod string // "gcloud auth" or "API Key"

	// Caches the client to avoid re-initializing on every request.
	ttsClient  *texttospeech.Client
	clientOnce sync.Once
	clientErr  error
}

// NewGoogleProvider creates a new Google TTS provider.
func NewGoogleProvider(projectID, apiKey, authMethod string) *GoogleProvider {
	return &GoogleProvider{
		ProjectID:  projectID,
		APIKey:     apiKey,
		AuthMethod: authMethod,
	}
}

// GetName returns the provider's name.
func (g *GoogleProvider) GetName() string {
	return "google"
}

// GetDefaultVoice returns the provider's default voice.
func (g *GoogleProvider) GetDefaultVoice() string {
	return "de-DE-Chirp3-HD-Sulafat" // Default to Chirp3-HD-Sulafat as requested
}

// GetSupportedFormats returns the audio formats supported by this provider.
func (g *GoogleProvider) GetSupportedFormats() []string {
	// These are the formats supported by the SDK's AudioEncoding enum
	return []string{"mp3", "linear16", "ogg_opus", "mulaw", "alaw"}
}

// ValidateConfig validates the provider's configuration.
func (g *GoogleProvider) ValidateConfig() error {
	if g.ProjectID == "" {
		return fmt.Errorf("Google Cloud project ID is required")
	}

	if g.AuthMethod == "API Key" && g.APIKey == "" {
		return fmt.Errorf("Google Cloud API key is required for API Key authentication")
	}
	return nil
}

// GetMaxTokensPerChunk returns a value based on the byte limit.
// Note: Google uses a byte/character limit, not tokens. This is an approximation.
func (g *GoogleProvider) GetMaxTokensPerChunk() int {
	return DefaultByteLimit / 3
}

// getClient initializes and returns a thread-safe, cached TTS client.
func (g *GoogleProvider) getClient(ctx context.Context) (*texttospeech.Client, error) {
	g.clientOnce.Do(func() {
		log.Println("Initializing Google TTS client...")
		var opts []option.ClientOption

		if g.AuthMethod == "API Key" {
			log.Println("Using API Key authentication.")
			opts = append(opts, option.WithAPIKey(g.APIKey))
		} else {
			log.Println("Using Application Default Credentials (gcloud auth).")
			// The SDK automatically uses ADC when no explicit credentials are provided.
			// The project ID is not passed as an option here but is used in headers if needed.
		}

		client, err := texttospeech.NewClient(ctx, opts...)
		if err != nil {
			g.clientErr = fmt.Errorf("failed to create Google TTS client: %w", err)
			log.Printf("Google TTS client initialization failed: %v", g.clientErr)
			return
		}
		g.ttsClient = client
		log.Println("Google TTS client initialized successfully.")
	})

	return g.ttsClient, g.clientErr
}

// CheckAuth verifies that the Google credentials are valid by attempting to list available voices.
func (g *GoogleProvider) CheckAuth(ctx context.Context) error {
	client, err := g.getClient(ctx)
	if err != nil {
		// Reset the client initialization state on failure to allow retry
		g.clientOnce = sync.Once{}
		g.ttsClient = nil
		g.clientErr = nil
		return fmt.Errorf("authentication failed during client creation: %w", err)
	}

	// Listing voices is a lightweight, non-billable way to test authentication.
	req := &texttospeechpb.ListVoicesRequest{}
	_, err = client.ListVoices(ctx, req)
	if err != nil {
		// Reset client on auth error to allow re-authentication (e.g., if key changes)
		g.clientOnce = sync.Once{}
		g.ttsClient = nil
		g.clientErr = nil
		return fmt.Errorf("authentication check (ListVoices) failed: %w", err)
	}

	return nil
}

// GenerateSpeech generates speech using the unified request format.
func (g *GoogleProvider) GenerateSpeech(ctx context.Context, req *UnifiedRequest) ([]byte, error) {
	if err := g.ValidateConfig(); err != nil {
		return nil, err
	}

	client, err := g.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// Parse language code and voice name from the unified voice string.
	languageCode, voiceName := g.parseVoice(req.Voice)

	// Prepare the SDK-specific request.
	ttsReq := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: req.Text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         voiceName,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: g.convertFormat(req.Format),
			SpeakingRate:  req.Speed,
		},
	}

	log.Printf("Sending request to Google TTS API for text: '%.30s...'", req.Text)
	resp, err := client.SynthesizeSpeech(ctx, ttsReq)
	if err != nil {
		// Try to log full error details if available
		type causer interface{ Unwrap() error }
		unwrapped := err
		for i := 0; i < 5; i++ {
			if unwrapped == nil {
				break
			}
			log.Printf("Google TTS error (unwrap %d): %v", i, unwrapped)
			if c, ok := unwrapped.(causer); ok {
				unwrapped = c.Unwrap()
			} else {
				break
			}
		}
		log.Printf("Google TTS SynthesizeSpeech failed: %v", err)
		return nil, fmt.Errorf("Google TTS API error: %w", err)
	}
	log.Printf("Successfully received audio data (len=%d)", len(resp.AudioContent))

	return resp.AudioContent, nil
}

// parseVoice extracts language code and voice name from the voice string.
// Example: "de-DE-Wavenet-F" -> "de-DE", "de-DE-Wavenet-F"
func (g *GoogleProvider) parseVoice(voice string) (languageCode, voiceName string) {
	// Default to a sensible language code if parsing fails.
	languageCode = "en-US"
	voiceName = voice

	parts := strings.Split(voice, "-")
	if len(parts) >= 2 {
		// A valid BCP-47 language tag is usually the first two parts.
		if len(parts[0]) == 2 && len(parts[1]) == 2 {
			languageCode = parts[0] + "-" + parts[1]
		}
	}
	return languageCode, voiceName
}

// convertFormat converts a unified format string to the Google TTS audio encoding enum.
func (g *GoogleProvider) convertFormat(format string) texttospeechpb.AudioEncoding {
	switch strings.ToUpper(format) {
	case "MP3":
		return texttospeechpb.AudioEncoding_MP3
	case "LINEAR16":
		return texttospeechpb.AudioEncoding_LINEAR16
	case "OGG_OPUS":
		return texttospeechpb.AudioEncoding_OGG_OPUS
	case "MULAW":
		return texttospeechpb.AudioEncoding_MULAW
	case "ALAW":
		return texttospeechpb.AudioEncoding_ALAW
	default:
		log.Printf("Unsupported format '%s', defaulting to MP3.", format)
		return texttospeechpb.AudioEncoding_MP3
	}
}
