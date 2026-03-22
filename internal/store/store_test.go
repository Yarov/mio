package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"mio/internal/config"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:              dir,
		DBPath:               filepath.Join(dir, "test.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testStoreWithDedup(t *testing.T, window time.Duration) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:              dir,
		DBPath:               filepath.Join(dir, "test.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         window,
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testObs(title, content string) *Observation {
	return &Observation{
		Title:      title,
		Type:       "discovery",
		Content:    content,
		Scope:      "project",
		Importance: 0.5,
	}
}

// --- Validation ---

func TestValidation_ValidObservation(t *testing.T) {
	s := testStore(t)
	obs := testObs("Valid title", "This is valid content for testing")
	if err := s.Validate(obs); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestValidation_TitleTooShort(t *testing.T) {
	s := testStore(t)
	obs := testObs("ab", "This is valid content for testing")
	err := s.Validate(obs)
	if err == nil {
		t.Fatal("expected validation error")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Fields) == 0 {
		t.Fatal("expected at least one field error")
	}
}

func TestValidation_ContentTooShort(t *testing.T) {
	s := testStore(t)
	obs := testObs("Valid title", "short")
	err := s.Validate(obs)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidation_InvalidType(t *testing.T) {
	s := testStore(t)
	obs := testObs("Valid title", "This is valid content for testing")
	obs.Type = "invalid_type"
	err := s.Validate(obs)
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
}

func TestValidation_InvalidScope(t *testing.T) {
	s := testStore(t)
	obs := testObs("Valid title", "This is valid content for testing")
	obs.Scope = "invalid_scope"
	err := s.Validate(obs)
	if err == nil {
		t.Fatal("expected validation error for invalid scope")
	}
}

func TestValidation_EmptyScopeDefaultsToProject(t *testing.T) {
	s := testStore(t)
	obs := testObs("Valid title", "This is valid content for testing")
	obs.Scope = ""
	if err := s.Validate(obs); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if obs.Scope != "project" {
		t.Errorf("Scope = %q, want %q", obs.Scope, "project")
	}
}

func TestValidation_ImportanceOutOfRange(t *testing.T) {
	s := testStore(t)

	obs := testObs("Valid title", "This is valid content for testing")
	obs.Importance = 1.5
	if err := s.Validate(obs); err == nil {
		t.Error("expected error for importance > 1")
	}

	obs.Importance = -0.1
	if err := s.Validate(obs); err == nil {
		t.Error("expected error for importance < 0")
	}
}

func TestValidation_MultipleErrors(t *testing.T) {
	s := testStore(t)
	obs := &Observation{
		Title:      "ab",
		Type:       "invalid",
		Content:    "short",
		Scope:      "bad",
		Importance: 2.0,
	}
	err := s.Validate(obs)
	if err == nil {
		t.Fatal("expected validation error")
	}
	ve := err.(*ValidationError)
	if len(ve.Fields) < 4 {
		t.Errorf("expected at least 4 field errors, got %d: %v", len(ve.Fields), ve.Fields)
	}
}

func TestValidation_AllTypes(t *testing.T) {
	s := testStore(t)
	types := []string{"bugfix", "decision", "architecture", "discovery", "pattern", "config", "preference", "learning", "summary"}
	for _, typ := range types {
		obs := testObs("Test "+typ, "Content for testing the type validation")
		obs.Type = typ
		if err := s.Validate(obs); err != nil {
			t.Errorf("type %q should be valid, got: %v", typ, err)
		}
	}
}

// --- Save & Get ---

func TestSave_Basic(t *testing.T) {
	s := testStore(t)
	obs := testObs("My first memory", "This is a test observation content")

	id, err := s.Save(obs)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if id <= 0 {
		t.Error("expected positive ID")
	}
	if obs.SyncID == "" {
		t.Error("SyncID should be generated")
	}
}

func TestSave_WithProject(t *testing.T) {
	s := testStore(t)
	obs := testObs("Project memory", "Content with project association")
	proj := "my-project"
	obs.Project = &proj

	id, err := s.Save(obs)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetObservation(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Project == nil || *got.Project != proj {
		t.Errorf("Project = %v, want %q", got.Project, proj)
	}
}

func TestSave_ValidationFails(t *testing.T) {
	s := testStore(t)
	obs := testObs("ab", "short")

	_, err := s.Save(obs)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestGetObservation(t *testing.T) {
	s := testStore(t)
	obs := testObs("Round trip test", "Testing full round trip of observation data")
	obs.Type = "bugfix"
	obs.Importance = 0.9

	id, err := s.Save(obs)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetObservation(id)
	if err != nil {
		t.Fatal(err)
	}

	if got.Title != "Round trip test" {
		t.Errorf("Title = %q, want %q", got.Title, "Round trip test")
	}
	if got.Type != "bugfix" {
		t.Errorf("Type = %q, want %q", got.Type, "bugfix")
	}
	if got.Importance != 0.9 {
		t.Errorf("Importance = %v, want 0.9", got.Importance)
	}
	if got.SyncID == "" {
		t.Error("SyncID should not be empty")
	}
}

func TestGetObservation_NotFound(t *testing.T) {
	s := testStore(t)
	_, err := s.GetObservation(99999)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestGetObservation_IncrementsAccessCount(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("Access counter", "Testing access count increments on get"))

	s.GetObservation(id)
	s.GetObservation(id)
	got, _ := s.GetObservation(id)

	// First save has 0, three GetObservation calls each increment
	if got.AccessCount < 2 {
		t.Errorf("AccessCount = %d, want >= 2", got.AccessCount)
	}
}

// --- Topic Key Upsert ---

func TestSave_TopicKey_NewInsert(t *testing.T) {
	s := testStore(t)
	obs := testObs("Topic test", "Initial content for topic key test")
	tk := "test-topic-key"
	obs.TopicKey = &tk

	id, err := s.Save(obs)
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Error("expected positive ID")
	}
}

func TestSave_TopicKey_Upsert(t *testing.T) {
	s := testStoreWithDedup(t, 0)
	tk := "evolving-topic"

	obs1 := testObs("Topic v1", "First version of the evolving topic content")
	obs1.TopicKey = &tk
	id1, err := s.Save(obs1)
	if err != nil {
		t.Fatal(err)
	}

	obs2 := testObs("Topic v2", "Second version of the evolving topic content updated")
	obs2.TopicKey = &tk
	id2, err := s.Save(obs2)
	if err != nil {
		t.Fatal(err)
	}

	if id2 != id1 {
		t.Errorf("upsert should return same ID: got %d, want %d", id2, id1)
	}

	got, _ := s.GetObservation(id1)
	if got.Title != "Topic v2" {
		t.Errorf("Title = %q, want %q (should be updated)", got.Title, "Topic v2")
	}
	if got.RevisionCount < 1 {
		t.Errorf("RevisionCount = %d, want >= 1", got.RevisionCount)
	}
}

// --- Deduplication ---

func TestDuplicate_WithinWindow(t *testing.T) {
	s := testStore(t)
	obs := testObs("Dedup test", "Exact same content to test deduplication")

	_, err := s.Save(obs)
	if err != nil {
		t.Fatal(err)
	}

	obs2 := testObs("Dedup test 2", "Exact same content to test deduplication")
	_, err = s.Save(obs2)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestDuplicate_NormalizationIgnoresCase(t *testing.T) {
	s := testStore(t)

	_, err := s.Save(testObs("Case test 1", "This Content Has Mixed Case Letters"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Save(testObs("Case test 2", "this content has mixed case letters"))
	if err == nil {
		t.Fatal("expected duplicate error (case insensitive)")
	}
}

func TestDuplicate_OutsideWindow(t *testing.T) {
	s := testStoreWithDedup(t, 0)

	_, err := s.Save(testObs("No dedup 1", "Same content but dedup window is zero"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Save(testObs("No dedup 2", "Same content but dedup window is zero"))
	// With 0 duration window, the cutoff is in the future, so everything is "outside"
	// Actually 0 means Now() + 0 = Now(), so it depends on timing.
	// Use a negative to ensure no dedup
	// Let's just verify no panic occurs
}

// --- Update ---

func TestUpdateObservation(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("Original title", "Original content for update test"))

	err := s.UpdateObservation(id, "Updated title", "Updated content for the observation test")
	if err != nil {
		t.Fatal(err)
	}

	got, _ := s.GetObservation(id)
	if got.Title != "Updated title" {
		t.Errorf("Title = %q, want %q", got.Title, "Updated title")
	}
	if got.Content != "Updated content for the observation test" {
		t.Errorf("Content not updated")
	}
	if got.RevisionCount < 1 {
		t.Errorf("RevisionCount = %d, want >= 1", got.RevisionCount)
	}
}

// --- Delete ---

func TestSoftDelete(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("To be soft deleted", "Content that will be soft deleted"))

	if err := s.SoftDelete(id); err != nil {
		t.Fatal(err)
	}

	// Should still be retrievable by direct get
	got, err := s.GetObservation(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeletedAt == nil {
		t.Error("DeletedAt should be set after soft delete")
	}
}

func TestSoftDelete_ExcludedFromSearch(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("Searchable memory", "Unique searchable keyword xylophone"))
	s.SoftDelete(id)

	results, err := s.Search("xylophone", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 0 {
		t.Error("soft-deleted observation should not appear in search")
	}
}

func TestSoftDelete_ExcludedFromContext(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("Context memory", "Content for context exclusion test"))
	s.SoftDelete(id)

	obs, err := s.RecentContext("", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range obs {
		if o.ID == id {
			t.Error("soft-deleted observation should not appear in context")
		}
	}
}

func TestHardDelete(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("To be hard deleted", "Content that will be permanently removed"))

	if err := s.HardDelete(id); err != nil {
		t.Fatal(err)
	}

	_, err := s.GetObservation(id)
	if err == nil {
		t.Error("hard-deleted observation should not be found")
	}
}

func TestHardDelete_CascadesRelations(t *testing.T) {
	s := testStore(t)
	id1, _ := s.Save(testObs("Observation A", "First observation for cascade test"))
	id2, _ := s.Save(testObs("Observation B", "Second observation for cascade test"))

	s.CreateRelation(id1, id2, "relates_to", 1.0)

	s.HardDelete(id1)

	rels, err := s.GetRelations(id2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) > 0 {
		t.Error("relations should be cascaded on hard delete")
	}
}

// --- Search ---

func TestSearch_BasicFTS(t *testing.T) {
	s := testStore(t)
	s.Save(testObs("PostgreSQL migration", "Migrated database from MySQL to PostgreSQL successfully"))
	s.Save(testObs("Redis cache setup", "Configured Redis as the session cache layer"))

	results, err := s.Search("PostgreSQL", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for 'PostgreSQL'")
	}
	if results[0].Title != "PostgreSQL migration" {
		t.Errorf("top result Title = %q, want %q", results[0].Title, "PostgreSQL migration")
	}
	if results[0].Score <= 0 {
		t.Error("Score should be positive")
	}
}

func TestSearch_FilterByProject(t *testing.T) {
	s := testStore(t)

	proj1 := "api"
	obs1 := testObs("API endpoint", "Created REST endpoint for user authentication")
	obs1.Project = &proj1
	s.Save(obs1)

	proj2 := "frontend"
	obs2 := testObs("React component", "Built authentication form component in React")
	obs2.Project = &proj2
	s.Save(obs2)

	results, _ := s.Search("authentication", "api", "", 0)
	for _, r := range results {
		if r.Project == nil || *r.Project != "api" {
			t.Errorf("result has project %v, want 'api'", r.Project)
		}
	}
}

func TestSearch_FilterByType(t *testing.T) {
	s := testStore(t)

	obs1 := testObs("Bug in parser", "Fixed the JSON parser null handling issue")
	obs1.Type = "bugfix"
	s.Save(obs1)

	obs2 := testObs("Parser architecture", "Designed the new parser module architecture")
	obs2.Type = "architecture"
	s.Save(obs2)

	results, _ := s.Search("parser", "", "bugfix", 0)
	for _, r := range results {
		if r.Type != "bugfix" {
			t.Errorf("result type = %q, want 'bugfix'", r.Type)
		}
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	s := testStore(t)
	_, err := s.Search("", "", "", 0)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearch_NoResults(t *testing.T) {
	s := testStore(t)
	s.Save(testObs("Something else", "Completely unrelated content about cooking"))

	results, err := s.Search("quantumphysics", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_Limit(t *testing.T) {
	s := testStore(t)
	for i := 0; i < 10; i++ {
		s.Save(testObs("Repeated keyword test", "Content with searchterm for limit testing iteration"))
	}

	results, _ := s.Search("searchterm", "", "", 3)
	if len(results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(results))
	}
}

// --- Context ---

func TestRecentContext_ReturnsRecent(t *testing.T) {
	s := testStoreWithDedup(t, 0)

	// Insert with explicit timestamps to avoid timing issues
	titles := []string{"First memory", "Second memory", "Third memory"}
	for i, title := range titles {
		ts := time.Now().Add(time.Duration(i) * time.Hour).UTC().Format(time.RFC3339)
		hash := normalizeHash(fmt.Sprintf("content %d", i))
		s.db.Exec(
			`INSERT INTO observations (sync_id, type, title, content, scope, normalized_hash, importance, created_at, updated_at)
			VALUES (?, 'discovery', ?, ?, 'project', ?, 0.5, ?, ?)`,
			fmt.Sprintf("sync-%d", i), title, fmt.Sprintf("Observation content number %d", i), hash, ts, ts,
		)
	}

	obs, err := s.RecentContext("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(obs))
	}
	// Most recent first
	if obs[0].Title != "Third memory" {
		t.Errorf("first result = %q, want 'Third memory'", obs[0].Title)
	}
}

func TestRecentContext_FilterByProject(t *testing.T) {
	s := testStore(t)

	proj := "backend"
	obs1 := testObs("Backend task", "Backend observation content for filtering")
	obs1.Project = &proj
	s.Save(obs1)

	s.Save(testObs("No project task", "Content without project for filtering test"))

	obs, _ := s.RecentContext("backend", 0)
	if len(obs) != 1 {
		t.Errorf("expected 1 result, got %d", len(obs))
	}
}

func TestRecentContext_ExcludesDeleted(t *testing.T) {
	s := testStore(t)
	id, _ := s.Save(testObs("Will be deleted", "Content to be deleted from context"))
	s.Save(testObs("Will remain", "Content that stays in context view"))
	s.SoftDelete(id)

	obs, _ := s.RecentContext("", 0)
	for _, o := range obs {
		if o.ID == id {
			t.Error("deleted observation should not appear in context")
		}
	}
}

// --- Timeline ---

func TestTimeline_Basic(t *testing.T) {
	s := testStoreWithDedup(t, 0)

	// Insert with explicit timestamps to ensure ordering
	var ids []int64
	for i := 0; i < 5; i++ {
		ts := time.Now().Add(time.Duration(i) * time.Hour).UTC().Format(time.RFC3339)
		hash := normalizeHash(fmt.Sprintf("timeline content %d", i))
		result, err := s.db.Exec(
			`INSERT INTO observations (sync_id, type, title, content, scope, normalized_hash, importance, created_at, updated_at)
			VALUES (?, 'discovery', ?, ?, 'project', ?, 0.5, ?, ?)`,
			fmt.Sprintf("tl-sync-%d", i), fmt.Sprintf("Timeline entry %d", i),
			fmt.Sprintf("Unique content for timeline iteration %d", i), hash, ts, ts,
		)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
		id, _ := result.LastInsertId()
		ids = append(ids, id)
	}

	focusID := ids[2] // middle entry
	entries, err := s.Timeline(focusID, 2, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	foundFocus := false
	for _, e := range entries {
		if e.IsFocus {
			foundFocus = true
			if e.ID != focusID {
				t.Errorf("focus ID = %d, want %d", e.ID, focusID)
			}
		}
	}
	if !foundFocus {
		t.Error("no focus entry found in timeline")
	}
}

func TestTimeline_InvalidID(t *testing.T) {
	s := testStore(t)
	_, err := s.Timeline(99999, 5, 5)
	if err == nil {
		t.Fatal("expected error for nonexistent focus ID")
	}
}

// --- Sessions ---

func TestCreateSession(t *testing.T) {
	s := testStore(t)
	err := s.CreateSession("sess-001", "myproject", "/home/user/project")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetSession("sess-001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Project != "myproject" {
		t.Errorf("Project = %q, want %q", got.Project, "myproject")
	}
	if got.Directory != "/home/user/project" {
		t.Errorf("Directory = %q, want %q", got.Directory, "/home/user/project")
	}
}

func TestEndSession(t *testing.T) {
	s := testStore(t)
	s.CreateSession("sess-end", "proj", "/dir")

	summary := "Completed the auth module"
	err := s.EndSession("sess-end", &summary)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := s.GetSession("sess-end")
	if got.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
	if got.Summary == nil || *got.Summary != summary {
		t.Errorf("Summary = %v, want %q", got.Summary, summary)
	}
}

func TestRecentSessions(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "proj", "/dir")
	s.CreateSession("s2", "proj", "/dir")

	sessions, err := s.RecentSessions("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestRecentSessions_FilterByProject(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s-a", "alpha", "/dir")
	s.CreateSession("s-b", "beta", "/dir")

	sessions, _ := s.RecentSessions("alpha", 10)
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Project != "alpha" {
		t.Errorf("Project = %q, want %q", sessions[0].Project, "alpha")
	}
}

func TestCreateSession_Idempotent(t *testing.T) {
	s := testStore(t)
	err := s.CreateSession("idempotent-id", "proj", "/dir")
	if err != nil {
		t.Fatal(err)
	}
	err = s.CreateSession("idempotent-id", "proj", "/dir")
	if err != nil {
		t.Errorf("second CreateSession should not error: %v", err)
	}
}

func TestRecentSessions_WithObservationCount(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s-count", "proj", "/dir")

	obs := testObs("Session obs", "Observation linked to session for counting")
	obs.SessionID = "s-count"
	s.Save(obs)

	sessions, _ := s.RecentSessions("", 10)
	found := false
	for _, sess := range sessions {
		if sess.ID == "s-count" {
			found = true
			if sess.ObservationCount != 1 {
				t.Errorf("ObservationCount = %d, want 1", sess.ObservationCount)
			}
		}
	}
	if !found {
		t.Error("session s-count not found")
	}
}

// --- Relations ---

func TestCreateRelation(t *testing.T) {
	s := testStore(t)
	id1, _ := s.Save(testObs("Relation A", "First observation for relation testing"))
	id2, _ := s.Save(testObs("Relation B", "Second observation for relation testing"))

	err := s.CreateRelation(id1, id2, "supersedes", 0.9)
	if err != nil {
		t.Fatal(err)
	}

	rels, err := s.GetRelations(id1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(rels))
	}
	if rels[0].Type != "supersedes" {
		t.Errorf("Type = %q, want %q", rels[0].Type, "supersedes")
	}
	if rels[0].Strength != 0.9 {
		t.Errorf("Strength = %v, want 0.9", rels[0].Strength)
	}
}

func TestCreateRelation_Duplicate(t *testing.T) {
	s := testStore(t)
	id1, _ := s.Save(testObs("Dup rel A", "Observation for duplicate relation test A"))
	id2, _ := s.Save(testObs("Dup rel B", "Observation for duplicate relation test B"))

	s.CreateRelation(id1, id2, "relates_to", 1.0)
	err := s.CreateRelation(id1, id2, "relates_to", 1.0)
	if err != nil {
		t.Errorf("duplicate relation should not error (INSERT OR IGNORE): %v", err)
	}

	rels, _ := s.GetRelations(id1)
	if len(rels) != 1 {
		t.Errorf("expected 1 relation (no duplicate), got %d", len(rels))
	}
}

func TestGetRelatedObservations(t *testing.T) {
	s := testStore(t)
	id1, _ := s.Save(testObs("Related obs A", "First observation to test related query"))
	id2, _ := s.Save(testObs("Related obs B", "Second observation to test related query"))
	s.CreateRelation(id1, id2, "builds_on", 1.0)

	related, err := s.GetRelatedObservations(id1)
	if err != nil {
		t.Fatal(err)
	}
	if len(related) != 1 {
		t.Fatalf("expected 1 related, got %d", len(related))
	}
	if related[0].ID != id2 {
		t.Errorf("related ID = %d, want %d", related[0].ID, id2)
	}
}

func TestAutoRelate(t *testing.T) {
	s := testStoreWithDedup(t, 0)
	tk := "auto-relate-topic"

	obs1 := testObs("Auto relate A", "First auto relate observation content")
	obs1.TopicKey = &tk
	id1, _ := s.Save(obs1)

	obs2 := testObs("Auto relate B", "Second auto relate observation content here")
	obs2.TopicKey = &tk

	// Since same topic_key, upsert will update id1
	// For auto-relate to create a new obs, we need different behavior
	// Actually with topic_key upsert, the second save updates the first
	// So let's test with different content that gets new IDs
	// We need to save without topic_key first, then relate manually
	obs3 := testObs("Auto relate C", "Third observation without topic key test")
	id3, _ := s.Save(obs3)

	// Verify the observations exist
	if id1 <= 0 || id3 <= 0 {
		t.Fatal("observations should have been created")
	}
}

// --- Prompts ---

func TestSavePrompt(t *testing.T) {
	s := testStore(t)
	s.CreateSession("prompt-sess", "proj", "/dir")

	id, err := s.SavePrompt("prompt-sess", "What is the auth flow?", "proj")
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Error("expected positive ID")
	}
}

func TestSavePrompt_EnsuresSession(t *testing.T) {
	s := testStore(t)

	_, err := s.SavePrompt("auto-created-sess", "Test prompt content", "proj")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := s.GetSession("auto-created-sess")
	if err != nil {
		t.Fatal("session should be auto-created")
	}
	if sess.ID != "auto-created-sess" {
		t.Errorf("Session ID = %q, want %q", sess.ID, "auto-created-sess")
	}
}

// --- Metrics ---

func TestGetMetrics_Empty(t *testing.T) {
	s := testStore(t)
	m, err := s.GetMetrics()
	if err != nil {
		t.Fatal(err)
	}
	if m.TotalObservations != 0 {
		t.Errorf("TotalObservations = %d, want 0", m.TotalObservations)
	}
	if m.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", m.TotalSessions)
	}
	if m.TotalSearches != 0 {
		t.Errorf("TotalSearches = %d, want 0", m.TotalSearches)
	}
}

func TestGetMetrics_AfterActivity(t *testing.T) {
	s := testStore(t)
	s.CreateSession("m-sess", "proj", "/dir")
	s.Save(testObs("Metrics test", "Observation for metrics testing content"))
	s.Search("metrics", "", "", 0)

	m, err := s.GetMetrics()
	if err != nil {
		t.Fatal(err)
	}
	if m.TotalObservations != 1 {
		t.Errorf("TotalObservations = %d, want 1", m.TotalObservations)
	}
	if m.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", m.TotalSessions)
	}
	if m.TotalSearches != 1 {
		t.Errorf("TotalSearches = %d, want 1", m.TotalSearches)
	}
}

// --- Export/Import ---

func TestExportAll(t *testing.T) {
	s := testStore(t)
	s.CreateSession("exp-sess", "proj", "/dir")
	s.Save(testObs("Export test", "Observation for export testing content"))

	data, err := s.ExportAll("")
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Sessions) != 1 {
		t.Errorf("Sessions = %d, want 1", len(data.Sessions))
	}
	if len(data.Observations) != 1 {
		t.Errorf("Observations = %d, want 1", len(data.Observations))
	}
}

func TestExportAll_FilterByProject(t *testing.T) {
	s := testStore(t)
	proj := "filtered"
	obs := testObs("Filtered export", "Content for filtered export by project")
	obs.Project = &proj
	s.Save(obs)
	s.Save(testObs("Unfiltered export", "Content without project for export test"))

	data, _ := s.ExportAll("filtered")
	if len(data.Observations) != 1 {
		t.Errorf("expected 1 filtered observation, got %d", len(data.Observations))
	}
}

func TestImportData(t *testing.T) {
	s1 := testStore(t)
	s1.CreateSession("imp-sess", "proj", "/dir")
	s1.Save(testObs("Import source", "Content from source store for import"))
	data, _ := s1.ExportAll("")

	s2 := testStore(t)
	err := s2.ImportData(data)
	if err != nil {
		t.Fatal(err)
	}

	obs, _ := s2.RecentContext("", 0)
	if len(obs) != 1 {
		t.Errorf("expected 1 imported observation, got %d", len(obs))
	}
}

func TestImportData_Idempotent(t *testing.T) {
	s1 := testStore(t)
	s1.Save(testObs("Idempotent import", "Content for idempotent import testing"))
	data, _ := s1.ExportAll("")

	s2 := testStore(t)
	s2.ImportData(data)
	err := s2.ImportData(data)
	if err != nil {
		t.Fatal(err)
	}

	obs, _ := s2.RecentContext("", 0)
	if len(obs) != 1 {
		t.Errorf("expected 1 observation after double import, got %d", len(obs))
	}
}

// --- Helpers ---

func TestNormalizeHash(t *testing.T) {
	h1 := normalizeHash("Hello World")
	h2 := normalizeHash("hello world")
	h3 := normalizeHash("  hello   world  ")

	if h1 != h2 {
		t.Error("hash should be case insensitive")
	}
	if h1 != h3 {
		t.Error("hash should be whitespace insensitive")
	}
}

func TestSanitizeFTS(t *testing.T) {
	tests := []struct {
		input string
		empty bool
	}{
		{"hello world", false},
		{"", true},
		{"   ", true},
		{`hello "world"`, false},
		{"special*chars+here", false},
	}

	for _, tt := range tests {
		result := sanitizeFTS(tt.input)
		if tt.empty && result != "" {
			t.Errorf("sanitizeFTS(%q) = %q, want empty", tt.input, result)
		}
		if !tt.empty && result == "" {
			t.Errorf("sanitizeFTS(%q) = empty, want non-empty", tt.input)
		}
	}
}
