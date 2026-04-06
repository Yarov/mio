package store

import (
	"strings"
	"testing"
	"time"
)

// Test extractKeywords
func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minWords int
		maxWords int
	}{
		{"short message ignored", "ok va", 0, 0},
		{"stopwords filtered", "the and for that this with are", 0, 0},
		{"spanish stopwords filtered", "como que por para con una", 0, 0},
		{"real keywords extracted", "implement database migration schema", 3, 4},
		{"max 8 keywords", "alpha bravo charlie delta echo foxtrot golf hotel india juliet", 8, 8},
		{"punctuation stripped", "hello! world? test. code,", 4, 4},
		{"short words ignored", "a an is it to go do no", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kw := extractKeywords(tt.input)
			if len(kw) < tt.minWords || len(kw) > tt.maxWords {
				t.Errorf("extractKeywords(%q) = %d words %v, want %d-%d", tt.input, len(kw), kw, tt.minWords, tt.maxWords)
			}
		})
	}
}

// Test extractSummary (the summarization function)
func TestExtractSummary(t *testing.T) {
	content := `What: Implemented new feature
Why: User requested it
Where: internal/store/
Learned: Key takeaway here
Extra detail line 1
Extra detail line 2
More unnecessary detail that should be cut`

	summary := extractSummary(content, 200)

	if len(summary) > 200 {
		t.Errorf("summary too long: %d > 200", len(summary))
	}
	if !strings.Contains(summary, "What:") {
		t.Error("summary missing What: line")
	}
	if !strings.Contains(summary, "Why:") {
		t.Error("summary missing Why: line")
	}
}

func TestExtractSummaryShortContent(t *testing.T) {
	content := "Short content"
	summary := extractSummary(content, 200)
	if summary != content {
		t.Errorf("short content should be unchanged: got %q", summary)
	}
}

func TestExtractSummaryMaxLen(t *testing.T) {
	// Very long content should be truncated
	content := strings.Repeat("What: This is a very long line that keeps going. ", 20)
	summary := extractSummary(content, 100)
	if len(summary) > 120 { // allow small overflow for "..."
		t.Errorf("summary exceeds maxLen: %d", len(summary))
	}
}

func TestSummarize(t *testing.T) {
	s := testStore(t)

	// Create an old observation with long content
	longContent := strings.Repeat("What: detailed content line. ", 20)
	obs := testObs("Old discovery", longContent)

	id, err := s.Save(obs)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Backdate the observation
	_, err = s.db.Exec("UPDATE observations SET created_at = ? WHERE id = ?",
		time.Now().UTC().AddDate(0, 0, -60).Format(time.RFC3339), id)
	if err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// Run summarize — note: the background autoMaintenance goroutine may have
	// already summarized this observation, so count could be 0 or 1.
	count, err := s.Summarize("", 30, 200)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}

	// Wait briefly for any concurrent auto-maintenance to finish, then verify
	// the content was compressed (either by our call or auto-maintenance).
	time.Sleep(50 * time.Millisecond)

	updated, err := s.GetObservation(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.HasPrefix(updated.Content, "[summarized]") {
		t.Errorf("summarized content should start with [summarized], count=%d, content prefix=%q", count, updated.Content[:min(50, len(updated.Content))])
	}
	if len(updated.Content) > 250 { // 200 + "[summarized] " prefix
		t.Errorf("content still too long: %d", len(updated.Content))
	}
}

func TestSurfaceRelevantShortMessage(t *testing.T) {
	s := testStore(t)

	results, err := s.SurfaceRelevant("ok", "", 5)
	if err != nil {
		t.Fatalf("surface: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for short message, got %d", len(results))
	}
}
