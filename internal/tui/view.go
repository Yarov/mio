package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mio/internal/store"
)

func (m Model) View() string {
	var content string
	switch m.screen {
	case ScreenDashboard:
		content = m.viewDashboard()
	case ScreenSearch:
		content = m.viewSearch()
	case ScreenSearchResults:
		content = m.viewSearchResults()
	case ScreenRecent:
		content = m.viewRecent()
	case ScreenObservationDetail:
		content = m.viewObservationDetail()
	case ScreenTimeline:
		content = m.viewTimeline()
	case ScreenSessions:
		content = m.viewSessions()
	case ScreenSessionDetail:
		content = m.viewSessionDetail()
	}

	if m.err != nil {
		content += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return content
}

// --- Dashboard ---

func (m Model) viewDashboard() string {
	var b strings.Builder

	// Logo
	logo := logoStyle.Render(`
  ███╗   ███╗██╗ ██████╗
  ████╗ ████║██║██╔═══██╗
  ██╔████╔██║██║██║   ██║
  ██║╚██╔╝██║██║██║   ██║
  ██║ ╚═╝ ██║██║╚██████╔╝
  ╚═╝     ╚═╝╚═╝ ╚═════╝`)
	b.WriteString(logo + "\n")
	b.WriteString(helpStyle.Render("  Persistent memory for AI agents") + "\n\n")

	// Stats
	if m.metrics != nil {
		stats := []string{
			renderStat("Memories", fmt.Sprintf("%d", m.metrics.TotalObservations)),
			renderStat("Sessions", fmt.Sprintf("%d", m.metrics.TotalSessions)),
			renderStat("Searches", fmt.Sprintf("%d", m.metrics.TotalSearches)),
			renderStat("Hit Rate", fmt.Sprintf("%.0f%%", m.metrics.SearchHitRate)),
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, stats...)
		b.WriteString("  " + row + "\n\n")

		// Top projects
		if len(m.metrics.TopProjects) > 0 {
			b.WriteString(sectionHeadingStyle.Render("  Top Projects") + "\n")
			for i, p := range m.metrics.TopProjects {
				if i >= 5 {
					break
				}
				b.WriteString(fmt.Sprintf("    %s %s\n",
					projectStyle.Render(p.Project),
					statLabelStyle.Render(fmt.Sprintf("(%d)", p.Count)),
				))
			}
			b.WriteString("\n")
		}

		if m.metrics.StaleMemoryCount > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %d stale memories (30+ days without access)", m.metrics.StaleMemoryCount)) + "\n\n")
		}
	}

	// Menu
	b.WriteString(sectionHeadingStyle.Render("  Navigation") + "\n")
	menuItems := []struct {
		key  string
		name string
	}{
		{"1", "Search memories"},
		{"2", "Recent memories"},
		{"3", "Sessions"},
	}
	for _, item := range menuItems {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			typeBadgeStyle("decision").Render("["+item.key+"]"),
			menuItemStyle.Render(item.name),
		))
	}

	b.WriteString("\n" + statusBarStyle.Render("  [1-3] navigate  [q] quit"))

	return b.String()
}

func renderStat(label, value string) string {
	return statCardStyle.Render(
		statNumberStyle.Render(value) + "\n" +
			statLabelStyle.Render(label),
	)
}

// --- Search ---

func (m Model) viewSearch() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Search Memories") + "\n\n")
	b.WriteString("  " + m.searchInput.View() + "\n\n")
	b.WriteString(statusBarStyle.Render("  [enter] search  [esc] back"))

	return b.String()
}

// --- Search Results ---

func (m Model) viewSearchResults() string {
	var b strings.Builder

	query := m.searchInput.Value()
	b.WriteString(titleStyle.Render(fmt.Sprintf("  Results for: %q", query)))
	b.WriteString(helpStyle.Render(fmt.Sprintf("  (%d found)", len(m.searchResults))) + "\n\n")

	if len(m.searchResults) == 0 {
		b.WriteString(helpStyle.Render("  No results found.") + "\n")
	} else {
		visible := m.visibleListItems()
		end := m.scrollOffset + visible
		if end > len(m.searchResults) {
			end = len(m.searchResults)
		}
		for i := m.scrollOffset; i < end; i++ {
			r := m.searchResults[i]
			selected := i == m.cursor
			b.WriteString(renderSearchItem(r, selected, m.width) + "\n")
		}

		if len(m.searchResults) > visible {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  showing %d-%d of %d", m.scrollOffset+1, end, len(m.searchResults))) + "\n")
		}
	}

	b.WriteString("\n" + statusBarStyle.Render("  [j/k] navigate  [enter] details  [esc] back"))

	return b.String()
}

func renderSearchItem(r store.SearchResult, selected bool, maxWidth int) string {
	badge := typeBadgeStyle(r.Type).Render("[" + r.Type + "]")
	id := idStyle.Render(fmt.Sprintf("#%d", r.ID))
	score := scoreStyle.Render(fmt.Sprintf("%.2f", r.Score))
	title := r.Title

	line1 := fmt.Sprintf("  %s %s %s  %s", id, badge, title, score)
	preview := truncateStr(r.Content, 80)
	line2 := "    " + contentPreviewStyle.Render(preview)

	if selected {
		line1 = listSelectedStyle.Render("> ") + line1[2:]
	}

	return line1 + "\n" + line2
}

// --- Recent ---

func (m Model) viewRecent() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("  Recent Memories (%d)", len(m.recentObs))) + "\n\n")

	if len(m.recentObs) == 0 {
		b.WriteString(helpStyle.Render("  No memories yet.") + "\n")
	} else {
		visible := m.visibleListItems()
		end := m.scrollOffset + visible
		if end > len(m.recentObs) {
			end = len(m.recentObs)
		}
		for i := m.scrollOffset; i < end; i++ {
			o := m.recentObs[i]
			selected := i == m.cursor
			b.WriteString(renderObservationItem(o, selected) + "\n")
		}

		if len(m.recentObs) > visible {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  showing %d-%d of %d", m.scrollOffset+1, end, len(m.recentObs))) + "\n")
		}
	}

	b.WriteString("\n" + statusBarStyle.Render("  [j/k] navigate  [enter] details  [esc] back"))

	return b.String()
}

func renderObservationItem(o store.Observation, selected bool) string {
	badge := typeBadgeStyle(o.Type).Render("[" + o.Type + "]")
	id := idStyle.Render(fmt.Sprintf("#%d", o.ID))
	ts := timestampStyle.Render(formatTime(o.CreatedAt))

	proj := ""
	if o.Project != nil {
		proj = " " + projectStyle.Render(*o.Project)
	}

	line1 := fmt.Sprintf("  %s %s %s  %s%s", id, badge, o.Title, ts, proj)
	preview := truncateStr(o.Content, 80)
	line2 := "    " + contentPreviewStyle.Render(preview)

	if selected {
		line1 = listSelectedStyle.Render("> ") + line1[2:]
	}

	return line1 + "\n" + line2
}

// --- Observation Detail ---

func (m Model) viewObservationDetail() string {
	if m.selectedObs == nil {
		return helpStyle.Render("  No observation selected")
	}

	var b strings.Builder
	obs := m.selectedObs

	// Header
	badge := typeBadgeStyle(obs.Type).Render("[" + obs.Type + "]")
	id := idStyle.Render(fmt.Sprintf("#%d", obs.ID))
	b.WriteString(titleStyle.Render(fmt.Sprintf("  %s %s %s", id, badge, obs.Title)) + "\n\n")

	// Metadata
	fields := []struct {
		label string
		value string
	}{
		{"Type", obs.Type},
		{"Scope", obs.Scope},
		{"Importance", fmt.Sprintf("%.1f", obs.Importance)},
		{"Accessed", fmt.Sprintf("%d times", obs.AccessCount)},
		{"Created", formatTime(obs.CreatedAt)},
		{"Updated", formatTime(obs.UpdatedAt)},
	}
	if obs.Project != nil {
		fields = append([]struct {
			label string
			value string
		}{{"Project", *obs.Project}}, fields...)
	}
	if obs.TopicKey != nil {
		fields = append(fields, struct {
			label string
			value string
		}{"Topic Key", *obs.TopicKey})
	}

	for _, f := range fields {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			detailLabelStyle.Render(f.label+":"),
			detailValueStyle.Render(f.value),
		))
	}

	// Content
	b.WriteString("\n" + sectionHeadingStyle.Render("  Content") + "\n")
	contentLines := strings.Split(obs.Content, "\n")
	maxContentWidth := m.width - 8
	if maxContentWidth < 40 {
		maxContentWidth = 40
	}

	// Apply scroll
	startLine := m.scrollOffset
	if startLine >= len(contentLines) {
		startLine = len(contentLines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}
	maxLines := m.height - 20
	if maxLines < 5 {
		maxLines = 5
	}
	endLine := startLine + maxLines
	if endLine > len(contentLines) {
		endLine = len(contentLines)
	}

	for i := startLine; i < endLine; i++ {
		line := contentLines[i]
		if len(line) > maxContentWidth {
			line = line[:maxContentWidth]
		}
		b.WriteString("    " + detailValueStyle.Render(line) + "\n")
	}

	if len(contentLines) > maxLines {
		b.WriteString(helpStyle.Render(fmt.Sprintf("    (%d/%d lines)", endLine, len(contentLines))) + "\n")
	}

	// Relations
	if len(m.relations) > 0 {
		b.WriteString("\n" + sectionHeadingStyle.Render(fmt.Sprintf("  Relations (%d)", len(m.relations))) + "\n")
		for _, r := range m.relations {
			other := r.ToID
			if r.ToID == obs.ID {
				other = r.FromID
			}
			b.WriteString(fmt.Sprintf("    %s #%d (%s, strength: %.1f)\n",
				typeBadgeStyle("decision").Render("["+r.Type+"]"),
				other,
				helpStyle.Render(r.Type),
				r.Strength,
			))
		}
	}

	b.WriteString("\n" + statusBarStyle.Render("  [j/k] scroll  [t] timeline  [esc] back"))

	return b.String()
}

// --- Timeline ---

func (m Model) viewTimeline() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Timeline") + "\n\n")

	if len(m.timeline) == 0 {
		b.WriteString(helpStyle.Render("  No timeline entries.") + "\n")
	} else {
		visible := m.visibleListItems()
		end := m.scrollOffset + visible
		if end > len(m.timeline) {
			end = len(m.timeline)
		}
		for i := m.scrollOffset; i < end; i++ {
			e := m.timeline[i]
			selected := i == m.cursor

			connector := timelineConnectorStyle.Render("  │ ")
			if i == 0 {
				connector = timelineConnectorStyle.Render("  ┌ ")
			}
			if i == len(m.timeline)-1 {
				connector = timelineConnectorStyle.Render("  └ ")
			}

			badge := typeBadgeStyle(e.Type).Render("[" + e.Type + "]")
			id := idStyle.Render(fmt.Sprintf("#%d", e.ID))
			ts := timestampStyle.Render(formatTime(e.CreatedAt))

			marker := " "
			style := timelineItemStyle
			if e.IsFocus {
				marker = ">"
				style = timelineFocusStyle
			}
			if selected {
				marker = ">"
				style = listSelectedStyle
			}

			line1 := fmt.Sprintf("%s%s %s %s %s  %s",
				connector, style.Render(marker), id, badge, e.Title, ts)

			preview := truncateStr(e.Content, 60)
			line2 := timelineConnectorStyle.Render("  │   ") + contentPreviewStyle.Render(preview)

			b.WriteString(line1 + "\n" + line2 + "\n")
		}
	}

	b.WriteString("\n" + statusBarStyle.Render("  [j/k] navigate  [enter] details  [esc] back"))

	return b.String()
}

// --- Sessions ---

func (m Model) viewSessions() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("  Sessions (%d)", len(m.sessions))) + "\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(helpStyle.Render("  No sessions yet.") + "\n")
	} else {
		visible := m.visibleListItems()
		end := m.scrollOffset + visible
		if end > len(m.sessions) {
			end = len(m.sessions)
		}
		for i := m.scrollOffset; i < end; i++ {
			s := m.sessions[i]
			selected := i == m.cursor

			status := projectStyle.Render("active")
			if s.EndedAt != nil {
				status = helpStyle.Render("ended")
			}

			shortID := s.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			line1 := fmt.Sprintf("  %s %s  %s  obs: %s  %s",
				idStyle.Render(shortID),
				projectStyle.Render(s.Project),
				timestampStyle.Render(formatTime(s.StartedAt)),
				statNumberStyle.Render(fmt.Sprintf("%d", s.ObservationCount)),
				status,
			)

			line2 := "    "
			if s.Summary != nil {
				line2 += contentPreviewStyle.Render(truncateStr(*s.Summary, 80))
			} else {
				line2 += helpStyle.Render("(no summary)")
			}

			if selected {
				line1 = listSelectedStyle.Render("> ") + line1[2:]
			}

			b.WriteString(line1 + "\n" + line2 + "\n")
		}
	}

	b.WriteString("\n" + statusBarStyle.Render("  [j/k] navigate  [enter] details  [esc] back"))

	return b.String()
}

// --- Session Detail ---

func (m Model) viewSessionDetail() string {
	if m.selectedSession == nil {
		return helpStyle.Render("  No session selected")
	}

	var b strings.Builder
	s := m.selectedSession

	b.WriteString(titleStyle.Render("  Session Detail") + "\n\n")

	fields := []struct {
		label string
		value string
	}{
		{"ID", s.ID},
		{"Project", s.Project},
		{"Directory", s.Directory},
		{"Started", formatTime(s.StartedAt)},
		{"Observations", fmt.Sprintf("%d", s.ObservationCount)},
	}

	if s.EndedAt != nil {
		fields = append(fields, struct {
			label string
			value string
		}{"Ended", formatTime(*s.EndedAt)})
	} else {
		fields = append(fields, struct {
			label string
			value string
		}{"Status", "active"})
	}

	for _, f := range fields {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			detailLabelStyle.Render(f.label+":"),
			detailValueStyle.Render(f.value),
		))
	}

	if s.Summary != nil {
		b.WriteString("\n" + sectionHeadingStyle.Render("  Summary") + "\n")
		b.WriteString("    " + detailValueStyle.Render(*s.Summary) + "\n")
	}

	b.WriteString("\n" + statusBarStyle.Render("  [esc] back"))

	return b.String()
}

// --- Helpers ---

func truncateStr(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func formatTime(t string) string {
	if len(t) > 16 {
		return t[:16] // YYYY-MM-DDTHH:MM
	}
	return t
}
