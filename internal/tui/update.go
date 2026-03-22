package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case metricsMsg:
		m.err = msg.err
		if msg.err == nil {
			m.metrics = msg.metrics
		}
		return m, nil

	case searchResultsMsg:
		m.err = msg.err
		if msg.err == nil {
			m.searchResults = msg.results
			m.screen = ScreenSearchResults
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case recentMsg:
		m.err = msg.err
		if msg.err == nil {
			m.recentObs = msg.obs
			m.screen = ScreenRecent
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case timelineMsg:
		m.err = msg.err
		if msg.err == nil {
			m.timeline = msg.entries
			m.prevScreen = m.screen
			m.screen = ScreenTimeline
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case sessionsMsg:
		m.err = msg.err
		if msg.err == nil {
			m.sessions = msg.sessions
			m.screen = ScreenSessions
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case observationMsg:
		m.err = msg.err
		if msg.err == nil {
			m.selectedObs = msg.obs
			m.relations = msg.relations
			m.prevScreen = m.screen
			m.screen = ScreenObservationDetail
			m.scrollOffset = 0
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKeys(msg)
	}

	return m, nil
}

func (m Model) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenDashboard:
		return m.handleDashboardKeys(msg)
	case ScreenSearch:
		return m.handleSearchKeys(msg)
	case ScreenSearchResults:
		return m.handleSearchResultsKeys(msg)
	case ScreenRecent:
		return m.handleRecentKeys(msg)
	case ScreenObservationDetail:
		return m.handleDetailKeys(msg)
	case ScreenTimeline:
		return m.handleTimelineKeys(msg)
	case ScreenSessions:
		return m.handleSessionsKeys(msg)
	case ScreenSessionDetail:
		return m.handleSessionDetailKeys(msg)
	}
	return m, nil
}

// --- Dashboard ---

func (m Model) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "1", "s", "/":
		m.screen = ScreenSearch
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		return m, nil
	case "2", "r":
		return m, fetchRecent(m.store)
	case "3", "e":
		return m, fetchSessions(m.store)
	}
	return m, nil
}

// --- Search ---

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.searchInput.Blur()
		return m, nil
	case "enter":
		query := m.searchInput.Value()
		if query == "" {
			return m, nil
		}
		m.searchInput.Blur()
		return m, fetchSearch(m.store, query)
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
}

// --- Search Results ---

func (m Model) handleSearchResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = ScreenSearch
		m.searchInput.Focus()
		return m, nil
	case "j", "down":
		if m.cursor < len(m.searchResults)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil
	case "enter":
		if len(m.searchResults) > 0 {
			id := m.searchResults[m.cursor].ID
			return m, fetchObservation(m.store, id)
		}
	}
	return m, nil
}

// --- Recent ---

func (m Model) handleRecentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = ScreenDashboard
		return m, nil
	case "j", "down":
		if m.cursor < len(m.recentObs)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil
	case "enter":
		if len(m.recentObs) > 0 {
			id := m.recentObs[m.cursor].ID
			return m, fetchObservation(m.store, id)
		}
	}
	return m, nil
}

// --- Observation Detail ---

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = m.prevScreen
		return m, nil
	case "j", "down":
		m.scrollOffset++
		return m, nil
	case "k", "up":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	case "t":
		if m.selectedObs != nil {
			return m, fetchTimeline(m.store, m.selectedObs.ID)
		}
	}
	return m, nil
}

// --- Timeline ---

func (m Model) handleTimelineKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = m.prevScreen
		return m, nil
	case "j", "down":
		if m.cursor < len(m.timeline)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil
	case "enter":
		if len(m.timeline) > 0 {
			id := m.timeline[m.cursor].ID
			return m, fetchObservation(m.store, id)
		}
	}
	return m, nil
}

// --- Sessions ---

func (m Model) handleSessionsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = ScreenDashboard
		return m, nil
	case "j", "down":
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil
	case "enter":
		if len(m.sessions) > 0 {
			sess := m.sessions[m.cursor]
			m.selectedSession = &sess
			m.prevScreen = m.screen
			m.screen = ScreenSessionDetail
			m.scrollOffset = 0
		}
		return m, nil
	}
	return m, nil
}

// --- Session Detail ---

func (m Model) handleSessionDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = ScreenSessions
		return m, nil
	}
	return m, nil
}

// --- Helpers ---

func (m *Model) adjustScroll() {
	visibleItems := m.visibleListItems()
	if visibleItems <= 0 {
		visibleItems = 10
	}
	if m.cursor >= m.scrollOffset+visibleItems {
		m.scrollOffset = m.cursor - visibleItems + 1
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
}

func (m Model) visibleListItems() int {
	// Reserve lines for header and footer
	available := m.height - 6
	return available / 2 // 2 lines per item
}
