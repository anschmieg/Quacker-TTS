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
	instructions.SetText(defaultInstructions)
	instructions.SetMinRowsVisible(7)
	instrCont := container.NewStack(
		canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)),
		instructions,
	)

	voice := widget.NewEntry()
	voice.SetText(defaultVoice)
	voiceCell := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(120, voice.MinSize().Height)),
		voice,
	)
	voiceCont := voiceCell

	speed := widget.NewSlider(0.5, 2.0)
	speed.Value = defaultSpeed // Use defaultSpeed constant
	speed.Step = 0.01
	speedCell := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(120, speed.MinSize().Height)),
		speed,
	)
	speedCont := speedCell

	input := widget.NewMultiLineEntry()
	input.SetText(defaultInput)
	input.SetMinRowsVisible(7)

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
	speedLabel := canvas.NewText(fmt.Sprintf("%.2f", speed.Value), theme.Color(theme.ColorNameForeground))
	speedLabel.TextStyle = fyne.TextStyle{Bold: true}
	speedLabel.TextSize = 18
	speed.OnChanged = func(val float64) {
		speedLabel.Text = fmt.Sprintf("%.2f", val)
		speedLabel.Refresh()
	}
	inputLabel := canvas.NewText("Input Text:", theme.Color(theme.ColorNameForeground))
	inputLabel.TextStyle = fyne.TextStyle{Bold: true}
	inputLabel.TextSize = 18

	submitBtn := widget.NewButton("Submit", nil)

	voiceRow := container.NewHBox(voiceLabel, voiceCont)
	speedRow := container.NewHBox(speedCont, speedLabel)
	inputRow := container.NewVBox(container.NewHBox(inputLabel), input)
	btnRow := container.NewHBox(layout.NewSpacer(), submitBtn, layout.NewSpacer())

	formContainer := container.New(layout.NewVBoxLayout(),
		container.New(layout.NewPaddedLayout(), instrLabel),
		container.New(layout.NewPaddedLayout(), instrCont),
		container.New(layout.NewPaddedLayout(), voiceRow),
		container.New(layout.NewPaddedLayout(), speedRow),
		container.New(layout.NewPaddedLayout(), inputRow),
		container.New(layout.NewPaddedLayout(), btnRow),
		container.New(layout.NewPaddedLayout(), responseText),
		container.New(layout.NewPaddedLayout(), errorText),
	)

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
				showError(w, formContainer, errorText, fmt.Sprintf("Request failed: %v", err))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				b, _ := io.ReadAll(resp.Body)
				showError(w, formContainer, errorText, fmt.Sprintf("Error %d: %s\n%s", resp.StatusCode, resp.Status, string(b)))
				return
			}

			b, _ := io.ReadAll(resp.Body)
			filename := "output.mp3"
			outPath := filepath.Join(os.Getenv("HOME"), "Downloads", filename)
			err = os.WriteFile(outPath, b, 0644)
			if err != nil {
				showError(w, formContainer, errorText, fmt.Sprintf("Failed to save file: %v", err))
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
