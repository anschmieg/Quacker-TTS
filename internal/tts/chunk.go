package tts

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkoukk/tiktoken-go"
)

var sentenceEndRegex = regexp.MustCompile(`([.!?])`)
var hrSeparatorRegex = regexp.MustCompile(`\n(?:-{3,}|_{3,})\n`)
var multiNewlineSeparatorRegex = regexp.MustCompile(`\n\s*\n`)
var sentenceEndNewlineRegex = regexp.MustCompile(`([.!?])\s*\n`)

const (
	// DefaultTokenLimit sets the token limit per chunk
	DefaultTokenLimit = 2000
	// RequestInterval ensures we don't send more than 1 request per second
	RequestInterval = time.Second
)

// Helper to get initial chunks based on HR or multi-newline
func getInitialChunks(text string) []string {
	hrParts := hrSeparatorRegex.Split(text, -1)
	// Filter out empty strings resulting from split
	var filteredHrParts []string
	for _, p := range hrParts {
		if strings.TrimSpace(p) != "" {
			filteredHrParts = append(filteredHrParts, p)
		}
	}
	if len(filteredHrParts) > 1 {
		log.Printf("Split by HR into %d parts", len(filteredHrParts))
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
		log.Printf("Split by multiple newlines into %d parts", len(filteredMultiNewlineParts))
		return filteredMultiNewlineParts
	}

	log.Printf("No primary separators found, starting with 1 part")
	// Return the original text trimmed, as a single chunk if no separators found
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return []string{}
	}
	return []string{trimmedText}
}

// Simple rune splitter fallback
func splitByRune(text string, maxTokens int) []string {
	log.Printf("...............Applying rune splitting to '%.30s...'", text)
	runes := []rune(text)
	var chunks []string
	if len(runes) == 0 {
		return chunks
	}
	// Estimate rune count per token ~3. Use maxTokens * 3 runes as step.
	step := maxTokens * 3
	if step <= 0 {
		step = maxTokens // Ensure step is at least maxTokens if calculation fails
	}
	if step <= 0 {
		step = 1 // Absolute minimum step
	}

	for i := 0; i < len(runes); i += step {
		end := i + step
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// Word splitting fallback
func splitByWord(text string, enc *tiktoken.Tiktoken, maxTokens int) []string {
	log.Printf(".........Applying word splitting to '%.30s...'", text)
	var chunks []string
	var chunkBuilder strings.Builder

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	rebuildAndCheck := func() (string, []int) {
		str := chunkBuilder.String()
		if str == "" {
			return "", nil
		}
		return str, enc.Encode(str, nil, nil)
	}

	currentChunkStr, currentTokens := rebuildAndCheck()

	for _, word := range words {
		tokWord := enc.Encode(word, nil, nil) // Tokenize word alone first

		// Check if word itself is too long
		if len(tokWord) > maxTokens {
			log.Printf("............Word '%s' too long (%d tokens), splitting by runes", word, len(tokWord))
			if chunkBuilder.Len() > 0 {
				chunks = append(chunks, currentChunkStr) // Add previous chunk
			}
			chunks = append(chunks, splitByRune(word, maxTokens)...) // Add rune-split word
			chunkBuilder.Reset()                                     // Reset for next word
			currentChunkStr, currentTokens = rebuildAndCheck()
			continue
		}

		// Try adding the word (with a space if needed)
		testBuilder := chunkBuilder
		if testBuilder.Len() > 0 {
			testBuilder.WriteString(" ")
		}
		testBuilder.WriteString(word)
		testStr := testBuilder.String()
		testTokens := enc.Encode(testStr, nil, nil)

		if len(testTokens) <= maxTokens {
			// It fits, update builder
			chunkBuilder.Reset()
			chunkBuilder.WriteString(testStr)
			currentChunkStr = testStr
			currentTokens = testTokens
		} else {
			// Doesn't fit. Finalize previous chunk.
			if chunkBuilder.Len() > 0 {
				chunks = append(chunks, currentChunkStr)
			}
			// Start new chunk with the current word
			chunkBuilder.Reset()
			chunkBuilder.WriteString(word)
			currentChunkStr, currentTokens = rebuildAndCheck()

			// Safety check: if the word *alone* is too long (should be caught above)
			if len(currentTokens) > maxTokens {
				log.Printf("............Word '%s' alone exceeds limit (%d tokens) after failing to combine, splitting by runes", word, len(currentTokens))
				// Add the rune-split word directly, don't add the oversized word chunk
				chunks = append(chunks, splitByRune(word, maxTokens)...)
				chunkBuilder.Reset()
				currentChunkStr, currentTokens = rebuildAndCheck()
			}
		}
	}

	// Add the last chunk
	if chunkBuilder.Len() > 0 {
		chunks = append(chunks, currentChunkStr)
	}
	return chunks
}

// Recursive splitting function
func splitChunkRecursively(chunk string, enc *tiktoken.Tiktoken, maxTokens int, level int) []string {
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return []string{}
	}

	var regex *regexp.Regexp
	var splitLog string
	nextLevel := level + 1

	switch level {
	case 0:
		regex = sentenceEndNewlineRegex
		splitLog = "sentence end + newline"
	case 1:
		regex = sentenceEndRegex
		splitLog = "sentence end"
	case 2:
		log.Printf("...Splitting by words")
		return splitByWord(chunk, enc, maxTokens)
	case 3:
		log.Printf("...Splitting by runes")
		return splitByRune(chunk, maxTokens)
	default:
		log.Printf("!!! Unknown split level %d, falling back to runes", level)
		return splitByRune(chunk, maxTokens)
	}

	indices := regex.FindAllStringIndex(chunk, -1)
	if len(indices) == 0 {
		log.Printf("...No matches for %s, trying next level", splitLog)
		return splitChunkRecursively(chunk, enc, maxTokens, nextLevel)
	}

	log.Printf("...Splitting by %s", splitLog)
	var resultChunks []string
	var chunkBuilder strings.Builder
	lastPos := 0

	rebuildAndCheck := func() (string, []int) {
		str := chunkBuilder.String()
		if str == "" {
			return "", nil
		}
		return str, enc.Encode(str, nil, nil)
	}
	currentChunkStr, currentTokens := rebuildAndCheck()

	processSegment := func(segment string) {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return
		}
		tokSegment := enc.Encode(segment, nil, nil)

		// Check if the segment *itself* is too long
		if len(tokSegment) > maxTokens {
			log.Printf("......Segment '%.30s...' too long (%d tokens), recursing to next level", segment, len(tokSegment))
			if chunkBuilder.Len() > 0 { // Finalize any pending chunk before recursing
				resultChunks = append(resultChunks, currentChunkStr)
				chunkBuilder.Reset()
				currentChunkStr, currentTokens = rebuildAndCheck()
			}
			subChunks := splitChunkRecursively(segment, enc, maxTokens, nextLevel)
			resultChunks = append(resultChunks, subChunks...)
			return // Finished processing this oversized segment
		}

		// Try adding to the current builder
		testBuilder := chunkBuilder
		if testBuilder.Len() > 0 {
			testBuilder.WriteString(" ") // Add space separator
		}
		testBuilder.WriteString(segment)
		testChunkStr := testBuilder.String()
		testTokChunk := enc.Encode(testChunkStr, nil, nil)

		if len(testTokChunk) <= maxTokens {
			// It fits, update the builder and tokens
			chunkBuilder.Reset()
			chunkBuilder.WriteString(testChunkStr)
			currentChunkStr = testChunkStr
			currentTokens = testTokChunk
		} else {
			// It doesn't fit, finalize the previous chunk and start a new one
			if chunkBuilder.Len() > 0 {
				resultChunks = append(resultChunks, currentChunkStr)
			}
			// Start new chunk with the current segment
			chunkBuilder.Reset()
			chunkBuilder.WriteString(segment)
			currentChunkStr, currentTokens = rebuildAndCheck()

			// Double check if the segment *alone* exceeds the limit (should have been caught above)
			if len(currentTokens) > maxTokens {
				log.Printf("......Segment '%.30s...' alone exceeds limit (%d tokens) after failing to combine, recursing", segment, len(currentTokens))
				// Recurse on the segment and add results, reset builder
				resultChunks = append(resultChunks, splitChunkRecursively(segment, enc, maxTokens, nextLevel)...)
				chunkBuilder.Reset()
				currentChunkStr, currentTokens = rebuildAndCheck()
			}
		}
	}

	for _, idx := range indices {
		// Process the text segment *up to and including* the separator found by the regex.
		// idx[0] is start of match, idx[1] is end of match.
		segment := chunk[lastPos:idx[1]]
		processSegment(segment)
		lastPos = idx[1]
	}

	// Handle remaining text after the last index
	if lastPos < len(chunk) {
		tail := chunk[lastPos:]
		processSegment(tail)
	}

	// Add the last built chunk if any
	if chunkBuilder.Len() > 0 {
		resultChunks = append(resultChunks, currentChunkStr)
	}

	return resultChunks
}

// splitText breaks text into token-bounded chunks using hierarchical separators.
func splitText(text, model string, maxTokens int) []string {
	log.Printf("splitText called: model=%s, input runes=%d", model, len([]rune(text)))
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	enc, err := tiktoken.GetEncoding(model)
	if err != nil {
		if model != "cl100k_base" { // Log only once per session for unknown encodings
			log.Printf("Warning: Unknown encoding for model %s. Falling back to cl100k_base.", model)
		}
		enc, err = tiktoken.GetEncoding("cl100k_base")
	}
	if err != nil {
		log.Printf("Error: Failed to get fallback encoding: %v. Using rune splitting.", err)
		return splitByRune(text, maxTokens) // Use rune splitting if tokenizer fails completely
	}

	initialChunks := getInitialChunks(text)
	var resultChunks []string

	for _, chunk := range initialChunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}

		tokChunk := enc.Encode(chunk, nil, nil)
		if len(tokChunk) <= maxTokens {
			resultChunks = append(resultChunks, chunk)
		} else {
			// Chunk is too long, apply hierarchical splitting recursively
			log.Printf("Chunk starting with '%.30s...' is too long (%d tokens), applying recursive splitting", chunk, len(tokChunk))
			subChunks := splitChunkRecursively(chunk, enc, maxTokens, 0) // Pass the pointer directly
			resultChunks = append(resultChunks, subChunks...)
		}
	}

	// post-process to ensure no chunk exceeds token limit (safety net)
	var finalChunks []string
	for _, c := range resultChunks { // Iterate over results from recursive splitting
		tokC := enc.Encode(c, nil, nil)
		if len(tokC) <= maxTokens {
			finalChunks = append(finalChunks, c)
		} else {
			log.Printf("Error: Post-processing needed for chunk '%.30s...' (%d tokens). Using rune splitting.", c, len(tokC))
			// Fallback to rune splitting for safety if recursion somehow produced oversized chunk
			subChunks := splitByRune(c, maxTokens)
			finalChunks = append(finalChunks, subChunks...)
		}
	}
	return finalChunks
}

// GenerateSpeechChunks splits the request input into sub-chunks, sends them in parallel at up to 1/sec,
// and concatenates the resulting audio blobs.
func (c *Client) GenerateSpeechChunks(req Request) ([]byte, error) {
	// determine max tokens per chunk
	maxTokens := DefaultTokenLimit
	parts := splitText(req.Input, req.Model, maxTokens)
	results := make([][]byte, len(parts))
	errs := make([]error, len(parts))
	var wg sync.WaitGroup
	ticker := time.NewTicker(RequestInterval)
	defer ticker.Stop()

	for i, chunk := range parts {
		wg.Add(1)
		go func(i int, textChunk string) {
			defer wg.Done()
			<-ticker.C
			subReq := req
			subReq.Input = textChunk
			data, err := c.GenerateSpeech(subReq)
			results[i] = data
			errs[i] = err
		}(i, chunk)
	}
	wg.Wait()

	// check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	// concatenate audio
	var combined []byte
	for _, blob := range results {
		combined = append(combined, blob...)
	}
	return combined, nil
}
