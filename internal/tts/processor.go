package tts

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// ProgressCallback is called after each successful chunk or sub-chunk.
type ProgressCallback func(completed, total int)

// ErrorCallback is called to display user-friendly errors.
type ErrorCallback func(msg string)

// ProcessorConfig allows tuning of chunking and retry parameters.
type ProcessorConfig struct {
	MinChunkBytes      int           // Minimum chunk size for fallback (bytes)
	ChunkDelay         time.Duration // Delay between chunk requests
	MaxRetries         int           // Retries per chunk
	GoogleFallbackVoices []string    // Optional: override fallback voices for Google
}

// DefaultProcessorConfig returns a sensible default config.
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		MinChunkBytes:      1, // one word
		ChunkDelay:         2 * time.Second,
		MaxRetries:         3,
		GoogleFallbackVoices: nil, // use dynamic logic
	}
}

// ProcessTextToSpeech handles chunking, retry, fallback, and error logic for TTS.
// Returns the concatenated audio or error.
func ProcessTextToSpeech(
	ctx context.Context,
	provider Provider,
	request *UnifiedRequest,
	progressCb ProgressCallback,
	errorCb ErrorCallback,
	cfg *ProcessorConfig,
) ([]byte, error) {
	if cfg == nil {
		cfg = DefaultProcessorConfig()
	}
	isGoogle := provider.GetName() == "google"
	var chunks []string
	if isGoogle {
		chunks = SplitTextByteLimit(request.Text, DefaultByteLimit)
	} else {
		chunks = SplitTextTokenLimit(request.Text, "cl100k_base", provider.GetMaxTokensPerChunk())
	}
	totalChunks := len(chunks)
	var audioData []byte
	completed := 0

	for _, chunk := range chunks {
		data, err := processChunkRecursively(
			ctx, provider, request, chunk, isGoogle,
			cfg.MinChunkBytes, cfg.MaxRetries, cfg.GoogleFallbackVoices,
			func() {
				completed++
				if progressCb != nil {
					progressCb(completed, totalChunks)
				}
			},
			errorCb,
		)
		if err != nil {
			// Error already reported via errorCb, continue to next chunk
			continue
		}
		audioData = append(audioData, data...)
	}
	return audioData, nil
}

// --- Internal helpers ---

// processChunkRecursively handles chunking, retry, fallback, and error chunk insertion for a single chunk.
func processChunkRecursively(
	ctx context.Context,
	provider Provider,
	request *UnifiedRequest,
	chunk string,
	isGoogle bool,
	minLimit int,
	maxRetries int,
	googleFallbackVoices []string,
	progressCb func(),
	errorCb ErrorCallback,
) ([]byte, error) {
	return processChunkRecursivelyWithDepth(ctx, provider, request, chunk, isGoogle, minLimit, maxRetries, googleFallbackVoices, progressCb, errorCb, 0, len([]byte(chunk)))
}

// Helper with recursion depth and previous chunk size tracking
func processChunkRecursivelyWithDepth(
	ctx context.Context,
	provider Provider,
	request *UnifiedRequest,
	chunk string,
	isGoogle bool,
	minLimit int,
	maxRetries int,
	googleFallbackVoices []string,
	progressCb func(),
	errorCb ErrorCallback,
	recursionLevel int,
	prevChunkBytes int,
) ([]byte, error) {
	var data []byte
	var err error
	origVoice := request.Voice
	origLang := extractLangCode(origVoice)
	words := strings.Fields(chunk)
	chunkBytes := len([]byte(chunk))

	// --- DEBUG LOGGING ---
	log.Printf("[TTS DEBUG] processChunkRecursively: chunkBytes=%d, len(words)=%d, chunk='%.60s...', minLimit=%d, recursionLevel=%d", chunkBytes, len(words), chunk, minLimit, recursionLevel)
	if ctx.Err() != nil {
		log.Printf("[TTS DEBUG] Context done in processChunkRecursively: %v", ctx.Err())
		return nil, ctx.Err()
	}
	// Recursion depth guard
	if recursionLevel > 20 {
		log.Printf("[TTS DEBUG] Recursion depth exceeded for chunk (len=%d): %.60s...", chunkBytes, chunk)
		if errorCb != nil {
			errorCb(fmt.Sprintf("Chunk recursion depth exceeded (%.40s...). Aborting this section.", chunk))
		}
		return nil, fmt.Errorf("recursion depth exceeded")
	}

	// 1. Normal attempts with exponential backoff on error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[TTS DEBUG] Attempt %d/%d for chunk (len=%d): %.60s...", attempt, maxRetries, chunkBytes, chunk)
		data, err = provider.GenerateSpeech(ctx, &UnifiedRequest{
			Text:   chunk,
			Voice:  request.Voice,
			Speed:  request.Speed,
			Format: request.Format,
			Model:  request.Model,
		})
		if err == nil {
			if progressCb != nil {
				progressCb()
			}
			log.Printf("[TTS DEBUG] Success for chunk (len=%d): %.60s...", chunkBytes, chunk)
			return data, nil
		}
		log.Printf("[TTS DEBUG] Error on attempt %d: %v", attempt, err)
		if attempt < maxRetries && isRetryableTTS(err) {
			if isQuotaOrRateError(err) && errorCb != nil {
				errorCb("Google TTS may be rate-limiting or throttling your requests. Waiting before retrying...")
			}
			delay := getBackoffDelay(attempt)
			log.Printf("[TTS DEBUG] Waiting %v before retrying...", delay)
			time.Sleep(delay)
			continue
		}
		break
	}

	// 2. Sub-chunking if possible
	if chunkBytes > minLimit && len(words) > 1 {
		log.Printf("[TTS DEBUG] Sub-chunking chunk (len=%d): %.60s...", chunkBytes, chunk)
		var subChunks []string
		if isGoogle {
			subChunks = SplitTextByteLimit(chunk, chunkBytes/2)
		} else {
			subChunks = SplitTextTokenLimit(chunk, "cl100k_base", provider.GetMaxTokensPerChunk()/2)
		}
		log.Printf("[TTS DEBUG] Sub-chunked into %d sub-chunks", len(subChunks))

		// If chunk cannot be split further (only one sub-chunk, same size), treat as minimum-size chunk
		if len(subChunks) == 1 && len([]byte(subChunks[0])) == chunkBytes {
			log.Printf("[TTS DEBUG] Sub-chunking did not reduce chunk size. Treating as minimum-size chunk.")
			goto MIN_CHUNK_LOGIC
		}

		var audio []byte
		for i, sub := range subChunks {
			log.Printf("[TTS DEBUG] Processing sub-chunk %d/%d (len=%d): %.60s...", i+1, len(subChunks), len([]byte(sub)), sub)
			subData, subErr := processChunkRecursivelyWithDepth(ctx, provider, request, sub, isGoogle, minLimit, maxRetries, googleFallbackVoices, progressCb, errorCb, recursionLevel+1, chunkBytes)
			if subErr != nil {
				log.Printf("[TTS DEBUG] Error in sub-chunk %d/%d: %v", i+1, len(subChunks), subErr)
				// Error already reported, continue to next sub-chunk
				continue
			}
			audio = append(audio, subData...)
		}
		if len(audio) > 0 {
			log.Printf("[TTS DEBUG] Returning audio from sub-chunks for parent chunk (len=%d)", chunkBytes)
			return audio, nil
		}
		log.Printf("[TTS DEBUG] All sub-chunks failed for parent chunk (len=%d)", chunkBytes)
	}

MIN_CHUNK_LOGIC:
	// 3. If chunk is a single word and <200 bytes, or chunk cannot be split further, treat as minimum-size chunk
	if len(words) == 1 && chunkBytes < 200 || chunkBytes <= minLimit {
		log.Printf("[TTS DEBUG] Minimum-size chunk logic triggered (len=%d): %.60s...", chunkBytes, chunk)
		sanitized := sanitizeWordForTTS(chunk)
		if sanitized != chunk && sanitized != "" {
			log.Printf("[TTS DEBUG] Trying sanitized word: %s", sanitized)
			data, err = provider.GenerateSpeech(ctx, &UnifiedRequest{
				Text:   sanitized,
				Voice:  request.Voice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				log.Printf("[TTS DEBUG] Success with sanitized word.")
				return data, nil
			}
			log.Printf("[TTS DEBUG] Sanitized word failed: %v", err)
		}
		// Try stripping Markdown and retry once more
		mdStripped := stripMarkdown(chunk)
		if mdStripped != chunk && mdStripped != "" {
			log.Printf("[TTS DEBUG] Trying Markdown-stripped word: %s", mdStripped)
			data, err = provider.GenerateSpeech(ctx, &UnifiedRequest{
				Text:   mdStripped,
				Voice:  request.Voice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				log.Printf("[TTS DEBUG] Success with Markdown-stripped word.")
				return data, nil
			}
			log.Printf("[TTS DEBUG] Markdown-stripped word failed: %v", err)
		}
		// Fallback voices for Google
		if isGoogle {
			fallbackVoices := googleFallbackVoices
			if fallbackVoices == nil {
				fallbackVoices = buildFallbackVoices(origLang, origVoice)
			}
			for _, fallbackVoice := range fallbackVoices {
				log.Printf("[TTS DEBUG] Trying fallback voice: %s", fallbackVoice)
				data, err = provider.GenerateSpeech(ctx, &UnifiedRequest{
					Text:   chunk,
					Voice:  fallbackVoice,
					Speed:  request.Speed,
					Format: request.Format,
					Model:  request.Model,
				})
				if err == nil {
					if progressCb != nil {
						progressCb()
					}
					log.Printf("[TTS DEBUG] Fallback voice succeeded: %s", fallbackVoice)
					return data, nil
				}
				log.Printf("[TTS DEBUG] Fallback voice failed: %v", err)
			}
			// If all fallback voices fail, try error message chunk in en-US
			log.Printf("[TTS DEBUG] All fallback voices failed for chunk (len=%d): %.100s", chunkBytes, chunk)
			if errorCb != nil {
				errorCb(fmt.Sprintf(
					"A section could not be processed (%.40s...). Substituting error message and continuing.", chunk))
			}
			data, err = provider.GenerateSpeech(ctx, &UnifiedRequest{
				Text:   "Error converting Text. Continuing.",
				Voice:  "en-US-" + origVoice,
				Speed:  request.Speed,
				Format: request.Format,
				Model:  request.Model,
			})
			if err == nil {
				if progressCb != nil {
					progressCb()
				}
				log.Printf("[TTS DEBUG] Error message chunk succeeded.")
				return data, nil
			}
			log.Printf("[TTS DEBUG] Error message chunk failed: %v", err)
		}
	}

	// Log and show user-friendly error
	log.Printf("[TTS DEBUG] Final failed chunk (len=%d): %.100s", chunkBytes, chunk)
	if errorCb != nil {
		errorCb(fmt.Sprintf(
			"A section could not be processed (%.40s...). Try rephrasing or splitting it manually.", chunk))
	}
	return nil, err
}

// --- Utility functions ---

func getBackoffDelay(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 30 * time.Second
	case 2:
		return 60 * time.Second
	default:
		return 120 * time.Second
	}
}

func isQuotaOrRateError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "quota") ||
		strings.Contains(msg, "rate") ||
		strings.Contains(msg, "limit") ||
		strings.Contains(msg, "deadline")
}

func isRetryableTTS(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "502") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "deadlineexceeded") ||
		strings.Contains(msg, "quota") ||
		strings.Contains(msg, "rate")
}

// Remove special characters, keep only letters, numbers, and spaces
func sanitizeWordForTTS(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Remove Markdown formatting (common symbols)
func stripMarkdown(s string) string {
	reg := regexp.MustCompile(`[\\*_#\\[\\]()>~\` + "`" + `]+`)
	return reg.ReplaceAllString(s, "")
}

// Extract language code from a voice string (e.g. de-DE-Chirp3-HD-Sulafat -> de-DE)
func extractLangCode(voice string) string {
	parts := strings.Split(voice, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return "en-US"
}

// Build fallback voices list for Google
func buildFallbackVoices(origLang, origVoice string) []string {
	// Use the last part of the original voice as the suffix
	origSuffix := ""
	if dash := strings.LastIndex(origVoice, "-"); dash != -1 {
		origSuffix = origVoice[dash+1:]
	}
	return []string{
		fmt.Sprintf("%s-Chirp3-HD-%s", origLang, origSuffix),
		fmt.Sprintf("%s-Chirp-HD-O", origLang),
		fmt.Sprintf("%s-Neural2-G", origLang),
		fmt.Sprintf("%s-Standard-G", origLang),
		fmt.Sprintf("%s-Studio-C", origLang),
	}
}
