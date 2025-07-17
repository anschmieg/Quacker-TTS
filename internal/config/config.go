package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/zalando/go-keyring"
)

const (
	// OpenAI keychain configuration
	openAIKeychainService = "Quacker_OpenAI"
	openAIKeychainUser    = "api_token"

	// Google Cloud keychain configuration
	googleKeychainService = "Quacker_Google"
	googleKeychainUser    = "project_id"
)

// Config holds configuration for all TTS providers.
type Config struct {
	// OpenAI configuration
	OpenAIAPIKey string

	// Google Cloud configuration
	GoogleProjectID string

	// Default provider
	DefaultProvider string
}

// LoadEnvFiles loads environment variables from .env files in the current
// directory and the user's home directory.
func LoadEnvFiles() {
	godotenv.Load() // Load .env from current directory

	homeDir, err := os.UserHomeDir()
	if err == nil {
		godotenv.Load(filepath.Join(homeDir, ".env")) // Load .env from home directory
	}
}

// LoadConfig loads configuration from environment variables and keychain.
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Load OpenAI configuration
	config.OpenAIAPIKey = getOpenAIAPIKey()

	// Load Google Cloud configuration
	config.GoogleProjectID = getGoogleProjectID()

	// Set default provider
	config.DefaultProvider = os.Getenv("DEFAULT_TTS_PROVIDER")
	if config.DefaultProvider == "" {
		// Auto-select based on available configuration
		if config.OpenAIAPIKey != "" {
			config.DefaultProvider = "openai"
		} else if config.GoogleProjectID != "" {
			config.DefaultProvider = "google"
		}
	}

	return config, nil
}

// GetAPIKey retrieves the OpenAI API key (backward compatibility).
func GetAPIKey() (string, error) {
	apiKey := getOpenAIAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not found in environment variables or keychain. Please set the environment variable or store it using keyring (e.g., 'keyring.Set(\"%s\", \"%s\", \"YOUR_API_KEY\")')", openAIKeychainService, openAIKeychainUser)
	}
	return apiKey, nil
}

// getOpenAIAPIKey retrieves the OpenAI API key from environment or keychain.
func getOpenAIAPIKey() string {
	// Check environment variable first
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		return apiKey
	}

	// Fall back to keychain
	apiKey, err := keyring.Get(openAIKeychainService, openAIKeychainUser)
	if err == nil && apiKey != "" {
		return apiKey
	}

	// Log warning but don't block
	if err != nil && err != keyring.ErrNotFound {
		fmt.Printf("Warning: OpenAI keychain access error: %v\n", err)
	}

	return ""
}

// getGoogleProjectID retrieves the Google Cloud project ID from various sources.
func getGoogleProjectID() string {
	// Check environment variable first
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID != "" {
		return projectID
	}

	// Check alternative environment variable
	projectID = os.Getenv("GCP_PROJECT")
	if projectID != "" {
		return projectID
	}

	// Try to get from gcloud config
	projectID = getGcloudProjectID()
	if projectID != "" {
		return projectID
	}

	// Fall back to keychain
	projectID, err := keyring.Get(googleKeychainService, googleKeychainUser)
	if err == nil && projectID != "" {
		return projectID
	}

	// Log warning but don't block
	if err != nil && err != keyring.ErrNotFound {
		fmt.Printf("Warning: Google Cloud keychain access error: %v\n", err)
	}

	return ""
}

// getGcloudProjectID tries to get the project ID from gcloud configuration.
func getGcloudProjectID() string {
	cmd := exec.Command("gcloud", "config", "list", "--format=value(core.project)")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	projectID := strings.TrimSpace(string(output))
	return projectID
}

// SetOpenAIAPIKey stores the OpenAI API key in the keychain.
func SetOpenAIAPIKey(apiKey string) error {
	return keyring.Set(openAIKeychainService, openAIKeychainUser, apiKey)
}

// SetGoogleProjectID stores the Google Cloud project ID in the keychain.
func SetGoogleProjectID(projectID string) error {
	return keyring.Set(googleKeychainService, googleKeychainUser, projectID)
}
