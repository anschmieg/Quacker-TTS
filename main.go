package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	defaultVoice = "echo"
	defaultSpeed = 1.0 // Restore defaultSpeed constant
	defaultInput = "Hello, world! This is a test of the text-to-speech system."
	apiURL       = "http://localhost:11434/api/tts"
)

func main() {
	a := app.New()
	w := a.NewWindow("Easy TTS")
	w.Resize(fyne.NewSize(900, 600))

	// Widgets
	instructions := widget.NewMultiLineEntry()
	instructions.Wrapping = fyne.TextWrapWord
	instructions.SetText(defaultInstructions)
	instrCont := container.NewScroll(instructions) // Scrollable container

	voice := widget.NewEntry()
	voice.SetText(defaultVoice)

	speed := widget.NewSlider(0.5, 2.0)
	speed.Value = defaultSpeed // Use defaultSpeed constant
	speed.Step = 0.01

	input := widget.NewMultiLineEntry()
	input.Wrapping = fyne.TextWrapWord
	input.SetText(defaultInput)
	inputCont := container.NewScroll(input) // Scrollable container

	responseText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	responseText.TextStyle = fyne.TextStyle{Bold: true}
	responseText.TextSize = 20
	responseText.Alignment = fyne.TextAlignCenter

	errorText := canvas.NewText("", theme.Color(theme.ColorNameError))
	errorText.Hide()

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

	// Use a 6-column grid layout for more precise control
	voiceSpeedRow := container.New(layout.NewGridLayout(6),
		voiceLabel,         // Col 1 (~16.7%)
		voice,              // Col 2 (~16.7%)
		layout.NewSpacer(), // Col 3 (~16.7%) - Spacer for padding
		speedTextLabel,     // Col 4 (~16.7%)
		speed,              // Col 5 (~16.7%)
		speedValueLabel,    // Col 6 (~16.7%)
	)

	btnRow := container.NewHBox(layout.NewSpacer(), submitBtn, layout.NewSpacer())

	instrCont.SetMinSize(fyne.NewSize(0, 150)) // Give it a minimum height
	inputCont.SetMinSize(fyne.NewSize(0, 150)) // Give it a minimum height

	topSection := container.NewVBox(
		instrLabel,
		voiceSpeedRow,
		inputLabel,
	)
	bottomSection := container.NewVBox(
		btnRow,
		responseText,
		errorText,
	)

	textSplit := container.NewVSplit(instrCont, inputCont)
	textSplit.Offset = 0.4 // Initial split ratio

	content := container.NewBorder(topSection, bottomSection, nil, nil, textSplit)

	submitBtn.OnTapped = func() {
		clean := func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `"`, `\"`)
			s = strings.ReplaceAll(s, "\n", "\\n")
			s = strings.ReplaceAll(s, "'", "\\'")
			return s
		}
		payload := map[string]any{
			"instructions": clean(instructions.Text),
			"voice":        voice.Text,
			"speed":        speed.Value,
			"input":        clean(input.Text),
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		errorText.Hide()
		responseText.Text = "Submitting request..."
		responseText.Refresh()

		go func() {
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				showError(w, content, errorText, fmt.Sprintf("Request failed: %v", err))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				b, _ := io.ReadAll(resp.Body)
				showError(w, content, errorText, fmt.Sprintf("Error %d: %s\n%s", resp.StatusCode, resp.Status, string(b)))
				return
			}

			b, _ := io.ReadAll(resp.Body)
			filename := "output.mp3"
			outPath := filepath.Join(os.Getenv("HOME"), "Downloads", filename)
			err = os.WriteFile(outPath, b, 0644)
			if err != nil {
				showError(w, content, errorText, fmt.Sprintf("Failed to save file: %v", err))
				return
			}

			responseText.Text = fmt.Sprintf("File saved successfully to: %s", outPath)
			responseText.Refresh()

			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Success",
				Content: fmt.Sprintf("Audio saved to: %s", filepath.Base(filename)),
			})
		}()
	}

	w.SetContent(content)
	w.ShowAndRun()
}

func showError(w fyne.Window, content *fyne.Container, errorText *canvas.Text, msg string) {
	w.Canvas().Refresh(content)
	w.SetContent(content)
	errorText.Text = msg
	errorText.Refresh()
	errorText.Show()
}
