package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
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
		OpenAIAPIKey:     appConfig.OpenAIAPIKey,
		GoogleProjectID:  appConfig.GoogleProjectID,
		GoogleAPIKey:     appConfig.GoogleAPIKey,
		GoogleAuthMethod: appConfig.GoogleAuthMethod,
		DefaultProvider:  appConfig.DefaultProvider,
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

// Remove Markdown formatting (common symbols)
func stripMarkdown(s string) string {
	reg := regexp.MustCompile(`[\\*_#\\[\\]()>~\` + "`" + `]+`)
	return reg.ReplaceAllString(s, "")
}

// Helper: extract language code from a voice string (e.g. de-DE-Chirp3-HD-Sulafat -> de-DE)
func extractLangCode(voice string) string {
	parts := strings.Split(voice, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return "en-US"
}

// Helper: build fallback voices list
func buildFallbackVoices(origLang, origVoice string) []string {
	return []string{
		fmt.Sprintf("%s-Chirp3-HD-%s", origLang, origVoice),
		fmt.Sprintf("%s-Chirp-HD-O", origLang),
		fmt.Sprintf("%s-Neural2-G", origLang),
		fmt.Sprintf("%s-Standard-G", origLang),
		fmt.Sprintf("%s-Studio-C", origLang),
	}
}

// Recursive chunk processing with sub-chunking on failure, one-word min, special char/Markdown sanitization, and voice fallback
func processChunkRecursively(
	ctx context.Context,
	provider tts.Provider,
	request *tts.UnifiedRequest,
	chunk string,
	chunkLimit int,
	minLimit int,
	isGoogle bool,
	progressCb func(),
	uiErrorCb func(string),
) ([]byte, error) {
	var data []byte
	var err error
	origVoice := request.Voice
	origLang := extractLangCode(origVoice)
	words := strings.Fields(chunk)

	// 1. Normal attempts
	for attempt := 1; attempt <= 3; attempt++ {
		data, err = provider.GenerateSpeech(ctx, &tts.UnifiedRequest{
			Text:   chunk,
			Voice:  request.Voice,
			Speed:  request.Speed,
			Format: request.Format,
			Model:  request.Model,
		})
		if err == nil {
			if progressCb != nil {
				progressCb()
			}
			return data, nil
		}
		if attempt < 3 && (strings.Contains(err.Error(), "502") ||
			strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "DeadlineExceeded")) {
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// 2. Sub-chunking if possible
	if chunkLimit > minLimit && len(words) > 1 {
		var subChunks []string
		if isGoogle {
			subChunks = tts.SplitTextByteLimit(chunk, chunkLimit/2)
		} else {
			subChunks = tts.SplitTextTokenLimit(chunk, "cl100k_base", chunkLimit/2)
		}
		var audio []byte
		for _, sub := range subChunks {
			subData, subErr := processChunkRecursively(ctx, provider, request, sub, chunkLimit/2, minLimit, isGoogle, progressCb, uiErrorCb)
			if subErr != nil {
				return nil, subErr
			}
			audio = append(audio, subData...)
		}
		return audio, nil
	}

	// 3. If chunk is a single word and <200 bytes, try sanitizing and retry once
	if len(words) == 1 && len([]byte(chunk)) < 200 {
		sanitized := sanitizeWordForTTS(chunk)
		if sanitized != chunk && sanitized != "" {
			data, err = provider.GenerateSpeech(ctx, &tts.UnifiedRequest{
				Text:   sanitized,
				Voice:  request.Voice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				return data, nil
			}
		}
		// Try stripping Markdown and retry once more
		mdStripped := stripMarkdown(chunk)
		if mdStripped != chunk && mdStripped != "" {
			data, err = provider.GenerateSpeech(ctx, &tts.UnifiedRequest{
				Text:   mdStripped,
				Voice:  request.Voice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				return data, nil
			}
		}
	}

	// 4. Fallback voices for Google provider only
	if isGoogle && len([]byte(chunk)) <= 200 {
		fallbackVoices := buildFallbackVoices(origLang, origVoice)
		for _, fallbackVoice := range fallbackVoices {
			data, err = provider.GenerateSpeech(ctx, &tts.UnifiedRequest{
				Text:   chunk,
				Voice:  fallbackVoice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				log.Printf("Fallback voice succeeded: %s", fallbackVoice)
				return data, nil
			}
		}
	}

	// 5. Final fallback: error message chunk (en-US)
	if isGoogle && len([]byte(chunk)) <= 200 {
		log.Printf("All fallback voices failed for chunk (len=%d): %.100s", len(chunk), chunk)
		if uiErrorCb != nil {
			uiErrorCb(fmt.Sprintf(
				"A section could not be processed (%.40s...). Substituting error message and continuing.", chunk))
		}
		data, err = provider.GenerateSpeech(ctx, &tts.UnifiedRequest{
			Text:   "Error converting Text. Continuing.",
			Voice:  "en-US-" + origVoice,
			Speed:  request.Speed,
			Format: request.Format,
			Model:  request.Model,
		})
		if err == nil {
			if progressCb != nil {
				progressCb()
			}
			return data, nil
		}
	}

	// Log and show user-friendly error
	log.Printf("Final failed chunk (len=%d): %.100s", len(chunk), chunk)
	if uiErrorCb != nil {
		uiErrorCb(fmt.Sprintf(
			"A section could not be processed (%.40s...). Try rephrasing or splitting it manually.", chunk))
	}
	return nil, err
}

// Remove special characters, keep only letters, numbers, and spaces
func sanitizeWordForTTS(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
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

	// Capture UI values before starting goroutine
	inputText := ui.Input.Text
	voice := ui.Voice.Text
	speed := ui.Speed.Value

	// Basic validation
	if inputText == "" {
		ui.ShowError("Please enter some text to convert to speech.")
		return
	}

	// Initialize UI state synchronously
	ui.SetSubmitEnabled(false)
	ui.SetProcessingMessage("Starting TTS processing...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// Do NOT defer cancel() here! Only call cancel() if you want to abort early or after all work is done.

	// Get provider instance
	provider, err := ttsManager.GetProvider(providerName)
	if err != nil {
		cancel()
		ui.ShowError(fmt.Sprintf("Provider error: %v", err))
		ui.SetSubmitEnabled(true)
		return
	}

	// Start processing in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in submit handler: %v", r)
				ui.SetSubmitEnabled(true)
				ui.ShowError(fmt.Sprintf("Internal error: %v", r))
			} else {
				ui.SetSubmitEnabled(true)
			}
		}()

		log.Printf("Starting TTS request: provider=%s, voice=%s, speed=%f, text_length=%d",
			providerName, voice, speed, len(inputText))

		// 1. Authorization check
		ui.SetProcessingMessage("Checking authorization...")
		if err := provider.CheckAuth(ctx); err != nil {
			log.Printf("Authorization failed: %v", err)
			ui.ShowError(fmt.Sprintf("Authorization failed: %v", err))
			return
		}

		// 2. Prepare request template
		request := &tts.UnifiedRequest{
			Text:   inputText,
			Voice:  voice,
			Speed:  speed,
			Format: "mp3",
		}
		if providerName == "openai" {
			request.Model = "gpt-4o-mini-tts"
		}

		// Determine total chunks for progress reporting
		var totalChunks int
		if provider.GetName() == "google" {
			totalChunks = len(tts.SplitTextByteLimit(inputText, tts.DefaultByteLimit))
		} else {
			totalChunks = len(tts.SplitTextTokenLimit(inputText, "cl100k_base", provider.GetMaxTokensPerChunk()))
		}
		ui.SetProgress(0)
		ui.SetProcessingMessage(fmt.Sprintf("Processing chunk 1 of %d...", totalChunks))

		// 3. Call the processor
		var audioData []byte
		progressCb := func(completed, total int) {
			ui.SetProgress(float64(completed) / float64(total))
			ui.SetProcessingMessage(fmt.Sprintf("Processing chunk %d of %d...", completed, total))
		}
		uiErrorCb := func(msg string) {
			ui.ShowError(msg)
		}

		audioData, err = tts.ProcessTextToSpeech(ctx, provider, request, progressCb, uiErrorCb, nil)
		// Always save audio file if any audio was produced, even on error
		if len(audioData) > 0 {
			filename := util.GenerateFilename(inputText)
			savedPath, saveErr := util.SaveAudioFile(audioData, filename)
			if err != nil {
				// Error occurred, but we have partial audio
				if saveErr == nil {
					ui.ShowError(fmt.Sprintf("Partial audio saved to %s. Some sections could not be processed.", filepath.Base(savedPath)))
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "Partial Success",
						Content: fmt.Sprintf("Partial audio saved to: %s", filepath.Base(savedPath)),
					})
				} else {
					ui.ShowError(fmt.Sprintf("Error occurred and failed to save partial audio: %v", saveErr))
				}
				return
			}
			log.Printf("TTS generation successful, audio data size: %d bytes", len(audioData))
		}

		// Update UI for file saving
		ui.SetProcessingMessage("Saving audio file...")

		filename := util.GenerateFilename(inputText)
		log.Printf("Saving audio file: %s", filename)
		savedPath, err := util.SaveAudioFile(audioData, filename)
		if err != nil {
			log.Printf("Failed to save file: %v", err)
			ui.ShowError(fmt.Sprintf("Failed to save file: %v", err))
			return
		}
		log.Printf("Audio file saved successfully: %s", savedPath)

		// Show success message
		log.Printf("TTS request completed successfully")
		ui.ShowSuccess(fmt.Sprintf("File saved to %s (Provider: %s)", filepath.Base(savedPath), providerName))
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Success",
			Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(savedPath)),
		})
		// Clean up context at the very end
		cancel()
	}()
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
	// Provider selection (moved above tabs)
	providerInfo := ttsManager.GetProviderInfo()
	var providerNames []string
	for _, info := range providerInfo {
		providerNames = append(providerNames, info.Name)
	}

	defaultProviderSelect := widget.NewSelect(providerNames, nil)
	defaultProviderSelect.SetSelected(ttsManager.GetConfig().DefaultProvider)

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
	googleProjectLabel := widget.NewLabel("Project ID:")

	googleAPIKeyEntry := widget.NewPasswordEntry()
	googleAPIKeyEntry.SetText(ttsManager.GetConfig().GoogleAPIKey)
	googleAPIKeyLabel := widget.NewLabel("API Key:")

	// updateGoogleFields toggles visibility of provider-specific fields
	updateGoogleFields := func(method string) {
		if method == "API Key" {
			googleProjectLabel.Hide()
			googleProjectEntry.Hide()
			googleAPIKeyLabel.Show()
			googleAPIKeyEntry.Show()
		} else { // "gcloud auth"
			googleProjectLabel.Show()
			googleProjectEntry.Show()
			googleAPIKeyLabel.Hide()
			googleAPIKeyEntry.Hide()
		}
	}

	// Google Cloud authentication method selection
	googleAuthMethods := []string{"gcloud auth", "API Key"}
	googleAuthSelect := widget.NewSelect(googleAuthMethods, updateGoogleFields)

	// Set current auth method from config and trigger initial field visibility
	currentAuthMethod := ttsManager.GetConfig().GoogleAuthMethod
	if currentAuthMethod == "" {
		currentAuthMethod = "gcloud auth" // Default to gcloud auth
	}
	googleAuthSelect.SetSelected(currentAuthMethod)
	updateGoogleFields(currentAuthMethod)

	googleContent := container.New(layout.NewFormLayout(),
		widget.NewLabel("Auth Method:"), googleAuthSelect,
		googleProjectLabel, googleProjectEntry,
		googleAPIKeyLabel, googleAPIKeyEntry,
	)
	tabs.Append(container.NewTabItem("Google Cloud", googleContent))

	mainContent := container.NewVBox(
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Default Provider:"), defaultProviderSelect,
		),
		tabs,
	)

	dialog := dialog.NewCustomConfirm("Provider Settings", "Save", "Cancel", mainContent, func(ok bool) {
		if !ok {
			return
		}

		// Update configuration
		newConfig := &tts.ProviderConfig{
			OpenAIAPIKey:     openAIAPIKeyEntry.Text,
			GoogleProjectID:  googleProjectEntry.Text,
			GoogleAPIKey:     googleAPIKeyEntry.Text,
			GoogleAuthMethod: googleAuthSelect.Selected,
			DefaultProvider:  defaultProviderSelect.Selected,
		}

		// Save to keychain
		if openAIAPIKeyEntry.Text != "" {
			config.SetOpenAIAPIKey(openAIAPIKeyEntry.Text)
		}
		if googleProjectEntry.Text != "" {
			config.SetGoogleProjectID(googleProjectEntry.Text)
		}
		if googleAPIKeyEntry.Text != "" {
			config.SetGoogleAPIKey(googleAPIKeyEntry.Text)
		}
		if googleAuthSelect.Selected != "" {
			config.SetGoogleAuthMethod(googleAuthSelect.Selected)
		}

 		// Persist default provider to keychain
 		if err := config.SetDefaultProvider(defaultProviderSelect.Selected); err != nil {
 			log.Printf("Failed to save default provider to keychain: %v", err)
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
