package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI holds all the UI elements and state.
type UI struct {
	Window          fyne.Window
	Instructions    *widget.Entry
	ProviderSelect  *widget.Select
	Voice           *widget.Entry
	Speed           *widget.Slider
	Input           *widget.Entry
	SubmitBtn       *widget.Button
	SuccessText     *canvas.Text
	ErrorText       *canvas.Text
	ProcessingText  *canvas.Text
	SpeedValueLabel *canvas.Text

	ProgressBar *widget.ProgressBar // Progress bar for TTS progress
}

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
	defaultVoice = "shimmer"
	defaultSpeed = 1.125
	defaultInput = "Dieser Text wird in Sprache umgewandelt. Ersetze ihn mit deinem eigenen Text."
)

// NewUI creates and lays out the main application window and its widgets.
func NewUI(app fyne.App, providers []string, onSubmit func(), onSettings func(), onProviderChange func(string)) *UI {
	w := app.NewWindow("Quacker – Text to Speech")
	w.Resize(fyne.NewSize(900, 600))

	// Add Preferences menu item under Quacker menu
	menu := fyne.NewMainMenu(
		fyne.NewMenu("Quacker",
			fyne.NewMenuItem("Preferences", onSettings),
		),
	)
	w.SetMainMenu(menu)

	ui := &UI{Window: w}

	// Create Widgets (using functions from widgets.go)
	ui.Instructions = createInstructionsEntry()
	ui.ProviderSelect = createProviderSelect(providers, onProviderChange)
	// Create and wrap the voice entry in a fixed-size container to make it wider
	voiceEntry := createVoiceEntry()
	voiceMin := voiceEntry.MinSize()
	voiceContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(300, voiceMin.Height)), voiceEntry)
	ui.Voice = voiceEntry
	ui.Speed, ui.SpeedValueLabel = createSpeedSlider()
	ui.Input = createInputEntry()
	ui.SubmitBtn = createSubmitButton(onSubmit)
	ui.SubmitBtn.Resize(fyne.NewSize(200, 40)) // Make submit button wider
	// Settings button in bottom left (commented out)
	// settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), onSettings)
	settingsBtnTopRight := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), onSettings)
	ui.SuccessText = createSuccessText()
	ui.ErrorText = createErrorText()
	ui.ProcessingText = createProcessingText()
	ui.ProgressBar = widget.NewProgressBar()
	ui.ProgressBar.Hide()

	// Layout
	instrCont := container.NewScroll(ui.Instructions)
	inputCont := container.NewScroll(ui.Input)

	instrLabel := createLabel("Instructions:", 18, true)
	providerLabel := createLabel("Provider:", 18, true)
	voiceLabel := createLabel("Voice:", 18, true)
	// speedTextLabel := createLabel("Speed:", 18, true) // COMMENTED OUT
	inputLabel := createLabel("Input Text:", 18, true)

	// Replace grid layout with HBox for right-alignment
	providerVoiceRow := container.NewHBox(
		providerLabel,
		ui.ProviderSelect,
		layout.NewSpacer(),
		voiceLabel,
		voiceContainer,
		layout.NewSpacer(),
		settingsBtnTopRight,
	)

	// Settings on left, submit button centered in window using 3-column layout
	btnRow := container.NewGridWithColumns(3,
		// settingsBtn, // COMMENTED OUT (bottom left)
		layout.NewSpacer(), // visually balances the settings button
		container.NewCenter(ui.SubmitBtn),
		layout.NewSpacer(),
	)

	instrGroup := container.NewBorder(instrLabel, nil, nil, nil, instrCont)
	inputGroup := container.NewBorder(inputLabel, nil, nil, nil, inputCont)

	separatorLine := canvas.NewRectangle(theme.Color(theme.ColorNameInputBorder))
	separatorLine.SetMinSize(fyne.NewSize(0, 1))
	topSection := container.NewVBox(
		providerVoiceRow,
		separatorLine,
	)
	bottomSection := container.NewVBox(
		btnRow,
		ui.ProgressBar, // Progress bar appears above messages
		ui.ProcessingText,
		ui.SuccessText,
		ui.ErrorText,
	)

	textSplit := container.NewVSplit(instrGroup, inputGroup)
	textSplit.Offset = 0.4

	content := container.NewBorder(topSection, bottomSection, nil, nil, textSplit)

	w.SetContent(content)

	return ui
}

// ShowError displays an error message in the UI.
func (ui *UI) ShowError(msg string) {
	ui.ProcessingText.Hide()
	ui.ErrorText.Text = msg
	ui.ErrorText.Show()
	ui.SuccessText.Hide()
	ui.ErrorText.Refresh()
}

// ShowSuccess displays a success message in the UI.
func (ui *UI) ShowSuccess(msg string) {
	ui.ProcessingText.Hide()
	ui.SuccessText.Text = msg
	ui.SuccessText.Show()
	ui.ErrorText.Hide()
	ui.SuccessText.Refresh()
}

// ShowProcessing displays the processing indicator.
func (ui *UI) ShowProcessing() {
	ui.ProcessingText.Show()
	ui.SuccessText.Hide()
	ui.ErrorText.Hide()
	ui.ProcessingText.Refresh()
}

// SetProcessingMessage updates the processing text field with a status message.
func (ui *UI) SetProcessingMessage(msg string) {
	ui.ProcessingText.Text = msg
	ui.ProcessingText.Show()
	ui.ProcessingText.Refresh()
	ui.SuccessText.Hide()
	ui.ErrorText.Hide()
}

// SetSubmitEnabled enables or disables the submit button.
func (ui *UI) SetSubmitEnabled(enabled bool) {
	if enabled {
		ui.SubmitBtn.Enable()
	} else {
		ui.SubmitBtn.Disable()
	}
}

// ShowProgressBar displays the progress bar and hides messages.
func (ui *UI) ShowProgressBar() {
	ui.ProgressBar.Show()
	ui.ProcessingText.Hide()
	ui.SuccessText.Hide()
	ui.ErrorText.Hide()
	ui.ProgressBar.Refresh()
}

// HideProgressBar hides the progress bar.
func (ui *UI) HideProgressBar() {
	ui.ProgressBar.Hide()
	ui.ProgressBar.Refresh()
}

// SetProgress sets the progress bar value (0.0 to 1.0).
func (ui *UI) SetProgress(value float64) {
	ui.ProgressBar.SetValue(value)
	ui.ProgressBar.Refresh()
}
