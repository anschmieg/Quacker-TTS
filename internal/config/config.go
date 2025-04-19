package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/zalando/go-keyring"
)

const (
	keychainService = "OpenAI_TTS"
	keychainUser    = "api_token"
)

// LoadEnvFiles loads environment variables from .env files in the current
// directory and the user's home directory.
func LoadEnvFiles() {
	godotenv.Load() // Load .env from current directory

	homeDir, err := os.UserHomeDir()
	if err == nil {
		godotenv.Load(filepath.Join(homeDir, ".env")) // Load .env from home directory
	}
}

// GetAPIKey retrieves the OpenAI API key.
// It checks the environment variable OPENAI_API_KEY first,
// then falls back to the system keychain.
// Returns the API key and an error if not found.
func GetAPIKey() (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		return apiKey, nil
	}

	apiKey, err := keyring.Get(keychainService, keychainUser)
	if err == nil && apiKey != "" {
		return apiKey, nil
	}

	// If keyring access failed or key wasn't found
	if err != nil && err != keyring.ErrNotFound {
		fmt.Printf("Warning: Keychain access error: %v\n", err) // Log warning but don't block
	}

	return "", fmt.Errorf("OPENAI_API_KEY not found in environment variables or keychain. Please set the environment variable or store it using keyring (e.g., 'keyring.Set(\"%s\", \"%s\", \"YOUR_API_KEY\")')", keychainService, keychainUser)
}
