package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"easy-tts/internal/config"
	"easy-tts/internal/gui"
	"easy-tts/internal/tts"
	"easy-tts/internal/util"

	keyring "github.com/zalando/go-keyring"
)

func main() {
	// Load configuration (API Key)
	config.LoadEnvFiles()
	// Configuration for keychain service
	const keychainService = "Quacker_OpenAI"
	const keychainUser = "api_token"
	var apiKey string
	var ttsClient *tts.Client

	// Placeholder for settings dialog callback
	var showSettings func()

	// Initialize the Fyne app
	a := app.New() // a is fyne.App

	// Create the UI with onSubmit and onSettings callbacks
	var ui *gui.UI
	ui = gui.NewUI(a,
		func() { handleSubmit(ui, ttsClient, apiKey) },
		func() { showSettings() },
	)

	// Define settings dialog function for selecting or entering API key
	showSettings = func() {
		env := os.Getenv("OPENAI_API_KEY")
		chain, _ := keyring.Get(keychainService, keychainUser)
		// Build choices including option for new key
		choices := []string{}
		if env != "" {
			choices = append(choices, "Environment variable")
		}
		if chain != "" {
			choices = append(choices, "Keychain")
		}
		choices = append(choices, "New API Key")
		// Create selection widget
		var selected string
		sel := widget.NewSelect(choices, func(c string) { selected = c })
		sel.PlaceHolder = "Select API Key Source"
		entry := widget.NewPasswordEntry()
		dialog.ShowForm("API Key Settings", "OK", "Cancel",
			[]*widget.FormItem{{Text: "Source", Widget: sel}, {Text: "API Key (for New)", Widget: entry}},
			func(ok bool) {
				if !ok {
					return
				}
				switch selected {
				case "Environment variable":
					apiKey = env
				case "Keychain":
					apiKey = chain
				case "New API Key":
					if entry.Text == "" {
						return
					}
					apiKey = entry.Text
					keyring.Set(keychainService, keychainUser, apiKey)
				default:
					return
				}
				ttsClient = tts.NewClient(apiKey)
			}, ui.Window)
	}
	// Show settings dialog at startup
	showSettings()

	// Run the app
	ui.Window.ShowAndRun()
}

// handleSubmit processes the submit action
func handleSubmit(ui *gui.UI, ttsClient *tts.Client, apiKey string) {
	// Re-check API key in case it wasn't available at startup
	if apiKey == "" || ttsClient == nil {
		ui.ShowError("Error: API Key not configured. Set OPENAI_API_KEY environment variable or store in keychain.")
		return
	}

	// Disable button and show processing message
	ui.SetSubmitEnabled(false)
	ui.ShowProcessing()

	// Get data from UI elements
	inputText := ui.Input.Text
	voice := ui.Voice.Text
	speed := ui.Speed.Value

	// Run the TTS generation in a goroutine to avoid blocking the UI thread
	go func(inputText, voice string, speed float64) {
		// Ensure button is re-enabled when goroutine finishes
		defer ui.SetSubmitEnabled(true)

		// Prepare the TTS request
		requestData := tts.Request{
			Model:          "gpt-4o-mini-tts",
			Voice:          voice,
			Speed:          speed,
			Input:          inputText,
			ResponseFormat: "mp3",
		}

		// Generate speech chunks (handles input splitting)
		audioData, err := ttsClient.GenerateSpeechChunks(requestData)
		if err != nil {
			ui.ShowError(fmt.Sprintf("TTS Generation failed: %v", err))
			return
		}

		// Generate filename and save the audio file
		filename := util.GenerateFilename(inputText)
		savedPath, err := util.SaveAudioFile(audioData, filename)
		if err != nil {
			ui.ShowError(fmt.Sprintf("Failed to save file: %v", err))
			return
		}

		ui.ShowSuccess(fmt.Sprintf("File saved to %s", filepath.Base(savedPath)))

		// Send desktop notification
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Success",
			Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(savedPath)),
		})
	}(inputText, voice, speed)
}
