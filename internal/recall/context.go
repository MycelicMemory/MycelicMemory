package recall

import (
	"path/filepath"
	"strings"
)

// Common stop words to filter from keyword extraction
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "shall": true, "can": true, "need": true, "dare": true,
	"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
	"at": true, "by": true, "from": true, "as": true, "into": true,
	"through": true, "during": true, "before": true, "after": true,
	"above": true, "below": true, "between": true, "out": true, "off": true,
	"over": true, "under": true, "again": true, "further": true, "then": true,
	"once": true, "and": true, "but": true, "or": true, "nor": true,
	"not": true, "so": true, "yet": true, "both": true, "either": true,
	"neither": true, "each": true, "every": true, "all": true, "any": true,
	"few": true, "more": true, "most": true, "other": true, "some": true,
	"such": true, "no": true, "only": true, "own": true, "same": true,
	"than": true, "too": true, "very": true, "just": true, "because": true,
	"if": true, "when": true, "while": true, "how": true, "what": true,
	"which": true, "who": true, "whom": true, "this": true, "that": true,
	"these": true, "those": true, "i": true, "me": true, "my": true,
	"we": true, "our": true, "you": true, "your": true, "he": true,
	"him": true, "his": true, "she": true, "her": true, "it": true,
	"its": true, "they": true, "them": true, "their": true, "about": true,
	"up": true, "also": true, "use": true, "using": true, "used": true,
}

// extractKeywords extracts meaningful keywords from text, filtering stop words
func extractKeywords(text string, maxKeywords int) []string {
	// Normalize: lowercase, replace punctuation with spaces
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(
		".", " ", ",", " ", ";", " ", ":", " ", "!", " ", "?", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
		"\"", " ", "'", " ", "`", " ", "\n", " ", "\r", " ", "\t", " ",
	)
	text = replacer.Replace(text)

	words := strings.Fields(text)
	seen := make(map[string]bool)
	var keywords []string

	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		if stopWords[word] {
			continue
		}
		if seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
		if len(keywords) >= maxKeywords {
			break
		}
	}

	return keywords
}

// extractTagsFromFiles extracts searchable tags from file paths
func extractTagsFromFiles(files []string) []string {
	seen := make(map[string]bool)
	var tags []string

	for _, f := range files {
		// Extract file extension as language tag
		ext := strings.TrimPrefix(filepath.Ext(f), ".")
		if ext != "" && !seen[ext] {
			seen[ext] = true
			tags = append(tags, ext)
		}

		// Extract directory components as domain tags
		dir := filepath.Dir(f)
		parts := strings.FieldsFunc(dir, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		for _, part := range parts {
			part = strings.ToLower(part)
			if len(part) > 1 && !seen[part] && !stopWords[part] {
				seen[part] = true
				tags = append(tags, part)
			}
		}
	}

	return tags
}

// buildSearchQuery joins keywords as an FTS5 OR query
func buildSearchQuery(keywords []string) string {
	if len(keywords) == 0 {
		return ""
	}
	return strings.Join(keywords, " OR ")
}

// truncateForEmbedding caps text for embedding quality
func truncateForEmbedding(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}
