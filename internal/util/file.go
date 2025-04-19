package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateFilename creates a filename based on the first few words of the input text.
func GenerateFilename(inputText string) string {
	words := strings.Fields(inputText)
	filename := "Text_output.mp3"
	if len(words) >= 2 {
		w1, w2 := SanitizeFilenameWord(words[0]), SanitizeFilenameWord(words[1])
		filename = fmt.Sprintf("Text_%s_%s.mp3", w1, w2)
	} else if len(words) == 1 {
		w1 := SanitizeFilenameWord(words[0])
		filename = fmt.Sprintf("Text_%s.mp3", w1)
	}
	return filename
}

// SaveAudioFile saves the audio data to the Downloads directory.
func SaveAudioFile(data []byte, filename string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	outPath := filepath.Join(homeDir, "Downloads", filename)

	err = os.WriteFile(outPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save file to %s: %w", outPath, err)
	}
	return outPath, nil
}
