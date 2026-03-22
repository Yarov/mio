package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mio/internal/store"
)

type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenSearch
	ScreenSearchResults
	ScreenRecent
	ScreenObservationDetail
	ScreenTimeline
	ScreenSessions
	ScreenSessionDetail
)

// Model holds all TUI state.
type Model struct {
	store *store.Store

	screen     Screen
	prevScreen Screen
	width      int
	height     int

	// Dashboard
	metrics *store.Metrics

	// Search
	searchInput   textinput.Model
	searchResults []store.SearchResult

	// Lists
	cursor       int
	scrollOffset int

	// Recent
	recentObs []store.Observation

	// Timeline
	timeline []store.TimelineEntry

	// Sessions
	sessions        []store.SessionSummary
	selectedSession *store.SessionSummary

	// Detail
	selectedObs *store.Observation
	relations   []store.Relation

	// Status
	err error
}

// New creates a new TUI model.
func New(s *store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "Search memories..."
	ti.CharLimit = 200

	return Model{
		store:       s,
		screen:      ScreenDashboard,
		searchInput: ti,
		width:       80,
		height:      24,
	}
}

func (m Model) Init() tea.Cmd {
	return fetchMetrics(m.store)
}

// --- Async messages ---

type metricsMsg struct {
	metrics *store.Metrics
	err     error
}

type searchResultsMsg struct {
	results []store.SearchResult
	err     error
}

type recentMsg struct {
	obs []store.Observation
	err error
}

type timelineMsg struct {
	entries []store.TimelineEntry
	err     error
}

type sessionsMsg struct {
	sessions []store.SessionSummary
	err      error
}

type observationMsg struct {
	obs       *store.Observation
	relations []store.Relation
	err       error
}

// --- Async commands ---

func fetchMetrics(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		m, err := s.GetMetrics()
		return metricsMsg{metrics: m, err: err}
	}
}

func fetchSearch(s *store.Store, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := s.Search(query, "", "", 50)
		return searchResultsMsg{results: results, err: err}
	}
}

func fetchRecent(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		obs, err := s.RecentContext("", 50)
		return recentMsg{obs: obs, err: err}
	}
}

func fetchTimeline(s *store.Store, id int64) tea.Cmd {
	return func() tea.Msg {
		entries, err := s.Timeline(id, 10, 10)
		return timelineMsg{entries: entries, err: err}
	}
}

func fetchSessions(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		sessions, err := s.RecentSessions("", 50)
		return sessionsMsg{sessions: sessions, err: err}
	}
}

func fetchObservation(s *store.Store, id int64) tea.Cmd {
	return func() tea.Msg {
		obs, err := s.GetObservation(id)
		if err != nil {
			return observationMsg{err: err}
		}
		rels, _ := s.GetRelations(id)
		return observationMsg{obs: obs, relations: rels}
	}
}
