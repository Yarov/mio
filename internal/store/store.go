package store

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"mio/internal/config"
)

// --- Types ---

type Observation struct {
	ID             int64
	SyncID         string
	SessionID      string
	Type           string
	Title          string
	Content        string
	ToolName       *string
	Project        *string
	Scope          string
	TopicKey       *string
	NormalizedHash string
	RevisionCount  int
	DuplicateCount int
	Importance     float64
	AccessCount    int
	LastAccessed   *string
	LastSeenAt     *string
	CreatedAt      string
	UpdatedAt      string
	DeletedAt      *string
}

type Session struct {
	ID        string
	Project   string
	Directory string
	StartedAt string
	EndedAt   *string
	Summary   *string
}

type Prompt struct {
	ID        int64
	SyncID    string
	SessionID string
	Content   string
	Project   string
	CreatedAt string
}

type SearchResult struct {
	Observation
	Score float64
}

type SessionSummary struct {
	Session
	ObservationCount int
}

type TimelineEntry struct {
	Observation
	IsFocus bool
}

type Relation struct {
	ID        int64
	FromID    int64
	ToID      int64
	Type      string
	Strength  float64
	CreatedAt string
}

type Metrics struct {
	TotalObservations  int
	TotalSessions      int
	TotalSearches      int
	SearchHitRate      float64
	AvgSearchLatencyMs int
	TopProjects        []ProjectStat
	StaleMemoryCount   int
}

type ProjectStat struct {
	Project string
	Count   int
}

type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Fields, "; ")
}

// Valid observation types
var validTypes = map[string]bool{
	"bugfix":       true,
	"decision":     true,
	"architecture": true,
	"discovery":    true,
	"pattern":      true,
	"config":       true,
	"preference":   true,
	"learning":     true,
	"summary":      true,
}

// --- Store ---

type Store struct {
	db     *sql.DB
	cfg    *config.Config
	mu     sync.RWMutex
	topicMu sync.Map // per-topic mutex
}

func New(cfg *config.Config) (*Store, error) {
	if err := cfg.EnsureDataDir(); err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer
	db.SetMaxIdleConns(2)

	s := &Store{db: db, cfg: cfg}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		project    TEXT NOT NULL DEFAULT '',
		directory  TEXT NOT NULL DEFAULT '',
		started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		ended_at   TEXT,
		summary    TEXT
	);

	CREATE TABLE IF NOT EXISTS observations (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		sync_id          TEXT UNIQUE NOT NULL,
		session_id       TEXT REFERENCES sessions(id),
		type             TEXT NOT NULL,
		title            TEXT NOT NULL,
		content          TEXT NOT NULL,
		tool_name        TEXT,
		project          TEXT,
		scope            TEXT DEFAULT 'project',
		topic_key        TEXT,
		normalized_hash  TEXT NOT NULL,
		revision_count   INTEGER DEFAULT 0,
		duplicate_count  INTEGER DEFAULT 0,
		importance       REAL DEFAULT 0.5,
		access_count     INTEGER DEFAULT 0,
		last_accessed    TEXT,
		last_seen_at     TEXT,
		created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		deleted_at       TEXT
	);

	CREATE TABLE IF NOT EXISTS user_prompts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		sync_id    TEXT UNIQUE NOT NULL,
		session_id TEXT REFERENCES sessions(id),
		content    TEXT NOT NULL,
		project    TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	);

	CREATE TABLE IF NOT EXISTS relations (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		from_id    INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
		to_id      INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
		type       TEXT NOT NULL,
		strength   REAL DEFAULT 1.0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		UNIQUE(from_id, to_id, type)
	);

	CREATE TABLE IF NOT EXISTS search_log (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		query        TEXT NOT NULL,
		result_count INTEGER DEFAULT 0,
		top_hit_id   INTEGER,
		search_type  TEXT DEFAULT 'fts',
		latency_ms   INTEGER DEFAULT 0,
		created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	);

	CREATE INDEX IF NOT EXISTS idx_obs_session ON observations(session_id);
	CREATE INDEX IF NOT EXISTS idx_obs_type ON observations(type);
	CREATE INDEX IF NOT EXISTS idx_obs_project ON observations(project);
	CREATE INDEX IF NOT EXISTS idx_obs_created ON observations(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_obs_scope ON observations(scope);
	CREATE INDEX IF NOT EXISTS idx_obs_sync ON observations(sync_id);
	CREATE INDEX IF NOT EXISTS idx_obs_topic ON observations(topic_key);
	CREATE INDEX IF NOT EXISTS idx_obs_deleted ON observations(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_obs_hash ON observations(normalized_hash);
	CREATE INDEX IF NOT EXISTS idx_obs_importance ON observations(importance DESC);
	CREATE INDEX IF NOT EXISTS idx_rel_from ON relations(from_id);
	CREATE INDEX IF NOT EXISTS idx_rel_to ON relations(to_id);
	CREATE INDEX IF NOT EXISTS idx_prompts_session ON user_prompts(session_id);
	CREATE INDEX IF NOT EXISTS idx_prompts_sync ON user_prompts(sync_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	// FTS5 virtual tables - created separately since IF NOT EXISTS behaves differently
	fts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
			title, content, tool_name, type, project,
			content='observations', content_rowid='id'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS prompts_fts USING fts5(
			content, project,
			content='user_prompts', content_rowid='id'
		)`,
	}

	for _, q := range fts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("create fts: %w", err)
		}
	}

	// Triggers to keep FTS in sync
	triggers := []string{
		`CREATE TRIGGER IF NOT EXISTS obs_ai AFTER INSERT ON observations BEGIN
			INSERT INTO observations_fts(rowid, title, content, tool_name, type, project)
			VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project);
		END`,
		`CREATE TRIGGER IF NOT EXISTS obs_ad AFTER DELETE ON observations BEGIN
			INSERT INTO observations_fts(observations_fts, rowid, title, content, tool_name, type, project)
			VALUES ('delete', old.id, old.title, old.content, old.tool_name, old.type, old.project);
		END`,
		`CREATE TRIGGER IF NOT EXISTS obs_au AFTER UPDATE ON observations BEGIN
			INSERT INTO observations_fts(observations_fts, rowid, title, content, tool_name, type, project)
			VALUES ('delete', old.id, old.title, old.content, old.tool_name, old.type, old.project);
			INSERT INTO observations_fts(rowid, title, content, tool_name, type, project)
			VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project);
		END`,
		`CREATE TRIGGER IF NOT EXISTS prompt_ai AFTER INSERT ON user_prompts BEGIN
			INSERT INTO prompts_fts(rowid, content, project)
			VALUES (new.id, new.content, new.project);
		END`,
		`CREATE TRIGGER IF NOT EXISTS prompt_ad AFTER DELETE ON user_prompts BEGIN
			INSERT INTO prompts_fts(prompts_fts, rowid, content, project)
			VALUES ('delete', old.id, old.content, old.project);
		END`,
	}

	for _, t := range triggers {
		if _, err := s.db.Exec(t); err != nil {
			// Ignore "already exists" errors for triggers
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("create trigger: %w", err)
			}
		}
	}

	return nil
}

// --- Validation ---

func (s *Store) Validate(obs *Observation) error {
	var errs []string

	if len(obs.Title) < 3 {
		errs = append(errs, "title must be at least 3 characters")
	}
	if len(obs.Title) > 500 {
		errs = append(errs, "title must be at most 500 characters")
	}
	if len(obs.Content) < 10 {
		errs = append(errs, "content must be at least 10 characters")
	}
	if len(obs.Content) > s.cfg.MaxObservationLength {
		errs = append(errs, fmt.Sprintf("content exceeds max length of %d", s.cfg.MaxObservationLength))
	}
	if !validTypes[obs.Type] {
		errs = append(errs, fmt.Sprintf("invalid type '%s'; valid types: %s", obs.Type, validTypesList()))
	}
	if obs.Scope == "" {
		obs.Scope = "project"
	}
	if obs.Scope != "project" && obs.Scope != "personal" && obs.Scope != "global" {
		errs = append(errs, fmt.Sprintf("invalid scope '%s'; valid: project, personal, global", obs.Scope))
	}
	if obs.Importance < 0 || obs.Importance > 1 {
		errs = append(errs, "importance must be between 0.0 and 1.0")
	}

	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

func validTypesList() string {
	types := make([]string, 0, len(validTypes))
	for t := range validTypes {
		types = append(types, t)
	}
	return strings.Join(types, ", ")
}

// --- Hashing ---

func normalizeHash(content string) string {
	normalized := strings.ToLower(strings.TrimSpace(content))
	normalized = strings.Join(strings.Fields(normalized), " ")
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

// --- Session CRUD ---

func (s *Store) CreateSession(id, project, directory string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, directory) VALUES (?, ?, ?)`,
		id, project, directory,
	)
	return err
}

func (s *Store) EndSession(id string, summary *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE sessions SET ended_at = ?, summary = ? WHERE id = ?`,
		now, summary, id,
	)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, project, directory, started_at, ended_at, summary FROM sessions WHERE id = ?`, id)
	var sess Session
	err := row.Scan(&sess.ID, &sess.Project, &sess.Directory, &sess.StartedAt, &sess.EndedAt, &sess.Summary)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) RecentSessions(project string, limit int) ([]SessionSummary, error) {
	query := `SELECT s.id, s.project, s.directory, s.started_at, s.ended_at, s.summary,
		(SELECT COUNT(*) FROM observations WHERE session_id = s.id AND deleted_at IS NULL) as obs_count
		FROM sessions s WHERE 1=1`
	args := []interface{}{}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}
	query += " ORDER BY s.started_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		if err := rows.Scan(&ss.ID, &ss.Project, &ss.Directory, &ss.StartedAt, &ss.EndedAt, &ss.Summary, &ss.ObservationCount); err != nil {
			return nil, err
		}
		results = append(results, ss)
	}
	return results, rows.Err()
}

// ensureSession creates a session if it doesn't exist
func (s *Store) ensureSession(sessionID, project string) error {
	if sessionID == "" {
		return nil
	}
	return s.CreateSession(sessionID, project, "")
}

// --- Observation CRUD ---

func (s *Store) Save(obs *Observation) (int64, error) {
	if err := s.Validate(obs); err != nil {
		return 0, err
	}

	obs.NormalizedHash = normalizeHash(obs.Content)

	// Check for duplicate within dedup window
	if s.isDuplicate(obs) {
		return 0, fmt.Errorf("duplicate observation detected within dedup window")
	}

	// Handle topic_key upsert
	if obs.TopicKey != nil && *obs.TopicKey != "" {
		return s.upsertByTopicKey(obs)
	}

	if obs.SyncID == "" {
		obs.SyncID = uuid.New().String()
	}

	project := ""
	if obs.Project != nil {
		project = *obs.Project
	}
	if obs.SessionID != "" {
		_ = s.ensureSession(obs.SessionID, project)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		`INSERT INTO observations (sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, importance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		obs.SyncID, nilIfEmpty(obs.SessionID), obs.Type, obs.Title, obs.Content,
		obs.ToolName, obs.Project, obs.Scope, obs.TopicKey, obs.NormalizedHash,
		obs.Importance, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert observation: %w", err)
	}

	id, _ := result.LastInsertId()

	// Auto-detect relations with existing observations
	s.autoRelate(id, obs)

	return id, nil
}

func (s *Store) upsertByTopicKey(obs *Observation) (int64, error) {
	// Lock per topic key
	key := *obs.TopicKey
	mu, _ := s.topicMu.LoadOrStore(key, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	var existingID int64
	err := s.db.QueryRow(
		`SELECT id FROM observations WHERE topic_key = ? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`,
		key,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		// No existing, insert new
		if obs.SyncID == "" {
			obs.SyncID = uuid.New().String()
		}
		now := time.Now().UTC().Format(time.RFC3339)
		result, err := s.db.Exec(
			`INSERT INTO observations (sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, importance, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			obs.SyncID, nilIfEmpty(obs.SessionID), obs.Type, obs.Title, obs.Content,
			obs.ToolName, obs.Project, obs.Scope, obs.TopicKey, obs.NormalizedHash,
			obs.Importance, now, now,
		)
		if err != nil {
			return 0, err
		}
		id, _ := result.LastInsertId()
		return id, nil
	}
	if err != nil {
		return 0, err
	}

	// Update existing
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`UPDATE observations SET title = ?, content = ?, normalized_hash = ?, revision_count = revision_count + 1, updated_at = ? WHERE id = ?`,
		obs.Title, obs.Content, obs.NormalizedHash, now, existingID,
	)
	return existingID, err
}

func (s *Store) isDuplicate(obs *Observation) bool {
	cutoff := time.Now().Add(-s.cfg.DedupeWindow).UTC().Format(time.RFC3339)
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM observations WHERE normalized_hash = ? AND created_at > ? AND deleted_at IS NULL`,
		obs.NormalizedHash, cutoff,
	).Scan(&count)
	if err != nil {
		return false
	}
	if count > 0 {
		// Increment duplicate count on existing
		s.db.Exec(
			`UPDATE observations SET duplicate_count = duplicate_count + 1, last_seen_at = ? WHERE normalized_hash = ? AND deleted_at IS NULL`,
			time.Now().UTC().Format(time.RFC3339), obs.NormalizedHash,
		)
		return true
	}
	return false
}

func (s *Store) GetObservation(id int64) (*Observation, error) {
	obs, err := s.scanObservation(
		s.db.QueryRow(`SELECT id, sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at FROM observations WHERE id = ?`, id),
	)
	if err != nil {
		return nil, err
	}

	// Increment access count
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec(`UPDATE observations SET access_count = access_count + 1, last_accessed = ? WHERE id = ?`, now, id)

	return obs, nil
}

func (s *Store) UpdateObservation(id int64, title, content string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	hash := normalizeHash(content)
	_, err := s.db.Exec(
		`UPDATE observations SET title = ?, content = ?, normalized_hash = ?, revision_count = revision_count + 1, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		title, content, hash, now, id,
	)
	return err
}

func (s *Store) SoftDelete(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE observations SET deleted_at = ? WHERE id = ?`, now, id)
	return err
}

func (s *Store) HardDelete(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM relations WHERE from_id = ? OR to_id = ?`, id, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM observations WHERE id = ?`, id)
	return err
}

// --- Search ---

func (s *Store) Search(query string, project string, obsType string, limit int) ([]SearchResult, error) {
	start := time.Now()

	if limit <= 0 {
		limit = s.cfg.MaxSearchResults
	}

	sanitized := sanitizeFTS(query)
	if sanitized == "" {
		return nil, fmt.Errorf("empty search query")
	}

	q := `SELECT o.id, o.sync_id, o.session_id, o.type, o.title, o.content, o.tool_name, o.project, o.scope, o.topic_key, o.normalized_hash, o.revision_count, o.duplicate_count, o.importance, o.access_count, o.last_accessed, o.last_seen_at, o.created_at, o.updated_at, o.deleted_at, rank
		FROM observations_fts f
		JOIN observations o ON o.id = f.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL`
	args := []interface{}{sanitized}

	if project != "" {
		q += " AND o.project = ?"
		args = append(args, project)
	}
	if obsType != "" {
		q += " AND o.type = ?"
		args = append(args, obsType)
	}

	q += " ORDER BY rank LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		var rank float64
		var sessionID sql.NullString
		if err := rows.Scan(
			&sr.ID, &sr.SyncID, &sessionID, &sr.Type, &sr.Title, &sr.Content,
			&sr.ToolName, &sr.Project, &sr.Scope, &sr.TopicKey, &sr.NormalizedHash,
			&sr.RevisionCount, &sr.DuplicateCount, &sr.Importance, &sr.AccessCount,
			&sr.LastAccessed, &sr.LastSeenAt, &sr.CreatedAt, &sr.UpdatedAt, &sr.DeletedAt,
			&rank,
		); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		sr.SessionID = sessionID.String
		sr.Score = -rank // FTS5 rank is negative, lower = better
		results = append(results, sr)
	}

	// Apply temporal decay and importance boost
	for i := range results {
		created, err := time.Parse(time.RFC3339, results[i].CreatedAt)
		if err == nil {
			age := time.Since(created).Hours() / 24 // days
			decay := math.Exp(-0.01 * age)          // λ = 0.01, half-life ~69 days
			results[i].Score *= decay
		}
		// Importance boost
		results[i].Score *= (0.7 + 0.3*results[i].Importance)
		// Access frequency boost (logarithmic)
		results[i].Score *= math.Log2(float64(results[i].AccessCount) + 2)
	}

	// Re-sort after adjustments
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Log search
	latency := time.Since(start).Milliseconds()
	var topHitID *int64
	if len(results) > 0 {
		topHitID = &results[0].ID
	}
	s.db.Exec(
		`INSERT INTO search_log (query, result_count, top_hit_id, search_type, latency_ms) VALUES (?, ?, ?, 'fts', ?)`,
		query, len(results), topHitID, latency,
	)

	return results, rows.Err()
}

func sanitizeFTS(query string) string {
	words := strings.Fields(strings.TrimSpace(query))
	if len(words) == 0 {
		return ""
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		// Remove FTS5 special characters
		clean := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' || r == '+' || r == '-' || r == '(' || r == ')' || r == ':' || r == '^' {
				return -1
			}
			return r
		}, w)
		if clean != "" {
			quoted[i] = `"` + clean + `"`
		}
	}
	return strings.Join(quoted, " ")
}

// --- Context ---

func (s *Store) RecentContext(project string, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = s.cfg.MaxContextResults
	}

	q := `SELECT id, sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at
		FROM observations WHERE deleted_at IS NULL`
	args := []interface{}{}

	if project != "" {
		q += " AND project = ?"
		args = append(args, project)
	}

	q += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	return s.queryObservations(q, args...)
}

// --- Timeline ---

func (s *Store) Timeline(obsID int64, before, after int) ([]TimelineEntry, error) {
	focus, err := s.GetObservation(obsID)
	if err != nil {
		return nil, fmt.Errorf("get focus observation: %w", err)
	}

	var entries []TimelineEntry

	// Get observations before
	befores, err := s.queryObservations(
		`SELECT id, sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at
		FROM observations WHERE created_at < ? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ?`,
		focus.CreatedAt, before,
	)
	if err != nil {
		return nil, err
	}
	for i := len(befores) - 1; i >= 0; i-- {
		entries = append(entries, TimelineEntry{Observation: befores[i], IsFocus: false})
	}

	entries = append(entries, TimelineEntry{Observation: *focus, IsFocus: true})

	// Get observations after
	afters, err := s.queryObservations(
		`SELECT id, sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at
		FROM observations WHERE created_at > ? AND deleted_at IS NULL ORDER BY created_at ASC LIMIT ?`,
		focus.CreatedAt, after,
	)
	if err != nil {
		return nil, err
	}
	for _, o := range afters {
		entries = append(entries, TimelineEntry{Observation: o, IsFocus: false})
	}

	return entries, nil
}

// --- Relations ---

func (s *Store) CreateRelation(fromID, toID int64, relType string, strength float64) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO relations (from_id, to_id, type, strength) VALUES (?, ?, ?, ?)`,
		fromID, toID, relType, strength,
	)
	return err
}

func (s *Store) GetRelations(obsID int64) ([]Relation, error) {
	rows, err := s.db.Query(
		`SELECT id, from_id, to_id, type, strength, created_at FROM relations WHERE from_id = ? OR to_id = ?`,
		obsID, obsID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relation
	for rows.Next() {
		var r Relation
		if err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Strength, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *Store) GetRelatedObservations(obsID int64) ([]Observation, error) {
	q := `SELECT DISTINCT o.id, o.sync_id, o.session_id, o.type, o.title, o.content, o.tool_name, o.project, o.scope, o.topic_key, o.normalized_hash, o.revision_count, o.duplicate_count, o.importance, o.access_count, o.last_accessed, o.last_seen_at, o.created_at, o.updated_at, o.deleted_at
		FROM observations o
		JOIN relations r ON (r.from_id = o.id OR r.to_id = o.id)
		WHERE (r.from_id = ? OR r.to_id = ?) AND o.id != ? AND o.deleted_at IS NULL
		ORDER BY r.strength DESC, o.created_at DESC`
	return s.queryObservations(q, obsID, obsID, obsID)
}

func (s *Store) autoRelate(newID int64, obs *Observation) {
	// Relate to same topic_key observations
	if obs.TopicKey != nil && *obs.TopicKey != "" {
		rows, err := s.db.Query(
			`SELECT id FROM observations WHERE topic_key = ? AND id != ? AND deleted_at IS NULL`,
			*obs.TopicKey, newID,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var relID int64
				if rows.Scan(&relID) == nil {
					s.CreateRelation(newID, relID, "relates_to", 0.8)
				}
			}
		}
	}
}

// --- Prompts ---

func (s *Store) SavePrompt(sessionID, content, project string) (int64, error) {
	syncID := uuid.New().String()
	if sessionID != "" {
		_ = s.ensureSession(sessionID, project)
	}
	result, err := s.db.Exec(
		`INSERT INTO user_prompts (sync_id, session_id, content, project) VALUES (?, ?, ?, ?)`,
		syncID, sessionID, content, project,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// --- Metrics ---

func (s *Store) GetMetrics() (*Metrics, error) {
	m := &Metrics{}

	s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE deleted_at IS NULL`).Scan(&m.TotalObservations)
	s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&m.TotalSessions)
	s.db.QueryRow(`SELECT COUNT(*) FROM search_log`).Scan(&m.TotalSearches)

	// Hit rate: searches with results > 0
	var hitsWithResults int
	s.db.QueryRow(`SELECT COUNT(*) FROM search_log WHERE result_count > 0`).Scan(&hitsWithResults)
	if m.TotalSearches > 0 {
		m.SearchHitRate = float64(hitsWithResults) / float64(m.TotalSearches) * 100
	}

	// Avg latency
	s.db.QueryRow(`SELECT COALESCE(AVG(latency_ms), 0) FROM search_log`).Scan(&m.AvgSearchLatencyMs)

	// Stale memories (no access in 30+ days)
	cutoff := time.Now().AddDate(0, 0, -30).UTC().Format(time.RFC3339)
	s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE deleted_at IS NULL AND (last_accessed IS NULL OR last_accessed < ?)`, cutoff).Scan(&m.StaleMemoryCount)

	// Top projects
	rows, err := s.db.Query(`SELECT COALESCE(project, '(none)'), COUNT(*) as c FROM observations WHERE deleted_at IS NULL GROUP BY project ORDER BY c DESC LIMIT 10`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ps ProjectStat
			if rows.Scan(&ps.Project, &ps.Count) == nil {
				m.TopProjects = append(m.TopProjects, ps)
			}
		}
	}

	return m, nil
}

// --- Helpers ---

func (s *Store) queryObservations(query string, args ...interface{}) ([]Observation, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Observation
	for rows.Next() {
		obs, err := s.scanObservationRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *obs)
	}
	return results, rows.Err()
}

func (s *Store) scanObservation(row *sql.Row) (*Observation, error) {
	var obs Observation
	var sessionID sql.NullString
	err := row.Scan(
		&obs.ID, &obs.SyncID, &sessionID, &obs.Type, &obs.Title, &obs.Content,
		&obs.ToolName, &obs.Project, &obs.Scope, &obs.TopicKey, &obs.NormalizedHash,
		&obs.RevisionCount, &obs.DuplicateCount, &obs.Importance, &obs.AccessCount,
		&obs.LastAccessed, &obs.LastSeenAt, &obs.CreatedAt, &obs.UpdatedAt, &obs.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	obs.SessionID = sessionID.String
	return &obs, nil
}

func (s *Store) scanObservationRows(rows *sql.Rows) (*Observation, error) {
	var obs Observation
	var sessionID sql.NullString
	err := rows.Scan(
		&obs.ID, &obs.SyncID, &sessionID, &obs.Type, &obs.Title, &obs.Content,
		&obs.ToolName, &obs.Project, &obs.Scope, &obs.TopicKey, &obs.NormalizedHash,
		&obs.RevisionCount, &obs.DuplicateCount, &obs.Importance, &obs.AccessCount,
		&obs.LastAccessed, &obs.LastSeenAt, &obs.CreatedAt, &obs.UpdatedAt, &obs.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	obs.SessionID = sessionID.String
	return &obs, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// --- Export/Import for Sync ---

type ExportData struct {
	Sessions     []Session     `json:"sessions"`
	Observations []Observation `json:"observations"`
	Prompts      []Prompt      `json:"prompts"`
}

func (s *Store) ExportAll(project string) (*ExportData, error) {
	data := &ExportData{}

	// Sessions
	sessQ := `SELECT id, project, directory, started_at, ended_at, summary FROM sessions`
	sessArgs := []interface{}{}
	if project != "" {
		sessQ += " WHERE project = ?"
		sessArgs = append(sessArgs, project)
	}
	rows, err := s.db.Query(sessQ, sessArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Directory, &sess.StartedAt, &sess.EndedAt, &sess.Summary); err != nil {
			return nil, err
		}
		data.Sessions = append(data.Sessions, sess)
	}

	// Observations
	obsQ := `SELECT id, sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at FROM observations`
	obsArgs := []interface{}{}
	if project != "" {
		obsQ += " WHERE project = ?"
		obsArgs = append(obsArgs, project)
	}
	obs, err := s.queryObservations(obsQ, obsArgs...)
	if err != nil {
		return nil, err
	}
	data.Observations = obs

	// Prompts
	promptQ := `SELECT id, sync_id, session_id, content, project, created_at FROM user_prompts`
	promptArgs := []interface{}{}
	if project != "" {
		promptQ += " WHERE project = ?"
		promptArgs = append(promptArgs, project)
	}
	pRows, err := s.db.Query(promptQ, promptArgs...)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()
	for pRows.Next() {
		var p Prompt
		if err := pRows.Scan(&p.ID, &p.SyncID, &p.SessionID, &p.Content, &p.Project, &p.CreatedAt); err != nil {
			return nil, err
		}
		data.Prompts = append(data.Prompts, p)
	}

	return data, nil
}

func (s *Store) ImportData(data *ExportData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, sess := range data.Sessions {
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, ended_at, summary) VALUES (?, ?, ?, ?, ?, ?)`,
			sess.ID, sess.Project, sess.Directory, sess.StartedAt, sess.EndedAt, sess.Summary,
		)
		if err != nil {
			return fmt.Errorf("import session: %w", err)
		}
	}

	for _, obs := range data.Observations {
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO observations (sync_id, session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, importance, access_count, last_accessed, last_seen_at, created_at, updated_at, deleted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			obs.SyncID, nilIfEmpty(obs.SessionID), obs.Type, obs.Title, obs.Content,
			obs.ToolName, obs.Project, obs.Scope, obs.TopicKey, obs.NormalizedHash,
			obs.RevisionCount, obs.DuplicateCount, obs.Importance, obs.AccessCount,
			obs.LastAccessed, obs.LastSeenAt, obs.CreatedAt, obs.UpdatedAt, obs.DeletedAt,
		)
		if err != nil {
			return fmt.Errorf("import observation: %w", err)
		}
	}

	for _, p := range data.Prompts {
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO user_prompts (sync_id, session_id, content, project, created_at) VALUES (?, ?, ?, ?, ?)`,
			p.SyncID, p.SessionID, p.Content, p.Project, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("import prompt: %w", err)
		}
	}

	return tx.Commit()
}
