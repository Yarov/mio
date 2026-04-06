package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"mio/internal/agents"
)

//go:embed templates/*.html
var templateFS embed.FS

var dashboardTmpl *template.Template

func init() {
	funcs := template.FuncMap{
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}
	dashboardTmpl = template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
}

// SkillInfo represents an installed skill
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version"`
}

// skillInstallDirs returns candidate skill roots in priority order (Cursor first, then Claude).
func skillInstallDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".cursor", "skills"),
		filepath.Join(home, ".claude", "skills"),
	}
}

// findSkillMarkdown returns the path to {name}/SKILL.md under Cursor or Claude skills.
func findSkillMarkdown(name string) string {
	for _, root := range skillInstallDirs() {
		p := filepath.Join(root, name, "SKILL.md")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// scanSkills reads installed skills from ~/.cursor/skills/ and ~/.claude/skills/.
// When the same skill exists in both, the Cursor copy wins.
func scanSkills() []SkillInfo {
	var skills []SkillInfo
	seen := make(map[string]struct{})

	for _, skillsDir := range skillInstallDirs() {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "_shared" {
				continue
			}
			if _, dup := seen[entry.Name()]; dup {
				continue
			}

			skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			skill := parseSkillFrontmatter(entry.Name(), string(data))
			seen[entry.Name()] = struct{}{}
			skills = append(skills, skill)
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Category != skills[j].Category {
			return skills[i].Category < skills[j].Category
		}
		return skills[i].Name < skills[j].Name
	})

	return skills
}

// parseSkillFrontmatter extracts name, description, and version from YAML frontmatter
func parseSkillFrontmatter(dirName, content string) SkillInfo {
	skill := SkillInfo{Name: dirName, Category: "other"}

	// Check for frontmatter
	if !strings.HasPrefix(content, "---") {
		return skill
	}

	end := strings.Index(content[3:], "---")
	if end < 0 {
		return skill
	}

	frontmatter := content[3 : end+3]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			skill.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			desc = strings.TrimPrefix(desc, ">")
			skill.Description = strings.TrimSpace(desc)
		} else if strings.HasPrefix(line, "version:") {
			skill.Version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "version:")), "\"")
		}
	}

	// Multi-line description (indented continuation)
	if skill.Description == "" || skill.Description == ">" {
		lines := strings.Split(frontmatter, "\n")
		var descLines []string
		inDesc := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "description:") {
				inDesc = true
				after := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
				after = strings.TrimPrefix(after, ">")
				after = strings.TrimSpace(after)
				if after != "" && after != ">" {
					descLines = append(descLines, after)
				}
				continue
			}
			if inDesc && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
				descLines = append(descLines, strings.TrimSpace(line))
			} else if inDesc {
				break
			}
		}
		if len(descLines) > 0 {
			skill.Description = strings.Join(descLines, " ")
		}
	}

	// Categorize
	switch {
	case strings.HasPrefix(skill.Name, "sdd-"):
		skill.Category = "sdd"
	case strings.HasPrefix(skill.Name, "mio-"):
		skill.Category = "orchestrator"
	case strings.HasPrefix(skill.Name, "ads"):
		skill.Category = "ads"
	case strings.Contains(skill.Name, "pr") || strings.Contains(skill.Name, "review"):
		skill.Category = "engineering"
	case skill.Name == "skill-creator" || skill.Name == "skill-registry" || skill.Name == "find-skills":
		skill.Category = "meta"
	default:
		skill.Category = "coding"
	}

	return skill
}

func (s *HTTPServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.store.GetMetrics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	recent, err := s.store.RecentContext("", 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sessions, err := s.store.RecentSessions("", 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	skills := scanSkills()

	// Group skills by category
	skillGroups := make(map[string][]SkillInfo)
	for _, sk := range skills {
		skillGroups[sk.Category] = append(skillGroups[sk.Category], sk)
	}

	agentStatuses := agents.StatusAll()

	data := map[string]interface{}{
		"Metrics":      metrics,
		"Observations": recent,
		"Sessions":     sessions,
		"Skills":       skills,
		"SkillGroups":  skillGroups,
		"SkillCount":   len(skills),
		"Agents":       agentStatuses,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *HTTPServer) handleSkills(w http.ResponseWriter, _ *http.Request) {
	skills := scanSkills()
	writeJSON(w, http.StatusOK, skills)
}

func (s *HTTPServer) handleSkillGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := findSkillMarkdown(name)

	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "skill not found: "+name)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"name":    name,
		"content": string(data),
		"path":    path,
	})
}

func (s *HTTPServer) handleSkillUpdate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := findSkillMarkdown(name)

	// Verify skill exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "skill not found: "+name)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "name": name})
}

func (s *HTTPServer) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	agentName := r.URL.Query().Get("agent")
	if agentName == "" {
		agentName = agents.DefaultSetupAgent()
	}

	exe, err := os.Executable()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot find binary: "+err.Error())
		return
	}

	cmd := exec.Command(exe, "setup", agentName)
	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"output": string(output),
		"agent":  agentName,
	}
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(result)
		return
	}

	result["status"] = "ok"
	result["message"] = fmt.Sprintf("Setup completed for %s. Restart the agent to activate.", agentName)
	writeJSON(w, http.StatusOK, result)
}

func (s *HTTPServer) handleAdminUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	agentName := r.URL.Query().Get("agent")
	if agentName == "" {
		writeError(w, http.StatusBadRequest, "agent parameter required")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot find binary: "+err.Error())
		return
	}

	cmd := exec.Command(exe, "uninstall", agentName)
	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"output": string(output),
		"agent":  agentName,
	}
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(result)
		return
	}

	result["status"] = "ok"
	result["message"] = fmt.Sprintf("Uninstalled Mio from %s.", agentName)
	writeJSON(w, http.StatusOK, result)
}

func (s *HTTPServer) handleAgents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, agents.StatusAll())
}
