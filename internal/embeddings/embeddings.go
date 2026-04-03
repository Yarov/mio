// Package embeddings provides self-contained text vectorization for semantic search.
// No external services required — everything runs inside the Mio binary.
package embeddings

import (
	"math"
	"strings"
	"sync"
	"unicode"
)

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(text string) ([]float64, error)
}

// NullEmbedder is a no-op embedder.
type NullEmbedder struct{}

func (e *NullEmbedder) Embed(text string) ([]float64, error) {
	return nil, nil
}

// TFIDFEmbedder generates sparse TF-IDF vectors using the corpus vocabulary.
// Self-contained — no external services needed.
type TFIDFEmbedder struct {
	mu         sync.RWMutex
	vocab      map[string]int // word → index
	idf        []float64      // inverse document frequency per word
	docCount   int            // total documents in corpus
	docFreq    map[string]int // how many documents contain each word
	vocabDirty bool           // needs rebuild
}

// NewTFIDFEmbedder creates a self-contained TF-IDF embedder.
func NewTFIDFEmbedder() *TFIDFEmbedder {
	return &TFIDFEmbedder{
		vocab:   make(map[string]int),
		docFreq: make(map[string]int),
	}
}

// AddDocument registers a document's terms in the corpus for IDF calculation.
// Call this when saving new observations.
func (e *TFIDFEmbedder) AddDocument(text string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.docCount++
	seen := make(map[string]bool)
	for _, word := range tokenize(text) {
		if !seen[word] {
			seen[word] = true
			e.docFreq[word]++
			if _, ok := e.vocab[word]; !ok {
				e.vocab[word] = len(e.vocab)
			}
		}
	}
	e.vocabDirty = true
}

// RebuildIDF recalculates IDF weights from the current corpus stats.
func (e *TFIDFEmbedder) RebuildIDF() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.idf = make([]float64, len(e.vocab))
	for word, idx := range e.vocab {
		df := e.docFreq[word]
		if df > 0 {
			// Standard IDF with smoothing: log((N + 1) / (df + 1)) + 1
			e.idf[idx] = math.Log(float64(e.docCount+1)/float64(df+1)) + 1.0
		}
	}
	e.vocabDirty = false
}

// Embed generates a TF-IDF vector for the given text.
func (e *TFIDFEmbedder) Embed(text string) ([]float64, error) {
	e.mu.RLock()
	vocabSize := len(e.vocab)
	dirty := e.vocabDirty
	e.mu.RUnlock()

	if vocabSize == 0 {
		return nil, nil
	}

	// Rebuild IDF if corpus changed
	if dirty {
		e.RebuildIDF()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Compute term frequency
	words := tokenize(text)
	if len(words) == 0 {
		return nil, nil
	}

	tf := make(map[int]float64)
	for _, word := range words {
		if idx, ok := e.vocab[word]; ok {
			tf[idx]++
		}
	}

	// Normalize TF by document length
	docLen := float64(len(words))
	for idx := range tf {
		tf[idx] /= docLen
	}

	// Build sparse vector: TF × IDF, then L2-normalize
	vec := make([]float64, vocabSize)
	var norm float64
	for idx, tfVal := range tf {
		if idx < len(e.idf) {
			vec[idx] = tfVal * e.idf[idx]
			norm += vec[idx] * vec[idx]
		}
	}

	// L2 normalize for cosine similarity
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			if vec[i] != 0 {
				vec[i] /= norm
			}
		}
	}

	return vec, nil
}

// VocabSize returns the current vocabulary size.
func (e *TFIDFEmbedder) VocabSize() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.vocab)
}

// DocCount returns the number of documents in the corpus.
func (e *TFIDFEmbedder) DocCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.docCount
}

// LoadCorpusStats initializes the embedder from pre-computed corpus data.
// Used to restore state from the database on startup.
func (e *TFIDFEmbedder) LoadCorpusStats(docCount int, docFreq map[string]int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.docCount = docCount
	e.docFreq = docFreq
	e.vocab = make(map[string]int)
	for word := range docFreq {
		e.vocab[word] = len(e.vocab)
	}
	e.vocabDirty = true
}

// tokenize splits text into normalized tokens for TF-IDF.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var tokens []string
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if stopWords[w] {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

// stopWords for English and Spanish — filtered from tokenization.
var stopWords = map[string]bool{
	// English
	"the": true, "and": true, "for": true, "that": true, "this": true,
	"with": true, "are": true, "was": true, "from": true, "have": true,
	"has": true, "been": true, "will": true, "can": true, "not": true,
	"but": true, "all": true, "they": true, "were": true, "would": true,
	"there": true, "their": true, "what": true, "about": true, "which": true,
	"when": true, "make": true, "like": true, "just": true, "over": true,
	"such": true, "also": true, "some": true, "into": true, "than": true,
	"could": true, "other": true, "then": true, "its": true, "only": true,
	"very": true, "should": true, "now": true, "these": true, "after": true,
	"use": true, "used": true, "using": true, "how": true, "each": true,
	// Spanish
	"que": true, "por": true, "para": true, "con": true, "una": true,
	"los": true, "las": true, "del": true, "est": true, "pero": true,
	"como": true, "más": true, "esto": true, "esta": true, "todo": true,
	"hay": true, "ser": true, "tiene": true, "ese": true, "eso": true,
	"son": true, "nos": true, "sin": true, "sobre": true, "entre": true,
}
