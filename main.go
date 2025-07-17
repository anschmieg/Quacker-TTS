package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"easy-tts/internal/config"
	"easy-tts/internal/gui"
	"easy-tts/internal/tts"
	"easy-tts/internal/util"
)

func main() {
	// Load configuration
	config.LoadEnvFiles()
	appConfig, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	// Create TTS provider configuration
	providerConfig := &tts.ProviderConfig{
		OpenAIAPIKey:    appConfig.OpenAIAPIKey,
		GoogleProjectID: appConfig.GoogleProjectID,
		DefaultProvider: appConfig.DefaultProvider,
	}

	// Initialize TTS manager
	ttsManager := tts.NewManager(providerConfig)

	// Get available providers
	availableProviders := ttsManager.GetAvailableProviders()
	if len(availableProviders) == 0 {
		fmt.Println("No TTS providers configured. Please configure at least one provider.")
	}

	// Placeholder for settings dialog callback
	var showSettings func()

	// Initialize the Fyne app
	a := app.New()

	// Current provider state
	var currentProvider string
	if len(availableProviders) > 0 {
		currentProvider = availableProviders[0]
		if appConfig.DefaultProvider != "" {
			currentProvider = appConfig.DefaultProvider
		}
	}

	// Track initialization state
	var uiInitialized bool

	// Create the UI with callbacks
	var ui *gui.UI
	ui = gui.NewUI(a, availableProviders,
		func() { handleSubmit(ui, ttsManager, currentProvider) },
		func() { showSettings() },
		func(provider string) {
			currentProvider = provider
			if uiInitialized {
				updateVoiceForProvider(ui, ttsManager, provider)
			}
		},
	)

	// Mark UI as initialized
	uiInitialized = true

	// Define settings dialog function for configuring providers
	showSettings = func() {
		showProviderSettingsDialog(ui, ttsManager, &currentProvider)
	}

	// Set initial provider after UI is fully initialized
	if currentProvider != "" {
		ui.ProviderSelect.SetSelected(currentProvider)
		updateVoiceForProvider(ui, ttsManager, currentProvider)
	}

	// Show settings dialog at startup only if no providers are configured
	if len(availableProviders) == 0 {
		showSettings()
	}

	// Run the app
	ui.Window.ShowAndRun()
}

// handleSubmit processes the submit action
func handleSubmit(ui *gui.UI, ttsManager *tts.Manager, providerName string) {
	if providerName == "" {
		fyne.Do(func() {
			ui.ShowError("Error: No TTS provider selected.")
		})
		return
	}

	// Validate provider configuration
	if err := ttsManager.ValidateProvider(providerName); err != nil {
		fyne.Do(func() {
			ui.ShowError(fmt.Sprintf("Provider '%s' configuration error: %v", providerName, err))
		})
		return
	}

	fyne.Do(func() {
		ui.SetSubmitEnabled(false)
		ui.ShowProgressBar()
		ui.SetProgress(0)
		ui.SetProcessingMessage("Starting TTS processing...")
	})

	inputText := ui.Input.Text
	voice := ui.Voice.Text
	speed := ui.Speed.Value

	go func(inputText, voice string, speed float64) {
		defer fyne.DoAndWait(func() {
			ui.SetSubmitEnabled(true)
		})

		// Create unified request
		request := &tts.UnifiedRequest{
			Text:   inputText,
			Voice:  voice,
			Speed:  speed,
			Format: "mp3",
		}

		// Add provider-specific fields
		if providerName == "openai" {
			request.Model = "gpt-4o-mini-tts"
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		fyne.Do(func() {
			ui.SetProcessingMessage("Generating speech...")
		})

		// Generate speech using the manager
		response, err := ttsManager.GenerateSpeech(ctx, request, providerName)
		if err != nil {
			fyne.Do(func() {
				ui.HideProgressBar()
				ui.ShowError(fmt.Sprintf("TTS Generation failed: %v", err))
			})
			return
		}

		fyne.Do(func() {
			ui.SetProgress(0.8)
			ui.SetProcessingMessage("Saving audio file...")
		})

		filename := util.GenerateFilename(inputText)
		savedPath, err := util.SaveAudioFile(response.AudioData, filename)
		if err != nil {
			fyne.Do(func() {
				ui.HideProgressBar()
				ui.ShowError(fmt.Sprintf("Failed to save file: %v", err))
			})
			return
		}

		fyne.Do(func() {
			ui.HideProgressBar()
			ui.ShowSuccess(fmt.Sprintf("File saved to %s (Provider: %s)", filepath.Base(savedPath), response.Provider))
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Success",
				Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(savedPath)),
			})
		})
	}(inputText, voice, speed)
}

// updateVoiceForProvider updates the voice field with the provider's default voice
func updateVoiceForProvider(ui *gui.UI, ttsManager *tts.Manager, providerName string) {
	if ui == nil || providerName == "" {
		return
	}

	provider, err := ttsManager.GetProvider(providerName)
	if err != nil {
		return
	}

	defaultVoice := provider.GetDefaultVoice()
	ui.Voice.SetText(defaultVoice)
}

// showProviderSettingsDialog shows the provider configuration dialog
func showProviderSettingsDialog(ui *gui.UI, ttsManager *tts.Manager, currentProvider *string) {
	// Create tabs for different providers
	tabs := container.NewAppTabs()

	// OpenAI tab
	openAIAPIKeyEntry := widget.NewPasswordEntry()
	openAIAPIKeyEntry.SetText(ttsManager.GetConfig().OpenAIAPIKey)

	openAIContent := container.New(layout.NewFormLayout(),
		widget.NewLabel("API Key:"), openAIAPIKeyEntry,
	)
	tabs.Append(container.NewTabItem("OpenAI", openAIContent))

	// Google Cloud tab
	googleProjectEntry := widget.NewEntry()
	googleProjectEntry.SetText(ttsManager.GetConfig().GoogleProjectID)

	googleContent := container.New(layout.NewFormLayout(),
		widget.NewLabel("Project ID:"), googleProjectEntry,
		widget.NewLabel("Auth:"), widget.NewLabel("Uses gcloud auth"),
	)
	tabs.Append(container.NewTabItem("Google Cloud", googleContent))

	// Provider selection
	providerInfo := ttsManager.GetProviderInfo()
	var providerNames []string
	for _, info := range providerInfo {
		providerNames = append(providerNames, info.Name)
	}

	defaultProviderSelect := widget.NewSelect(providerNames, nil)
	defaultProviderSelect.SetSelected(ttsManager.GetConfig().DefaultProvider)

	mainContent := container.NewVBox(
		tabs,
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Default Provider:"), defaultProviderSelect,
		),
	)

	dialog := dialog.NewCustomConfirm("Provider Settings", "Save", "Cancel", mainContent, func(ok bool) {
		if !ok {
			return
		}

		// Update configuration
		newConfig := &tts.ProviderConfig{
			OpenAIAPIKey:    openAIAPIKeyEntry.Text,
			GoogleProjectID: googleProjectEntry.Text,
			DefaultProvider: defaultProviderSelect.Selected,
		}

		// Save to keychain
		if openAIAPIKeyEntry.Text != "" {
			config.SetOpenAIAPIKey(openAIAPIKeyEntry.Text)
		}
		if googleProjectEntry.Text != "" {
			config.SetGoogleProjectID(googleProjectEntry.Text)
		}

		// Update manager
		ttsManager.UpdateConfig(newConfig)

		// Update UI
		availableProviders := ttsManager.GetAvailableProviders()
		ui.ProviderSelect.Options = availableProviders

		if len(availableProviders) > 0 {
			newProvider := newConfig.DefaultProvider
			if newProvider == "" {
				newProvider = availableProviders[0]
			}
			*currentProvider = newProvider
			ui.ProviderSelect.SetSelected(newProvider)
			updateVoiceForProvider(ui, ttsManager, newProvider)
		}

	}, ui.Window)

	dialog.Resize(fyne.NewSize(500, 400))
	dialog.Show()
}
