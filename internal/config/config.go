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
	googleAPIKeyKeychainService = "Quacker_Google_API"
	googleAPIKeyKeychainUser    = "api_key"
	googleAuthMethodKeychainService = "Quacker_Google_Auth"
	googleAuthMethodKeychainUser    = "auth_method"
)

// Keychain configuration for default provider
const (
	defaultProviderKeychainService = "Quacker_DefaultProvider"
	defaultProviderKeychainUser    = "default"
)

// Config holds configuration for all TTS providers.
type Config struct {
	// OpenAI configuration
	OpenAIAPIKey string

	// Google Cloud configuration
	GoogleProjectID  string
	GoogleAPIKey     string
	GoogleAuthMethod string

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
	config.GoogleAPIKey = getGoogleAPIKey()
	config.GoogleAuthMethod = getGoogleAuthMethod()

	// Set default provider from keychain, then env, then auto
	config.DefaultProvider = GetDefaultProviderFromKeychain()
	if config.DefaultProvider == "" {
		config.DefaultProvider = os.Getenv("DEFAULT_TTS_PROVIDER")
	}
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

// SetDefaultProvider stores the default provider in the keychain.
func SetDefaultProvider(provider string) error {
	return keyring.Set(defaultProviderKeychainService, defaultProviderKeychainUser, provider)
}

// GetDefaultProviderFromKeychain retrieves the default provider from the keychain.
func GetDefaultProviderFromKeychain() string {
	val, err := keyring.Get(defaultProviderKeychainService, defaultProviderKeychainUser)
	if err == nil && val != "" {
		return val
	}
	return ""
}

// getGoogleAPIKey retrieves the Google Cloud API key from environment or keychain.
func getGoogleAPIKey() string {
	// Check environment variable first
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey != "" {
		return apiKey
	}

	// Check alternative environment variable
	apiKey = os.Getenv("GOOGLE_CLOUD_API_KEY")
	if apiKey != "" {
		return apiKey
	}

	// Fall back to keychain
	apiKey, err := keyring.Get(googleAPIKeyKeychainService, googleAPIKeyKeychainUser)
	if err == nil && apiKey != "" {
		return apiKey
	}

	// Log warning but don't block
	if err != nil && err != keyring.ErrNotFound {
		fmt.Printf("Warning: Google API key keychain access error: %v\n", err)
	}

	return ""
}

// getGoogleAuthMethod retrieves the Google Cloud authentication method from keychain.
func getGoogleAuthMethod() string {
	// Check environment variable first
	method := os.Getenv("GOOGLE_AUTH_METHOD")
	if method != "" {
		return method
	}

	// Fall back to keychain
	method, err := keyring.Get(googleAuthMethodKeychainService, googleAuthMethodKeychainUser)
	if err == nil && method != "" {
		return method
	}

	// Log warning but don't block
	if err != nil && err != keyring.ErrNotFound {
		fmt.Printf("Warning: Google auth method keychain access error: %v\n", err)
	}

	// Default to gcloud auth
	return "gcloud auth"
}

// SetGoogleAPIKey stores the Google Cloud API key in the keychain.
func SetGoogleAPIKey(apiKey string) error {
	return keyring.Set(googleAPIKeyKeychainService, googleAPIKeyKeychainUser, apiKey)
}

// SetGoogleAuthMethod stores the Google Cloud authentication method in the keychain.
func SetGoogleAuthMethod(method string) error {
	return keyring.Set(googleAuthMethodKeychainService, googleAuthMethodKeychainUser, method)
}
