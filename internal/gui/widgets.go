package gui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createInstructionsEntry creates the multi-line entry for instructions.
func createInstructionsEntry() *widget.Entry {
	instructions := widget.NewMultiLineEntry()
	instructions.Wrapping = fyne.TextWrapWord
	instructions.SetText(defaultInstructions)
	return instructions
}

// createProviderSelect creates the provider selection widget.
func createProviderSelect(providers []string, onChanged func(string)) *widget.Select {
	// Create widget without callback first
	providerSelect := widget.NewSelect(providers, nil)
	providerSelect.PlaceHolder = "Select TTS Provider"

	// Set callback after creation to avoid initialization issues
	providerSelect.OnChanged = func(provider string) {
		if onChanged != nil {
			onChanged(provider)
		}
	}

	return providerSelect
}

// createVoiceEntry creates the entry for the voice setting.
func createVoiceEntry() *widget.Entry {
	voice := widget.NewEntry()
	voice.SetText(defaultVoice)
	return voice
}

// createSpeedSlider creates the speed slider and its value label.
func createSpeedSlider() (*widget.Slider, *canvas.Text) {
	speed := widget.NewSlider(0.5, 2.0)
	speed.Value = defaultSpeed
	speed.Step = 0.01

	speedValueLabel := canvas.NewText(fmt.Sprintf("%.2f", speed.Value), theme.Color(theme.ColorNameForeground))
	speedValueLabel.TextStyle = fyne.TextStyle{Bold: true}
	speedValueLabel.TextSize = 18

	speed.OnChanged = func(val float64) {
		speedValueLabel.Text = fmt.Sprintf("%.2f", val)
		speedValueLabel.Refresh()
	}
	return speed, speedValueLabel
}

// createInputEntry creates the multi-line entry for the input text.
func createInputEntry() *widget.Entry {
	input := widget.NewMultiLineEntry()
	input.Wrapping = fyne.TextWrapWord
	input.SetText(defaultInput)
	return input
}

// createSubmitButton creates the main submit button.
func createSubmitButton(onTapped func()) *widget.Button {
	submitBtn := widget.NewButton("Submit", onTapped)
	submitBtn.Importance = widget.HighImportance
	return submitBtn
}

// createSuccessText creates the text element for success messages.
func createSuccessText() *canvas.Text {
	successText := canvas.NewText("", theme.Color(theme.ColorNamePrimary))
	successText.Alignment = fyne.TextAlignCenter
	successText.TextStyle = fyne.TextStyle{Bold: true}
	successText.Hide()
	return successText
}

// createErrorText creates the text element for error messages.
func createErrorText() *canvas.Text {
	errorText := canvas.NewText("", color.RGBA{R: 255, G: 0, B: 0, A: 255})
	errorText.Alignment = fyne.TextAlignLeading
	errorText.Hide()
	return errorText
}

// createProcessingText creates the text element for processing indication.
func createProcessingText() *canvas.Text {
	processingText := canvas.NewText("Processing...", theme.Color(theme.ColorNameForeground))
	processingText.Alignment = fyne.TextAlignCenter
	processingText.Hide()
	return processingText
}

// createLabel creates a standard text label.
func createLabel(text string, size float32, bold bool) *canvas.Text {
	label := canvas.NewText(text, theme.Color(theme.ColorNameForeground))
	label.TextSize = size
	label.TextStyle = fyne.TextStyle{Bold: bold}
	return label
}
