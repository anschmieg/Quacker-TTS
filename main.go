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
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/joho/godotenv"
	"github.com/zalando/go-keyring"
)

const (
	defaultInstructions = `Du bist die Stimme eines deutschsprachigen Lern-Podcasts. Du erklärst Themen klar, ruhig und niedrigschwellig. Zielgruppe: Studierende des Fachs. Sprich natürlich, in einem zügigen, aber gelassenen Tempo. Vermeide jeden Eindruck von Roboter-Stimme oder Werbe-Sprech.
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
	defaultVoice    = "shimmer"
	defaultSpeed    = 1.25
	defaultInput    = "Dieser Text wird in Sprache umgewandelt. Ersetze ihn mit deinem eigenen Text."
	keychainService = "OpenAI_TTS"
	keychainUser    = "api_token"
	openAIAPIURL    = "https://api.openai.com/v1/audio/speech"
)

func main() {
	loadEnvFiles()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey, _ = keyring.Get(keychainService, keychainUser)
	}
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY not found in environment variables or keychain.")
		fmt.Println("Please set the OPENAI_API_KEY environment variable or store it in the keychain using:")
		fmt.Printf("keyring.Set(\"%s\", \"%s\", \"YOUR_API_KEY\")\n", keychainService, keychainUser)
	}

	a := app.New()
	w := a.NewWindow("Quacker – Text to Speech")
	w.Resize(fyne.NewSize(900, 600))

	instructions := widget.NewMultiLineEntry()
	instructions.Wrapping = fyne.TextWrapWord
	instructions.SetText(defaultInstructions)
	instrCont := container.NewScroll(instructions)

	voice := widget.NewEntry()
	voice.SetText(defaultVoice)

	speed := widget.NewSlider(0.5, 2.0)
	speed.Value = defaultSpeed
	speed.Step = 0.01

	input := widget.NewMultiLineEntry()
	input.Wrapping = fyne.TextWrapWord
	input.SetText(defaultInput)
	inputCont := container.NewScroll(input)

	successText := canvas.NewText("", theme.Color(theme.ColorNamePrimary))
	successText.Alignment = fyne.TextAlignCenter
	successText.TextStyle = fyne.TextStyle{Bold: true}
	successText.Hide()

	errorText := canvas.NewText("", color.RGBA{R: 255, G: 0, B: 0, A: 255})
	errorText.Alignment = fyne.TextAlignLeading
	errorText.Hide()

	processingText := canvas.NewText("Processing...", theme.Color(theme.ColorNameForeground))
	processingText.Alignment = fyne.TextAlignCenter
	processingText.Hide()

	instrLabel := canvas.NewText("Instructions:", theme.Color(theme.ColorNameForeground))
	instrLabel.TextStyle = fyne.TextStyle{Bold: true}
	instrLabel.TextSize = 18
	voiceLabel := canvas.NewText("Voice:", theme.Color(theme.ColorNameForeground))
	voiceLabel.TextStyle = fyne.TextStyle{Bold: true}
	voiceLabel.TextSize = 18
	speedTextLabel := canvas.NewText("Speed:", theme.Color(theme.ColorNameForeground))
	speedTextLabel.TextStyle = fyne.TextStyle{Bold: true}
	speedTextLabel.TextSize = 18
	speedValueLabel := canvas.NewText(fmt.Sprintf("%.2f", speed.Value), theme.Color(theme.ColorNameForeground))
	speedValueLabel.TextStyle = fyne.TextStyle{Bold: true}
	speedValueLabel.TextSize = 18
	speed.OnChanged = func(val float64) {
		speedValueLabel.Text = fmt.Sprintf("%.2f", val)
		speedValueLabel.Refresh()
	}
	inputLabel := canvas.NewText("Input Text:", theme.Color(theme.ColorNameForeground))
	inputLabel.TextStyle = fyne.TextStyle{Bold: true}
	inputLabel.TextSize = 18

	submitBtn := widget.NewButton("Submit", nil)
	submitBtn.Importance = widget.HighImportance

	voiceSpeedRow := container.New(layout.NewGridLayout(6),
		voiceLabel,
		voice,
		layout.NewSpacer(),
		speedTextLabel,
		speed,
		speedValueLabel,
	)

	btnRow := container.NewHBox(layout.NewSpacer(), submitBtn, layout.NewSpacer())

	instrGroup := container.NewBorder(instrLabel, nil, nil, nil, instrCont)
	inputGroup := container.NewBorder(inputLabel, nil, nil, nil, inputCont)

	separatorLine := canvas.NewRectangle(theme.Color(theme.ColorNameInputBorder))
	separatorLine.SetMinSize(fyne.NewSize(0, 1))
	topSection := container.NewVBox(
		voiceSpeedRow,
		separatorLine,
	)
	bottomSection := container.NewVBox(
		btnRow,
		processingText,
		successText,
		errorText,
	)

	textSplit := container.NewVSplit(instrGroup, inputGroup)
	textSplit.Offset = 0.4

	content := container.NewBorder(topSection, bottomSection, nil, nil, textSplit)

	submitBtn.OnTapped = func() {
		if apiKey == "" {
			showError(w, content, errorText, successText, processingText, "Error: API Key not configured. Set OPENAI_API_KEY environment variable or store in keychain.")
			return
		}
		clean := func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `"`, `\"`)
			s = strings.ReplaceAll(s, "\n", "\\n")
			s = strings.ReplaceAll(s, "'", "\\'")
			return s
		}
		payload := map[string]any{
			"model":           "tts-1",
			"voice":           voice.Text,
			"speed":           speed.Value,
			"input":           clean(input.Text),
			"response_format": "mp3",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", openAIAPIURL, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		// Disable button and show processing message
		submitBtn.Disable()
		processingText.Show()
		successText.Hide()
		errorText.Hide()
		processingText.Refresh()
		successText.Refresh()
		errorText.Refresh()

		go func() {
			// Ensure button is re-enabled when goroutine finishes
			defer func() {
				submitBtn.Enable()
			}()

			client := &http.Client{}
			resp, err := client.Do(req)

			if err != nil {
				showError(w, content, errorText, successText, processingText, fmt.Sprintf("Request failed: %v", err))
				return // Button will be re-enabled by defer
			}
			defer resp.Body.Close()

			b, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				showError(w, content, errorText, successText, processingText, fmt.Sprintf("Failed to read response body: %v", readErr))
				return // Button will be re-enabled by defer
			}

			if resp.StatusCode != http.StatusOK {
				errMsg := "An error occurred:"
				if len(b) > 0 {
					var prettyJSON bytes.Buffer
					if json.Indent(&prettyJSON, b, "", "  ") == nil {
						errMsg += "\n" + prettyJSON.String()
					} else {
						errMsg += "\n" + string(b)
					}
				} else {
					errMsg += " " + resp.Status
				}
				showError(w, content, errorText, successText, processingText, errMsg)
				return // Button will be re-enabled by defer
			}

			words := strings.Fields(input.Text)
			filename := "Text_output.mp3"
			if len(words) >= 2 {
				sanitize := func(word string) string {
					reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
					sanitized := reg.ReplaceAllString(word, "_")
					if len(sanitized) > 28 {
						return sanitized[:28]
					}
					return sanitized
				}
				w1, w2 := sanitize(words[0]), sanitize(words[1])
				filename = fmt.Sprintf("Text_%s_%s.mp3", w1, w2)
			} else if len(words) == 1 {
				sanitize := func(word string) string {
					reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
					sanitized := reg.ReplaceAllString(word, "_")
					if len(sanitized) > 28 {
						return sanitized[:28]
					}
					return sanitized
				}
				w1 := sanitize(words[0])
				filename = fmt.Sprintf("Text_%s.mp3", w1)
			}

			homeDir, homeErr := os.UserHomeDir()
			if homeErr != nil {
				showError(w, content, errorText, successText, processingText, fmt.Sprintf("Failed to get home directory: %v", homeErr))
				return // Button will be re-enabled by defer
			}
			outPath := filepath.Join(homeDir, "Downloads", filename)

			err = os.WriteFile(outPath, b, 0644)
			if err != nil {
				showError(w, content, errorText, successText, processingText, fmt.Sprintf("Failed to save file: %v", err))
				return // Button will be re-enabled by defer
			}

			processingText.Hide()
			successText.Text = "File saved to Downloads"
			successText.Show()
			errorText.Hide()
			processingText.Refresh()
			successText.Refresh()
			errorText.Refresh()

			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Success",
				Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(filename)),
			})
		}()
	}

	w.SetContent(content)
	w.ShowAndRun()
}

func showError(_ fyne.Window, _ fyne.CanvasObject, errorText *canvas.Text, successText *canvas.Text, processingText *canvas.Text, msg string) {
	processingText.Hide()
	errorText.Text = msg
	errorText.Show()
	successText.Hide()
	processingText.Refresh()
	errorText.Refresh()
	successText.Refresh()
}

func loadEnvFiles() {
	godotenv.Load()

	homeDir, err := os.UserHomeDir()
	if err == nil {
		godotenv.Load(filepath.Join(homeDir, ".env"))
	}
}
