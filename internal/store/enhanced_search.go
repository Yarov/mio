package store

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

// EnhancedSearch is like Search but with better scoring: scope-aware boosting,
// revision boost (frequently updated = more valuable), and agent diversity boost.
func (s *Store) EnhancedSearch(query string, project string, obsType string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = s.cfg.MaxSearchResults
	}

	sanitized := sanitizeFTS(query)
	if sanitized == "" {
		return nil, fmt.Errorf("empty search query")
	}

	q := `SELECT ` + obsColumnsAliased + `, rank,
		julianday('now') - julianday(o.created_at) AS age_days
		FROM observations_fts f
		JOIN observations o ON o.id = f.rowid
		WHERE observations_fts MATCH ?
		AND o.deleted_at IS NULL`
	args := []interface{}{sanitized}

	q, args = appendProjectFilter(q, args, "o.project", project)
	if obsType != "" {
		q += ` AND o.type = ?`
		args = append(args, obsType)
	}

	// Fetch more than needed so we can re-rank and trim.
	fetchLimit := limit * 3
	if fetchLimit < 50 {
		fetchLimit = 50
	}
	q += ` ORDER BY rank LIMIT ?`
	args = append(args, fetchLimit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("enhanced search query: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	_ = now // used indirectly via SQL age_days

	type scoredResult struct {
		sr   SearchResult
		rank float64
	}

	var scored []scoredResult
	for rows.Next() {
		var obs Observation
		var sessionID sql.NullString
		var rank float64
		var ageDays float64

		fields := scanObsFields(&obs, &sessionID)
		fields = append(fields, &rank, &ageDays)
		if err := rows.Scan(fields...); err != nil {
			return nil, fmt.Errorf("enhanced search scan: %w", err)
		}
		obs.SessionID = sessionID.String

		// base_score: FTS5 rank is negative, so negate it.
		baseScore := -rank

		// temporal_decay = exp(-0.01 * age_days)
		temporalDecay := math.Exp(-0.01 * ageDays)

		// importance_boost = 0.7 + 0.3 * importance
		importanceBoost := 0.7 + 0.3*obs.Importance

		// access_boost = log2(access_count + 2)
		accessBoost := math.Log2(float64(obs.AccessCount) + 2)

		// revision_boost = 1.0 + 0.1 * min(revision_count, 10)
		revCount := obs.RevisionCount
		if revCount > 10 {
			revCount = 10
		}
		revisionBoost := 1.0 + 0.1*float64(revCount)

		// scope_boost
		var scopeBoost float64
		switch obs.Scope {
		case "global":
			scopeBoost = 2.0
		case "personal":
			scopeBoost = 1.5
		default:
			scopeBoost = 1.0
		}

		// consolidated_penalty
		consolidatedPenalty := 1.0
		if obs.Consolidated == 1 {
			consolidatedPenalty = 0.5
		}

		finalScore := baseScore * temporalDecay * importanceBoost * accessBoost * revisionBoost * scopeBoost * consolidatedPenalty

		sr := SearchResult{
			Observation: obs,
			Score:       finalScore,
		}
		scored = append(scored, scoredResult{sr: sr, rank: finalScore})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("enhanced search rows: %w", err)
	}

	// Blend vector similarity if enabled
	if s.embedder != nil {
		srSlice := make([]SearchResult, len(scored))
		for i := range scored {
			srSlice[i] = scored[i].sr
		}
		srSlice = s.blendVectorScores(query, srSlice)
		for i := range scored {
			scored[i].sr = srSlice[i]
			scored[i].rank = srSlice[i].Score
		}
	}

	// Sort by final_score descending.
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].rank > scored[i].rank {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Normalize scores to 0-10 scale (BM25 ranks can be very small for single-term queries)
	if len(scored) > 0 {
		maxScore := scored[0].rank
		if maxScore > 0 {
			for i := range scored {
				scored[i].sr.Score = (scored[i].rank / maxScore) * 10
			}
		}
	}

	results := make([]SearchResult, len(scored))
	for i, s := range scored {
		results[i] = s.sr
	}
	return results, nil
}

// AgentContributions returns observations grouped by agent for a project.
func (s *Store) AgentContributions(project string, limit int) (map[string][]Observation, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get agents ordered by contribution count.
	agentQ := `SELECT agent, COUNT(*) AS cnt
		FROM observations
		WHERE deleted_at IS NULL`
	agentArgs := []interface{}{}
	agentQ, agentArgs = appendProjectFilter(agentQ, agentArgs, "project", project)
	agentQ += ` GROUP BY agent
		ORDER BY cnt DESC`
	agentRows, err := s.db.Query(agentQ, agentArgs...)
	if err != nil {
		return nil, fmt.Errorf("agent contributions query: %w", err)
	}
	defer agentRows.Close()

	var agents []string
	for agentRows.Next() {
		var agent string
		var cnt int
		if err := agentRows.Scan(&agent, &cnt); err != nil {
			return nil, fmt.Errorf("agent contributions scan: %w", err)
		}
		agents = append(agents, agent)
	}
	if err := agentRows.Err(); err != nil {
		return nil, fmt.Errorf("agent contributions rows: %w", err)
	}

	result := make(map[string][]Observation, len(agents))
	for _, agent := range agents {
		obsQ := `SELECT ` + obsColumns + ` FROM observations
			WHERE agent = ? AND deleted_at IS NULL`
		obsArgs := []interface{}{agent}
		obsQ, obsArgs = appendProjectFilter(obsQ, obsArgs, "project", project)
		obsQ += ` ORDER BY importance DESC
			LIMIT ?`
		obsArgs = append(obsArgs, limit)
		rows, err := s.db.Query(obsQ, obsArgs...)
		if err != nil {
			return nil, fmt.Errorf("agent observations query: %w", err)
		}

		var obs []Observation
		for rows.Next() {
			o, err := s.scanObservationRows(rows)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("agent observations scan: %w", err)
			}
			obs = append(obs, *o)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("agent observations rows: %w", err)
		}
		result[agent] = obs
	}

	return result, nil
}

// AgentKnowledge returns what a specific agent has learned across projects.
func (s *Store) AgentKnowledge(agentName string, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(
		`SELECT `+obsColumns+` FROM observations
		WHERE agent = ? AND deleted_at IS NULL
		ORDER BY importance DESC, created_at DESC
		LIMIT ?`,
		agentName, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("agent knowledge query: %w", err)
	}
	defer rows.Close()

	var results []Observation
	for rows.Next() {
		obs, err := s.scanObservationRows(rows)
		if err != nil {
			return nil, fmt.Errorf("agent knowledge scan: %w", err)
		}
		results = append(results, *obs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("agent knowledge rows: %w", err)
	}
	return results, nil
}

// SetObservationAgent sets the agent field for an observation.
func (s *Store) SetObservationAgent(id int64, agent string) error {
	res, err := s.db.Exec(
		`UPDATE observations SET agent = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		agent, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("set observation agent: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set observation agent rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("observation %d not found or already deleted", id)
	}
	return nil
}
