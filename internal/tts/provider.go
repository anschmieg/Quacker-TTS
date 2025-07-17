package tts

import "context"

// Provider defines the interface that all TTS providers must implement.
type Provider interface {
	// GenerateSpeech generates audio from text using the provider's API
	GenerateSpeech(ctx context.Context, req *UnifiedRequest) ([]byte, error)

	// GetName returns the provider's name (e.g., "openai", "google")
	GetName() string

	// GetDefaultVoice returns the provider's default voice
	GetDefaultVoice() string

	// GetSupportedFormats returns the audio formats supported by this provider
	GetSupportedFormats() []string

	// ValidateConfig validates the provider's configuration
	ValidateConfig() error

	// GetMaxTokensPerChunk returns the maximum tokens per request for this provider
	GetMaxTokensPerChunk() int
}

// UnifiedRequest represents a unified TTS request that works across providers
type UnifiedRequest struct {
	// Common fields
	Text   string  `json:"text"`
	Voice  string  `json:"voice"`
	Speed  float64 `json:"speed"`
	Format string  `json:"format"`

	// Provider-specific fields (optional)
	Model        string `json:"model,omitempty"`        // OpenAI specific
	LanguageCode string `json:"language_code,omitempty"` // Google specific
	Instructions string `json:"instructions,omitempty"`  // For future use
}

// UnifiedResponse represents a unified TTS response
type UnifiedResponse struct {
	AudioData []byte
	Format    string
	Provider  string
}

// ProviderConfig holds configuration for all providers
type ProviderConfig struct {
	// OpenAI configuration
	OpenAIAPIKey string

	// Google Cloud configuration
	GoogleProjectID   string
	GoogleCredentials string // Path to service account JSON or JSON content

	// Default provider
	DefaultProvider string
}

// VoiceInfo represents information about a voice
type VoiceInfo struct {
	Name         string
	DisplayName  string
	LanguageCode string
	Gender       string
	Provider     string
}

// ProviderInfo represents information about a TTS provider
type ProviderInfo struct {
	Name            string
	DisplayName     string
	DefaultVoice    string
	SupportedFormats []string
	RequiresAuth    bool
	Configured      bool
}
