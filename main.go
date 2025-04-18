package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/joho/godotenv"
	"github.com/zalando/go-keyring"
)

const (
	keychainService = "OpenAI_TTS"
	keychainUser    = "api_token"
)

func main() {
	// Load environment variables from .env files if present
	loadEnvFiles()

	// Check for OPENAI_API_KEY environment variable or fallback to keychain
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey, _ = keyring.Get(keychainService, keychainUser)
	}

	a := app.New()
	w := a.NewWindow("OpenAI TTS Generator")

	// Set default window size to a standard macOS screen size
	w.Resize(fyne.NewSize(1440, 900))

	// Prefilled values
	defaultInstructions := `Du bist die Stimme eines deutschsprachigen Lern-Podcasts. Du erklärst Themen klar, ruhig und niedrigschwellig. Zielgruppe: Studierende des Fachs. Sprich natürlich, in einem zügigen, aber gelassenen Tempo. Vermeide jeden Eindruck von Roboter-Stimme oder Werbe-Sprech.

Sprechstil:
- Sprich zügig, aber ruhig – nicht gehetzt, nicht träge.
- Nutze natürliche Intonation: Betone Wichtiges etwas stärker, aber vermeide übertriebene Dynamik oder theatralische Betonung.
- Keine künstliche Fröhlichkeit. Klinge kompetent und freundlich, aber zurückhaltend.

Pausen:
- Setze kurze Pausen (ca. 0,5–1 Sekunde) nach Absätzen und wichtigen Aussagen, damit Hörer:innen mitdenken können.
- Füge längere Pausen zwischen Abschnitten oder Kapiteln ein.
- **Keine Pausen mitten im Satz**, auch wenn der Text dort einen Umbruch enthält.

Aussprache:
- Fremdwörter, Fachbegriffe und Abkürzungen klar und deutlich aussprechen.
- Bei Listen: Deutlich zählen („Erstens… Zweitens… Drittens…").
- Zwischenüberschriften leicht betonen („Kapitel 2: Datenanalyse").
- Beispiele dürfen einen leicht erzählenden Ton haben, um lebendiger zu wirken.

Hinweise zur Verarbeitung:
- Abschnitte in Lautschrift bitte vollständig überspringen – **nicht aussprechen**.
- Der Text ist Markdown-formatiert – **sprich die Markdown-Symbole nicht aus**, aber nutze sie, um die Rolle eines Text-Elements zu verstehen!`
	defaultVoice := "shimmer"
	defaultSpeed := 1.25
	defaultInput := "Dieser Text wird in Sprache umgewandelt. Ersetze ihn mit deinem eigenen Text."

	// Widgets
	instructions := widget.NewMultiLineEntry()
	instructions.SetText(defaultInstructions)
	voice := widget.NewEntry()
	voice.SetText(defaultVoice)
	speed := widget.NewSlider(0.5, 2.0)
	speed.Value = defaultSpeed
	speed.Step = 0.01
	speedLabel := widget.NewLabel(fmt.Sprintf("Speed: %.2f", speed.Value))
	speed.OnChanged = func(val float64) {
		speedLabel.SetText(fmt.Sprintf("Speed: %.2f", val))
	}
	input := widget.NewMultiLineEntry()
	input.SetText(defaultInput)

	// Status text with support for scrollable response
	responseText := widget.NewMultiLineEntry()
	responseText.Wrapping = fyne.TextWrapWord
	responseText.Disable()
	responseScroll := container.NewVScroll(responseText)
	responseScroll.SetMinSize(fyne.NewSize(600, 200))

	// Create red text for errors
	errorText := canvas.NewText("", color.RGBA{R: 255, G: 0, B: 0, A: 255})
	errorText.Hide()

	// Token management
	tokenEntry := widget.NewPasswordEntry()
	if apiKey != "" {
		tokenEntry.SetText("********")
	}

	// Submit button declaration (must be defined before the form)
	submitBtn := widget.NewButton("Submit", nil) // Function will be set after we create all UI elements

	// Layout
	formContainer := container.NewVBox(
		widget.NewLabel("Instructions:"), instructions,
		widget.NewLabel("Voice:"), voice,
		speedLabel, speed,
		widget.NewLabel("Input Text:"), input,
		submitBtn,
		responseScroll,
		errorText,
	)

	// Set the submit button callback now that we have formContainer
	submitBtn.OnTapped = func() {
		// Clean input
		clean := func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `"`, `\"`)
			s = strings.ReplaceAll(s, "\n", "\\n")
			s = strings.ReplaceAll(s, "'", "\\'")
			return s
		}
		payload := map[string]any{
			"model":        "gpt-4o-mini-tts",
			"instructions": clean(instructions.Text),
			"voice":        voice.Text,
			"speed":        speed.Value,
			"input":        clean(input.Text),
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		errorText.Hide()
		responseText.SetText("Submitting request...")
		responseText.Refresh()

		go func() {
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				showError(w, formContainer, errorText, fmt.Sprintf("Request failed: %v", err))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				b, _ := io.ReadAll(resp.Body)
				showError(w, formContainer, errorText, fmt.Sprintf("Error %d: %s\n%s", resp.StatusCode, resp.Status, string(b)))
				return
			}

			// Read response body
			b, _ := io.ReadAll(resp.Body)

			// Generate filename from input text
			words := strings.Fields(input.Text)
			filename := "Text_output.mp3"
			if len(words) >= 2 {
				// Take first two words, limit to 28 chars each
				w1, w2 := words[0], words[1]
				if len(w1) > 28 {
					w1 = w1[:28]
				}
				if len(w2) > 28 {
					w2 = w2[:28]
				}
				filename = fmt.Sprintf("Text_%s_%s.mp3", w1, w2)
			}

			// Save the file
			outPath := filepath.Join(os.Getenv("HOME"), "Downloads", filename)
			err = os.WriteFile(outPath, b, 0644)
			if err != nil {
				showError(w, formContainer, errorText, fmt.Sprintf("Failed to save file: %v", err))
				return
			}

			// Update UI in the main thread
			w.Canvas().Refresh(responseText)
			responseText.SetText(fmt.Sprintf("File saved successfully to: %s", outPath))

			// Show notification
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Success",
				Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(filename)),
			})
		}()
	}

	w.SetContent(formContainer)
	w.ShowAndRun()
}

func showError(w fyne.Window, content *fyne.Container, errorText *canvas.Text, msg string) {
	w.Canvas().Refresh(content)
	w.SetContent(content)
	errorText.Text = msg
	errorText.Refresh()
	errorText.Show()
}

func loadEnvFiles() {
	paths := []string{
		".env",
		filepath.Join(os.Getenv("HOME"), ".env"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			godotenv.Load(path)
		}
	}
}
