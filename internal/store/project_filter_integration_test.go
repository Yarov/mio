package store

import "testing"

func TestSearch_FuzzyProjectFilter_SeparatorsAndCase(t *testing.T) {
	s := testStore(t)

	storedName := "element-adds"
	obs := testObs("Project filter search", "Content for project filter integration search")
	obs.Project = &storedName
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}

	for _, variant := range []string{"elementAdds", "ELEMENT_ADDS", "element adds"} {
		results, err := s.Search("integration", variant, "", 10)
		if err != nil {
			t.Fatalf("variant %q: %v", variant, err)
		}
		if len(results) != 1 {
			t.Fatalf("variant %q: want 1 result, got %d", variant, len(results))
		}
	}
}

func TestSearch_ProjectFilter_DoesNotOvermatchDifferentPunctuation(t *testing.T) {
	s := testStore(t)

	storedName := "my.app"
	obs := testObs("Dot project", "Content for punctuation-sensitive filter")
	obs.Project = &storedName
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}

	results, err := s.Search("punctuation", "my-app", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for my-app vs my.app, got %d", len(results))
	}

	results, err = s.Search("punctuation", "my.app", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 exact punctuation match, got %d", len(results))
	}
}

func TestRecentContext_FuzzyProjectFilter(t *testing.T) {
	s := testStore(t)

	storedName := "element-adds"
	obs := testObs("Context project", "Context filter should match by folded project key")
	obs.Project = &storedName
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}

	results, err := s.RecentContext("Element_Adds", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 context result, got %d", len(results))
	}
}

func TestRecentSessions_FuzzyProjectFilter(t *testing.T) {
	s := testStore(t)

	if err := s.CreateSession("sess-fuzzy", "element-adds", "/tmp"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession("sess-other", "another-project", "/tmp"); err != nil {
		t.Fatal(err)
	}

	results, err := s.RecentSessions("Element_Adds", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 session, got %d", len(results))
	}
	if results[0].ID != "sess-fuzzy" {
		t.Fatalf("unexpected session %q", results[0].ID)
	}
}

func TestEnhancedSearch_FuzzyProjectFilter(t *testing.T) {
	s := testStore(t)

	storedName := "element-adds"
	obs := testObs("Enhanced search project", "ranked enhanced filter query tokens")
	obs.Project = &storedName
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}

	results, err := s.EnhancedSearch("enhanced", "element_adds", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 enhanced-search result, got %d", len(results))
	}
}

func TestAgentContributions_FuzzyProjectFilter(t *testing.T) {
	s := testStore(t)

	p1 := "element-adds"
	obs1 := testObs("Agent one", "Agent contribution one")
	obs1.Project = &p1
	obs1.Agent = "cursor"
	if _, err := s.Save(obs1); err != nil {
		t.Fatal(err)
	}

	p2 := "other-project"
	obs2 := testObs("Agent two", "Agent contribution two")
	obs2.Project = &p2
	obs2.Agent = "claude-code"
	if _, err := s.Save(obs2); err != nil {
		t.Fatal(err)
	}

	results, err := s.AgentContributions("ELEMENT_ADDS", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly one agent in filtered contributions, got %d", len(results))
	}
	if _, ok := results["cursor"]; !ok {
		t.Fatalf("expected cursor agent in filtered contributions")
	}
}

func TestExportAll_FuzzyProjectFilter(t *testing.T) {
	s := testStore(t)

	if err := s.CreateSession("sess-export-fuzzy", "element-adds", "/tmp"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession("sess-export-other", "other-project", "/tmp"); err != nil {
		t.Fatal(err)
	}

	project := "element-adds"
	obs := testObs("Export fuzzy", "Export should include only folded project data")
	obs.Project = &project
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}

	otherProject := "other-project"
	otherObs := testObs("Export other", "Should be excluded by project filter")
	otherObs.Project = &otherProject
	if _, err := s.Save(otherObs); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SavePrompt("sess-export-fuzzy", "prompt for fuzzy export", "element-adds"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SavePrompt("sess-export-other", "prompt for other export", "other-project"); err != nil {
		t.Fatal(err)
	}

	data, err := s.ExportAll("Element_Adds")
	if err != nil {
		t.Fatal(err)
	}

	if len(data.Sessions) != 1 {
		t.Fatalf("expected 1 session in export, got %d", len(data.Sessions))
	}
	if len(data.Observations) != 1 {
		t.Fatalf("expected 1 observation in export, got %d", len(data.Observations))
	}
	if len(data.Prompts) != 1 {
		t.Fatalf("expected 1 prompt in export, got %d", len(data.Prompts))
	}
}
