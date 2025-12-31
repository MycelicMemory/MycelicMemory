package memory

import (
	"strings"
	"unicode"
)

// ChunkConfig contains configuration for memory chunking
type ChunkConfig struct {
	// MaxChunkSize is the maximum number of characters per chunk (not tokens, for simplicity)
	// Default: 1000 characters (~200-250 tokens)
	MaxChunkSize int

	// OverlapSize is the number of characters to overlap between chunks
	// Default: 100 characters
	OverlapSize int

	// MinChunkSize is the minimum size to consider creating chunks
	// If content is smaller than this, don't chunk at all
	// Default: 1500 characters
	MinChunkSize int
}

// DefaultChunkConfig returns the default chunking configuration
func DefaultChunkConfig() *ChunkConfig {
	return &ChunkConfig{
		MaxChunkSize: 1000,  // ~200-250 tokens
		OverlapSize:  100,   // ~25 tokens overlap
		MinChunkSize: 1500,  // Only chunk content larger than this
	}
}

// Chunk represents a piece of a larger memory
type Chunk struct {
	Content    string
	Index      int
	Level      int // 1 = paragraph level
	StartPos   int // Position in original content
	EndPos     int
}

// Chunker handles splitting content into hierarchical chunks
type Chunker struct {
	config *ChunkConfig
}

// NewChunker creates a new Chunker with the given configuration
func NewChunker(config *ChunkConfig) *Chunker {
	if config == nil {
		config = DefaultChunkConfig()
	}
	return &Chunker{config: config}
}

// ShouldChunk determines if content should be chunked based on size
func (c *Chunker) ShouldChunk(content string) bool {
	return len(content) > c.config.MinChunkSize
}

// ChunkContent splits content into chunks with overlap
// Returns nil if content doesn't need chunking
func (c *Chunker) ChunkContent(content string) []Chunk {
	if !c.ShouldChunk(content) {
		return nil
	}

	// First, try to split on paragraph boundaries
	paragraphs := splitIntoParagraphs(content)

	if len(paragraphs) > 1 {
		return c.chunkByParagraphs(paragraphs, content)
	}

	// If no paragraphs, split by sentence boundaries
	return c.chunkBySentences(content)
}

// chunkByParagraphs groups paragraphs into chunks respecting max size
func (c *Chunker) chunkByParagraphs(paragraphs []string, originalContent string) []Chunk {
	var chunks []Chunk
	var currentChunk strings.Builder
	var currentStart int
	chunkIndex := 0
	position := 0

	for i, para := range paragraphs {
		paraWithSep := para
		if i < len(paragraphs)-1 {
			paraWithSep = para + "\n\n"
		}

		// If adding this paragraph exceeds max size and we have content, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(paraWithSep) > c.config.MaxChunkSize {
			chunks = append(chunks, Chunk{
				Content:  strings.TrimSpace(currentChunk.String()),
				Index:    chunkIndex,
				Level:    1,
				StartPos: currentStart,
				EndPos:   position,
			})
			chunkIndex++

			// Start new chunk with overlap from previous
			overlapContent := getOverlapSuffix(currentChunk.String(), c.config.OverlapSize)
			currentChunk.Reset()
			currentChunk.WriteString(overlapContent)
			currentStart = position - len(overlapContent)
		}

		currentChunk.WriteString(paraWithSep)
		position += len(paraWithSep)
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Content:  strings.TrimSpace(currentChunk.String()),
			Index:    chunkIndex,
			Level:    1,
			StartPos: currentStart,
			EndPos:   position,
		})
	}

	return chunks
}

// chunkBySentences splits content by sentences when no paragraphs exist
func (c *Chunker) chunkBySentences(content string) []Chunk {
	sentences := splitIntoSentences(content)

	var chunks []Chunk
	var currentChunk strings.Builder
	var currentStart int
	chunkIndex := 0
	position := 0

	for _, sentence := range sentences {
		sentenceWithSpace := sentence + " "

		if currentChunk.Len() > 0 && currentChunk.Len()+len(sentenceWithSpace) > c.config.MaxChunkSize {
			chunks = append(chunks, Chunk{
				Content:  strings.TrimSpace(currentChunk.String()),
				Index:    chunkIndex,
				Level:    1,
				StartPos: currentStart,
				EndPos:   position,
			})
			chunkIndex++

			overlapContent := getOverlapSuffix(currentChunk.String(), c.config.OverlapSize)
			currentChunk.Reset()
			currentChunk.WriteString(overlapContent)
			currentStart = position - len(overlapContent)
		}

		currentChunk.WriteString(sentenceWithSpace)
		position += len(sentenceWithSpace)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Content:  strings.TrimSpace(currentChunk.String()),
			Index:    chunkIndex,
			Level:    1,
			StartPos: currentStart,
			EndPos:   position,
		})
	}

	return chunks
}

// splitIntoParagraphs splits content by paragraph boundaries
func splitIntoParagraphs(content string) []string {
	// Split on double newlines (paragraph separator)
	rawParagraphs := strings.Split(content, "\n\n")

	var paragraphs []string
	for _, p := range rawParagraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			paragraphs = append(paragraphs, trimmed)
		}
	}

	return paragraphs
}

// splitIntoSentences splits content by sentence boundaries
func splitIntoSentences(content string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range content {
		current.WriteRune(r)

		// Check for sentence-ending punctuation followed by space or end
		if isSentenceEnd(r) {
			// Look ahead to see if followed by space or end
			if i == len(content)-1 || (i+1 < len(content) && unicode.IsSpace(rune(content[i+1]))) {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Don't forget remaining content
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

// isSentenceEnd checks if a rune is a sentence-ending character
func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}

// getOverlapSuffix returns the last n characters for overlap
func getOverlapSuffix(content string, n int) string {
	if len(content) <= n {
		return content
	}
	return content[len(content)-n:]
}
