package store

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

// stopWords contains common English and Spanish words to filter from keyword extraction.
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "this": true,
	"with": true, "are": true, "was": true, "from": true, "have": true,
	"has": true, "been": true, "will": true, "can": true, "como": true,
	"que": true, "por": true, "para": true, "con": true, "una": true,
	"los": true, "las": true, "del": true, "est": true, "pero": true,
}

// DecayAndGC applies importance decay to stale memories and archives truly dead ones.
// Returns (decayed count, archived count, error).
func (s *Store) DecayAndGC(staleDays int, archiveThreshold float64) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if staleDays <= 0 {
		staleDays = 30
	}
	if archiveThreshold <= 0 {
		archiveThreshold = 0.1
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -staleDays).Format(time.RFC3339)

	// Find stale observations: last_accessed < cutoff OR (last_accessed IS NULL AND created_at < cutoff)
	rows, err := s.db.Query(
		`SELECT `+obsColumns+` FROM observations
		 WHERE deleted_at IS NULL
		   AND (
		       (last_accessed IS NOT NULL AND last_accessed < ?)
		       OR (last_accessed IS NULL AND created_at < ?)
		   )`,
		cutoff, cutoff,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("query stale observations: %w", err)
	}
	defer rows.Close()

	type staleObs struct {
		Observation
		newImportance float64
	}
	var stale []staleObs
	for rows.Next() {
		var obs Observation
		var sessionID sql.NullString
		if err := rows.Scan(scanObsFields(&obs, &sessionID)...); err != nil {
			return 0, 0, fmt.Errorf("scan stale observation: %w", err)
		}
		obs.SessionID = sessionID.String
		newImp := obs.Importance * 0.8
		stale = append(stale, staleObs{Observation: obs, newImportance: newImp})
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterate stale observations: %w", err)
	}

	decayed := 0
	archived := 0
	nowStr := now.Format(time.RFC3339)

	for _, so := range stale {
		if so.newImportance < archiveThreshold {
			// Archive: move to memory_archive and soft-delete
			_, err := s.db.Exec(
				`INSERT INTO memory_archive (original_id, sync_id, type, title, content, project, topic_key, importance, access_count, agent, created_at, archived_at, reason)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'decay')`,
				so.ID, so.SyncID, so.Type, so.Title, so.Content,
				so.Project, so.TopicKey, so.newImportance, so.AccessCount,
				so.Agent, so.CreatedAt, nowStr,
			)
			if err != nil {
				return decayed, archived, fmt.Errorf("archive observation %d: %w", so.ID, err)
			}
			// Soft-delete
			_, err = s.db.Exec(
				`UPDATE observations SET deleted_at = ?, updated_at = ? WHERE id = ?`,
				nowStr, nowStr, so.ID,
			)
			if err != nil {
				return decayed, archived, fmt.Errorf("soft-delete observation %d: %w", so.ID, err)
			}
			archived++
		} else {
			// Decay: reduce importance
			_, err := s.db.Exec(
				`UPDATE observations SET importance = ?, updated_at = ? WHERE id = ?`,
				so.newImportance, nowStr, so.ID,
			)
			if err != nil {
				return decayed, archived, fmt.Errorf("decay observation %d: %w", so.ID, err)
			}
			decayed++
		}
	}

	return decayed, archived, nil
}

// Consolidate groups observations by topic_key and merges duplicates for a given project.
// Returns (consolidated count, error).
func (s *Store) Consolidate(project string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find topic_keys that have 3+ non-deleted observations
	q := `SELECT topic_key, COUNT(*) as cnt
	       FROM observations
	       WHERE deleted_at IS NULL AND topic_key IS NOT NULL AND topic_key != ''`
	args := []interface{}{}
	if project != "" {
		q += ` AND project = ?`
		args = append(args, project)
	}
	q += ` GROUP BY topic_key HAVING cnt >= 3`

	topicRows, err := s.db.Query(q, args...)
	if err != nil {
		return 0, fmt.Errorf("query topic groups: %w", err)
	}
	defer topicRows.Close()

	type topicGroup struct {
		topicKey string
		count    int
	}
	var groups []topicGroup
	for topicRows.Next() {
		var tg topicGroup
		if err := topicRows.Scan(&tg.topicKey, &tg.count); err != nil {
			return 0, fmt.Errorf("scan topic group: %w", err)
		}
		groups = append(groups, tg)
	}
	if err := topicRows.Err(); err != nil {
		return 0, fmt.Errorf("iterate topic groups: %w", err)
	}

	consolidated := 0
	nowStr := time.Now().UTC().Format(time.RFC3339)

	for _, g := range groups {
		// Get all observations for this topic_key, ordered by created_at DESC (most recent first)
		obsQ := `SELECT ` + obsColumns + ` FROM observations
		         WHERE deleted_at IS NULL AND topic_key = ?`
		obsArgs := []interface{}{g.topicKey}
		if project != "" {
			obsQ += ` AND project = ?`
			obsArgs = append(obsArgs, project)
		}
		obsQ += ` ORDER BY created_at DESC`

		obsRows, err := s.db.Query(obsQ, obsArgs...)
		if err != nil {
			return consolidated, fmt.Errorf("query observations for topic %s: %w", g.topicKey, err)
		}

		var observations []Observation
		for obsRows.Next() {
			var obs Observation
			var sessionID sql.NullString
			if err := obsRows.Scan(scanObsFields(&obs, &sessionID)...); err != nil {
				obsRows.Close()
				return consolidated, fmt.Errorf("scan observation: %w", err)
			}
			obs.SessionID = sessionID.String
			observations = append(observations, obs)
		}
		obsRows.Close()
		if len(observations) < 3 {
			continue
		}

		// Keep the most recent (first), merge older ones into it
		keeper := observations[0]
		older := observations[1:]

		// Build consolidated content
		var titles []string
		for _, o := range older {
			titles = append(titles, fmt.Sprintf("- %s", o.Title))
		}
		mergedContent := keeper.Content + fmt.Sprintf("\n\n--- Consolidated from %d memories ---\n%s", len(older), strings.Join(titles, "\n"))

		// Update the keeper
		_, err = s.db.Exec(
			`UPDATE observations SET content = ?, consolidated = 1, updated_at = ? WHERE id = ?`,
			mergedContent, nowStr, keeper.ID,
		)
		if err != nil {
			return consolidated, fmt.Errorf("update keeper %d: %w", keeper.ID, err)
		}

		// Soft-delete the older duplicates
		for _, o := range older {
			_, err = s.db.Exec(
				`UPDATE observations SET deleted_at = ?, updated_at = ? WHERE id = ?`,
				nowStr, nowStr, o.ID,
			)
			if err != nil {
				return consolidated, fmt.Errorf("soft-delete observation %d: %w", o.ID, err)
			}
			consolidated++
		}
	}

	return consolidated, nil
}

// CrossProjectSearch searches across ALL projects, prioritizing global scope.
// Returns results from all projects sorted by relevance.
func (s *Store) CrossProjectSearch(query string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = s.cfg.MaxSearchResults
	}

	sanitized := sanitizeFTS(query)
	if sanitized == "" {
		return nil, fmt.Errorf("empty search query")
	}

	q := `SELECT ` + obsColumnsAliased + `, rank
	      FROM observations_fts f
	      JOIN observations o ON o.id = f.rowid
	      WHERE observations_fts MATCH ? AND o.deleted_at IS NULL
	      ORDER BY rank LIMIT ?`

	rows, err := s.db.Query(q, sanitized, limit*3) // fetch extra to allow for re-ranking
	if err != nil {
		return nil, fmt.Errorf("cross-project search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		var rank float64
		var sessionID sql.NullString
		fields := scanObsFields(&sr.Observation, &sessionID)
		fields = append(fields, &rank)
		if err := rows.Scan(fields...); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		sr.SessionID = sessionID.String
		sr.Score = -rank // FTS5 rank is negative, lower = better
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate results: %w", err)
	}

	// Apply scope boosting, temporal decay, and importance boost
	for i := range results {
		// Scope boost
		switch results[i].Scope {
		case "global":
			results[i].Score *= 2.0
		case "personal":
			results[i].Score *= 1.5
		}

		// Temporal decay
		created, err := time.Parse(time.RFC3339, results[i].CreatedAt)
		if err == nil {
			age := time.Since(created).Hours() / 24
			decay := math.Exp(-0.01 * age)
			results[i].Score *= decay
		}

		// Importance boost
		results[i].Score *= (0.7 + 0.3*results[i].Importance)
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Cap at limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// BuildDecisionGraph traverses relations from a focal observation, up to maxDepth.
func (s *Store) BuildDecisionGraph(focalID int64, maxDepth int) (*DecisionGraph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if maxDepth <= 0 {
		maxDepth = 3
	}

	graph := &DecisionGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	visited := map[int64]bool{}
	edgeSet := map[string]bool{}
	queue := []struct {
		id    int64
		depth int
	}{{id: focalID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true

		// Fetch the observation for this node
		row := s.db.QueryRow(`SELECT `+obsColumns+` FROM observations WHERE id = ? AND deleted_at IS NULL`, current.id)
		var obs Observation
		var sessionID sql.NullString
		err := row.Scan(scanObsFields(&obs, &sessionID)...)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("fetch node %d: %w", current.id, err)
		}

		node := GraphNode{
			ID:      obs.ID,
			Title:   obs.Title,
			Type:    obs.Type,
			IsFocus: obs.ID == focalID,
		}
		graph.Nodes = append(graph.Nodes, node)

		// Stop expanding if we've reached max depth
		if current.depth >= maxDepth {
			continue
		}

		// Fetch relations
		relRows, err := s.db.Query(
			`SELECT id, from_id, to_id, type, strength, created_at FROM relations WHERE from_id = ? OR to_id = ?`,
			current.id, current.id,
		)
		if err != nil {
			return nil, fmt.Errorf("fetch relations for %d: %w", current.id, err)
		}

		var rels []Relation
		for relRows.Next() {
			var r Relation
			if err := relRows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Strength, &r.CreatedAt); err != nil {
				relRows.Close()
				return nil, fmt.Errorf("scan relation: %w", err)
			}
			rels = append(rels, r)
		}
		relRows.Close()

		for _, r := range rels {
			edgeKey := fmt.Sprintf("%d-%d-%s", r.FromID, r.ToID, r.Type)
			if !edgeSet[edgeKey] {
				edgeSet[edgeKey] = true
				graph.Edges = append(graph.Edges, GraphEdge{
					FromID:   r.FromID,
					ToID:     r.ToID,
					Type:     r.Type,
					Strength: r.Strength,
				})
			}

			// Enqueue the other side of the relation
			neighborID := r.ToID
			if neighborID == current.id {
				neighborID = r.FromID
			}
			if !visited[neighborID] {
				queue = append(queue, struct {
					id    int64
					depth int
				}{id: neighborID, depth: current.depth + 1})
			}
		}
	}

	return graph, nil
}

// SurfaceRelevant finds memories that match keywords extracted from the given text.
// Returns top matches that the agent should be reminded about.
func (s *Store) SurfaceRelevant(text string, project string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 3
	}

	// Extract meaningful keywords
	keywords := extractKeywords(text)
	if len(keywords) == 0 {
		return nil, nil
	}

	// Build FTS query: OR keywords together
	quoted := make([]string, len(keywords))
	for i, kw := range keywords {
		clean := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' || r == '+' || r == '-' || r == '(' || r == ')' || r == ':' || r == '^' {
				return -1
			}
			return r
		}, kw)
		if clean != "" {
			quoted[i] = `"` + clean + `"`
		}
	}
	ftsQuery := strings.Join(quoted, " OR ")
	if ftsQuery == "" {
		return nil, nil
	}

	q := `SELECT ` + obsColumnsAliased + `, rank
	      FROM observations_fts f
	      JOIN observations o ON o.id = f.rowid
	      WHERE observations_fts MATCH ? AND o.deleted_at IS NULL`
	args := []interface{}{ftsQuery}

	if project != "" {
		q += ` AND o.project = ?`
		args = append(args, project)
	}

	q += ` ORDER BY rank LIMIT ?`
	args = append(args, limit*5) // fetch extra for re-ranking

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("surface relevant search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		var rank float64
		var sessionID sql.NullString
		fields := scanObsFields(&sr.Observation, &sessionID)
		fields = append(fields, &rank)
		if err := rows.Scan(fields...); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		sr.SessionID = sessionID.String
		sr.Score = -rank
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate results: %w", err)
	}

	// Blend vector similarity if enabled
	results = s.blendVectorScores(text, results)

	// Apply importance and recency boost
	for i := range results {
		// Importance boost
		results[i].Score *= (0.5 + 0.5*results[i].Importance)

		// Recency boost
		created, err := time.Parse(time.RFC3339, results[i].CreatedAt)
		if err == nil {
			age := time.Since(created).Hours() / 24
			decay := math.Exp(-0.01 * age)
			results[i].Score *= decay
		}

		// Access frequency boost
		results[i].Score *= math.Log2(float64(results[i].AccessCount) + 2)
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Normalize scores to 0-10 scale (BM25 ranks can be very small for single-term queries)
	if len(results) > 0 {
		maxScore := results[0].Score
		if maxScore > 0 {
			for i := range results {
				results[i].Score = (results[i].Score / maxScore) * 10
			}
		}
	}

	// Filter by threshold (normalized scale: 5% of max) and cap at limit
	var filtered []SearchResult
	for _, r := range results {
		if r.Score > 0.5 {
			filtered = append(filtered, r)
		}
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

// extractKeywords splits text into meaningful keywords for search.
func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	seen := map[string]bool{}
	var keywords []string

	for _, w := range words {
		// Strip punctuation from edges
		w = strings.Trim(w, ".,;:!?\"'`()[]{}#@$%^&*~<>/\\|")
		if len(w) < 4 {
			continue
		}
		if stopWords[w] {
			continue
		}
		if seen[w] {
			continue
		}
		seen[w] = true
		keywords = append(keywords, w)
		if len(keywords) >= 8 {
			break
		}
	}

	return keywords
}
