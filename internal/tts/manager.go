package tts

import (
	"context"
	"fmt"
	"strings"
)

// Manager handles multiple TTS providers and provides a unified interface.
type Manager struct {
	providers       map[string]Provider
	defaultProvider string
	config          *ProviderConfig
}

// NewManager creates a new TTS provider manager.
func NewManager(config *ProviderConfig) *Manager {
	m := &Manager{
		providers: make(map[string]Provider),
		config:    config,
	}

	// Initialize providers based on configuration
	m.initializeProviders()

	return m
}

// initializeProviders sets up all available providers based on configuration.
func (m *Manager) initializeProviders() {
	// Initialize OpenAI provider if API key is available
	if m.config.OpenAIAPIKey != "" {
		openaiProvider := NewOpenAIProvider(m.config.OpenAIAPIKey)
		m.providers["openai"] = openaiProvider
	}

	// Initialize Google provider if project ID is available
	if m.config.GoogleProjectID != "" {
		googleProvider := NewGoogleProvider(m.config.GoogleProjectID)
		m.providers["google"] = googleProvider
	}

	// Set default provider
	if m.config.DefaultProvider != "" {
		m.defaultProvider = m.config.DefaultProvider
	} else {
		// Auto-select first available provider
		for name := range m.providers {
			m.defaultProvider = name
			break
		}
	}
}

// GetProvider returns a specific provider by name.
func (m *Manager) GetProvider(name string) (Provider, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	return provider, nil
}

// GetDefaultProvider returns the default provider.
func (m *Manager) GetDefaultProvider() (Provider, error) {
	if m.defaultProvider == "" {
		return nil, fmt.Errorf("no default provider configured")
	}
	return m.GetProvider(m.defaultProvider)
}

// SetDefaultProvider sets the default provider.
func (m *Manager) SetDefaultProvider(name string) error {
	if _, exists := m.providers[name]; !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}
	m.defaultProvider = name
	return nil
}

// GetAvailableProviders returns a list of all available provider names.
func (m *Manager) GetAvailableProviders() []string {
	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GetProviderInfo returns information about all available providers.
func (m *Manager) GetProviderInfo() []ProviderInfo {
	var infos []ProviderInfo
	for name, provider := range m.providers {
		info := ProviderInfo{
			Name:             name,
			DisplayName:      strings.Title(name),
			DefaultVoice:     provider.GetDefaultVoice(),
			SupportedFormats: provider.GetSupportedFormats(),
			RequiresAuth:     true,
			Configured:       provider.ValidateConfig() == nil,
		}
		infos = append(infos, info)
	}
	return infos
}

// GenerateSpeech generates speech using the specified provider or default provider.
func (m *Manager) GenerateSpeech(ctx context.Context, req *UnifiedRequest, providerName string) (*UnifiedResponse, error) {
	var provider Provider
	var err error

	if providerName != "" {
		provider, err = m.GetProvider(providerName)
	} else {
		provider, err = m.GetDefaultProvider()
	}

	if err != nil {
		return nil, err
	}

	// Validate the provider configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider '%s' configuration error: %w", provider.GetName(), err)
	}

	// Set default values based on provider
	if req.Voice == "" {
		req.Voice = provider.GetDefaultVoice()
	}
	if req.Format == "" {
		formats := provider.GetSupportedFormats()
		if len(formats) > 0 {
			req.Format = formats[0]
		}
	}
	if req.Speed <= 0 {
		req.Speed = 1.0
	}

	// Generate speech
	audioData, err := provider.GenerateSpeech(ctx, req)
	if err != nil {
		return nil, err
	}

	return &UnifiedResponse{
		AudioData: audioData,
		Format:    req.Format,
		Provider:  provider.GetName(),
	}, nil
}

// ValidateProvider checks if a provider is properly configured.
func (m *Manager) ValidateProvider(name string) error {
	provider, err := m.GetProvider(name)
	if err != nil {
		return err
	}
	return provider.ValidateConfig()
}

// GetVoicesForProvider returns available voices for a specific provider.
// This is a placeholder for future implementation when we add voice discovery.
func (m *Manager) GetVoicesForProvider(providerName string) ([]VoiceInfo, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// For now, return the default voice
	// In the future, we can implement API calls to get available voices
	defaultVoice := VoiceInfo{
		Name:         provider.GetDefaultVoice(),
		DisplayName:  provider.GetDefaultVoice(),
		LanguageCode: "en-US", // Default, should be provider-specific
		Gender:       "neutral",
		Provider:     provider.GetName(),
	}

	return []VoiceInfo{defaultVoice}, nil
}

// UpdateConfig updates the provider configuration and reinitializes providers.
func (m *Manager) UpdateConfig(config *ProviderConfig) {
	m.config = config
	m.providers = make(map[string]Provider)
	m.initializeProviders()
}

// GetConfig returns the current provider configuration.
func (m *Manager) GetConfig() *ProviderConfig {
	return m.config
}
