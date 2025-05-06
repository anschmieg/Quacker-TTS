package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Initial load of API key (env or keychain)
	env := os.Getenv("OPENAI_API_KEY")
	chainVal, _ := keyring.Get(keychainService, keychainUser)
	if env != "" {
		apiKey = env
		ttsClient = tts.NewClient(apiKey)
	} else if chainVal != "" {
		apiKey = chainVal
		ttsClient = tts.NewClient(apiKey)
	}

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
		// Prepare select and conditional widgets
		env := os.Getenv("OPENAI_API_KEY")
		chain, _ := keyring.Get(keychainService, keychainUser)
		choices := []string{}
		if env != "" {
			choices = append(choices, "Environment variable")
		}
		if chain != "" {
			choices = append(choices, "Keychain")
		}
		choices = append(choices, "Enter new key")

		sel := widget.NewSelect(choices, nil)
		sel.PlaceHolder = "Select API Key Source"

		entry := widget.NewPasswordEntry()

		// Initial dialog content using FormLayout for spacing
		content := container.New(layout.NewFormLayout(),
			widget.NewLabel("Source"), sel,
		)

		// Create custom confirmation dialog
		dlg := dialog.NewCustomConfirm("API Key Settings", "OK", "Cancel", content, func(ok bool) {
			if !ok {
				return
			}
			// Determine final API key
			switch sel.Selected {
			case "Environment variable":
				apiKey = env
			case "Keychain":
				apiKey = chain
			case "Enter new key":
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

		// Update dialog when selection changes
		sel.OnChanged = func(c string) {
			// Rebuild content with conditional row
			objs := []fyne.CanvasObject{
				widget.NewLabel("Source"), sel,
			}
			if c == "Enter new key" {
				objs = append(objs,
					widget.NewLabel("API Key (for New)"), entry,
				)
			}
			content.Objects = objs
			content.Refresh()
			dlg.Resize(content.MinSize().Add(fyne.NewSize(0, 40)))
		}

		// Show the dialog
		dlg.Show()
	}

	// Show settings dialog at startup only if API key not found
	if apiKey == "" {
		showSettings()
	}

	// Run the app
	ui.Window.ShowAndRun()
}

// handleSubmit processes the submit action
func handleSubmit(ui *gui.UI, ttsClient *tts.Client, apiKey string) {
	if apiKey == "" || ttsClient == nil {
		fyne.Do(func() {
			ui.ShowError("Error: API Key not configured. Set OPENAI_API_KEY environment variable or store in keychain.")
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

		requestData := tts.Request{
			Model:          "gpt-4o-mini-tts",
			Voice:          voice,
			Speed:          speed,
			Input:          inputText,
			ResponseFormat: "mp3",
		}

		// --- Progress calculation setup ---
		maxTokens := 2000
		model := requestData.Model
		chunks := tts.SplitTextForProgress(inputText, model, maxTokens)
		numChunks := len(chunks)
		tokenCounts := make([]int, numChunks)
		totalTokens := 0
		for i, c := range chunks {
			tokenCounts[i] = tts.CountTokens(model, c)
			totalTokens += tokenCounts[i]
		}
		x := 10.0
		y := 5.0
		z := 2.0
		total := x + y*float64(numChunks) + float64(totalTokens)
		if numChunks > 1 {
			total += z * float64(numChunks)
		}
		progress := 0.0

		progress += x
		fyne.Do(func() {
			ui.SetProgress(progress / total)
			ui.SetProcessingMessage(fmt.Sprintf("Preparing %d chunk(s) for synthesis...", numChunks))
		})

		results := make([][]byte, numChunks)
		hasErr := false
		for i, chunk := range chunks {
			fyne.Do(func() {
				ui.SetProcessingMessage(fmt.Sprintf("Processing chunk %d of %d...", i+1, numChunks))
			})
			subReq := requestData
			subReq.Input = chunk
			data, err := ttsClient.GenerateSpeech(subReq)
			results[i] = data
			if err != nil {
				hasErr = true
				fyne.Do(func() {
					ui.HideProgressBar()
					ui.ShowError(fmt.Sprintf("TTS Generation failed: %v", err))
				})
				return
			}
			progress += y + float64(tokenCounts[i])
			fyne.Do(func() {
				ui.SetProgress(progress / total)
			})
		}

		if numChunks > 1 {
			progress += z * float64(numChunks)
			fyne.Do(func() {
				ui.SetProgress(progress / total)
				ui.SetProcessingMessage("Combining audio chunks...")
			})
		}

		if hasErr {
			return
		}

		var audioData []byte
		for _, blob := range results {
			audioData = append(audioData, blob...)
		}

		filename := util.GenerateFilename(inputText)
		fyne.Do(func() {
			ui.SetProcessingMessage("Saving audio file...")
		})
		savedPath, err := util.SaveAudioFile(audioData, filename)
		if err != nil {
			fyne.Do(func() {
				ui.HideProgressBar()
				ui.ShowError(fmt.Sprintf("Failed to save file: %v", err))
			})
			return
		}

		fyne.Do(func() {
			ui.HideProgressBar()
			ui.ShowSuccess(fmt.Sprintf("File saved to %s", filepath.Base(savedPath)))
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Success",
				Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(savedPath)),
			})
		})
	}(inputText, voice, speed)
}
