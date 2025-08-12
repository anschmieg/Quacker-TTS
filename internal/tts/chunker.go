package tts

import (
	"log"
	"regexp"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

var (
	sentenceEndRegex            = regexp.MustCompile(`([.!?])`)
	hrSeparatorRegex            = regexp.MustCompile(`\n(?:-{3,}|_{3,})\n`)
	multiNewlineSeparatorRegex  = regexp.MustCompile(`\n\s*\n`)
	sentenceEndNewlineRegex     = regexp.MustCompile(`([.!?])\s*\n`)
)

// Default chunking limits
const (
	DefaultTokenLimit = 2000 // OpenAI: tokens per chunk
	DefaultByteLimit  = 4500 // Google: bytes per chunk
)

// getInitialChunks splits text by major separators like horizontal rules or multiple newlines.
func GetInitialChunks(text string) []string {
	hrParts := hrSeparatorRegex.Split(text, -1)
	var filteredHrParts []string
	for _, p := range hrParts {
		if strings.TrimSpace(p) != "" {
			filteredHrParts = append(filteredHrParts, p)
		}
	}
	if len(filteredHrParts) > 1 {
		log.Printf("Split text by horizontal rule into %d major chunks.", len(filteredHrParts))
		return filteredHrParts
	}

	multiNewlineParts := multiNewlineSeparatorRegex.Split(text, -1)
	var filteredMultiNewlineParts []string
	for _, p := range multiNewlineParts {
		if strings.TrimSpace(p) != "" {
			filteredMultiNewlineParts = append(filteredMultiNewlineParts, p)
		}
	}
	if len(filteredMultiNewlineParts) > 1 {
		log.Printf("Split text by multiple newlines into %d major chunks.", len(filteredMultiNewlineParts))
		return filteredMultiNewlineParts
	}

	log.Println("No major separators found; treating text as a single chunk.")
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return []string{}
	}
	return []string{trimmedText}
}

// ----------- TOKEN-BASED CHUNKING (OpenAI) -----------

// splitTextTokenLimit splits text into chunks based on token limits.
func SplitTextTokenLimit(text, model string, maxTokens int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Error getting tokenizer encoding, falling back to rune splitting: %v", err)
		return splitByRune(text, maxTokens*3)
	}

	initialChunks := GetInitialChunks(text)
	var finalChunks []string
	for _, chunk := range initialChunks {
		if len(enc.Encode(chunk, nil, nil)) <= maxTokens {
			finalChunks = append(finalChunks, chunk)
		} else {
			log.Printf("Major chunk exceeds token limit, applying recursive splitting...")
			finalChunks = append(finalChunks, splitChunkRecursively(chunk, enc, maxTokens, 0)...)
		}
	}
	return finalChunks
}

func splitChunkRecursively(chunk string, enc *tiktoken.Tiktoken, maxTokens int, level int) []string {
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return nil
	}

	var regex *regexp.Regexp
	switch level {
	case 0:
		regex = sentenceEndNewlineRegex
	case 1:
		regex = sentenceEndRegex
	case 2:
		return splitByWord(chunk, enc, maxTokens)
	default:
		return splitByRune(chunk, maxTokens*3)
	}

	indices := regex.FindAllStringIndex(chunk, -1)
	if len(indices) == 0 {
		return splitChunkRecursively(chunk, enc, maxTokens, level+1)
	}

	var resultChunks []string
	lastPos := 0
	var currentChunk strings.Builder

	for _, idx := range indices {
		segment := chunk[lastPos:idx[1]]
		segment = strings.TrimSpace(segment)
		if segment == "" {
			lastPos = idx[1]
			continue
		}
		if len(enc.Encode(currentChunk.String()+segment, nil, nil)) > maxTokens {
			if currentChunk.Len() > 0 {
				resultChunks = append(resultChunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		currentChunk.WriteString(segment)
		lastPos = idx[1]
	}
	// Add any trailing text
	if lastPos < len(chunk) {
		tail := chunk[lastPos:]
		if len(enc.Encode(currentChunk.String()+tail, nil, nil)) > maxTokens {
			if currentChunk.Len() > 0 {
				resultChunks = append(resultChunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		currentChunk.WriteString(tail)
	}
	if currentChunk.Len() > 0 {
		resultChunks = append(resultChunks, currentChunk.String())
	}
	return resultChunks
}

func splitByWord(text string, enc *tiktoken.Tiktoken, maxTokens int) []string {
	var chunks []string
	var currentChunk strings.Builder
	words := strings.Fields(text)

	for _, word := range words {
		if len(enc.Encode(word, nil, nil)) > maxTokens {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
			chunks = append(chunks, splitByRune(word, maxTokens*3)...)
			continue
		}
		if len(enc.Encode(currentChunk.String()+" "+word, nil, nil)) > maxTokens {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}

func splitByRune(text string, maxRunes int) []string {
	var chunks []string
	runes := []rune(text)
	for i := 0; i < len(runes); i += maxRunes {
		end := i + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// ----------- BYTE-BASED CHUNKING (Google) -----------

// splitTextByteLimit splits text into chunks based on byte limits.
func SplitTextByteLimit(text string, maxBytes int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	initialChunks := GetInitialChunks(text)
	var finalChunks []string
	for _, chunk := range initialChunks {
		if len([]byte(chunk)) <= maxBytes {
			finalChunks = append(finalChunks, chunk)
		} else {
			log.Printf("Major chunk exceeds byte limit, applying recursive splitting...")
			finalChunks = append(finalChunks, splitChunkRecursivelyBytes(chunk, maxBytes, 0)...)
		}
	}
	return finalChunks
}

func splitChunkRecursivelyBytes(chunk string, maxBytes int, level int) []string {
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return nil
	}

	var regex *regexp.Regexp
	switch level {
	case 0:
		regex = sentenceEndNewlineRegex
	case 1:
		regex = sentenceEndRegex
	case 2:
		return splitByWordBytes(chunk, maxBytes)
	default:
		return splitByRuneBytes(chunk, maxBytes)
	}

	indices := regex.FindAllStringIndex(chunk, -1)
	if len(indices) == 0 {
		return splitChunkRecursivelyBytes(chunk, maxBytes, level+1)
	}

	var resultChunks []string
	lastPos := 0
	var currentChunk strings.Builder

	for _, idx := range indices {
		segment := chunk[lastPos:idx[1]]
		segment = strings.TrimSpace(segment)
		if segment == "" {
			lastPos = idx[1]
			continue
		}
		if len([]byte(currentChunk.String()+segment)) > maxBytes {
			if currentChunk.Len() > 0 {
				resultChunks = append(resultChunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		currentChunk.WriteString(segment)
		lastPos = idx[1]
	}
	// Add any trailing text
	if lastPos < len(chunk) {
		tail := chunk[lastPos:]
		if len([]byte(currentChunk.String()+tail)) > maxBytes {
			if currentChunk.Len() > 0 {
				resultChunks = append(resultChunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		currentChunk.WriteString(tail)
	}
	if currentChunk.Len() > 0 {
		resultChunks = append(resultChunks, currentChunk.String())
	}
	return resultChunks
}

func splitByWordBytes(text string, maxBytes int) []string {
	var chunks []string
	var currentChunk strings.Builder
	words := strings.Fields(text)

	for _, word := range words {
		if len([]byte(word)) > maxBytes {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
			chunks = append(chunks, splitByRuneBytes(word, maxBytes)...)
			continue
		}
		if len([]byte(currentChunk.String()+" "+word)) > maxBytes {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}

func splitByRuneBytes(text string, maxBytes int) []string {
	var chunks []string
	var currentChunk strings.Builder
	for _, r := range text {
		if len([]byte(currentChunk.String()+string(r))) > maxBytes {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		currentChunk.WriteRune(r)
	}
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}
