package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"mio/internal/config"
	"mio/internal/mcp"
	"mio/internal/server"
	"mio/internal/store"
	msync "mio/internal/sync"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.Default()

	// Override data dir from env
	if dir := os.Getenv("MIO_DATA_DIR"); dir != "" {
		cfg.DataDir = dir
		cfg.DBPath = dir + "/mio.db"
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "mcp":
		runMCP(cfg)
	case "serve":
		runServe(cfg, args)
	case "save":
		runSave(cfg, args)
	case "search":
		runSearch(cfg, args)
	case "context":
		runContext(cfg, args)
	case "timeline":
		runTimeline(cfg, args)
	case "stats":
		runStats(cfg)
	case "export":
		runExport(cfg, args)
	case "import":
		runImport(cfg, args)
	case "sync":
		runSync(cfg, args)
	case "version":
		fmt.Printf("mio %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`mio - Persistent memory for AI agents

Usage: mio <command> [args]

Commands:
  mcp                  Start MCP stdio server (for agent integration)
  serve [port]         Start HTTP API server (default: 7438)
  save <title> <content> [--type TYPE] [--project PROJECT]
                       Save a memory directly
  search <query> [--project PROJECT] [--type TYPE] [--limit N]
                       Search memories
  context [--project PROJECT] [--limit N]
                       Get recent context
  timeline <id> [--before N] [--after N]
                       Show timeline around an observation
  stats                Show memory statistics
  export [--project PROJECT] [file]
                       Export memories to JSON
  import <file>        Import memories from JSON
  sync                 Export new memories as sync chunk
  sync --import        Import pending sync chunks
  sync --status        Show sync status
  version              Show version
  help                 Show this help

Environment:
  MIO_DATA_DIR         Data directory (default: ~/.mio)`)
}

func openStore(cfg *config.Config) *store.Store {
	s, err := store.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return s
}

func runMCP(cfg *config.Config) {
	s := openStore(cfg)
	defer s.Close()

	srv := mcp.New(s, cfg)
	if err := srv.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "mcp error: %v\n", err)
		os.Exit(1)
	}
}

func runServe(cfg *config.Config, args []string) {
	if len(args) > 0 {
		port, err := strconv.Atoi(args[0])
		if err == nil {
			cfg.HTTPPort = port
		}
	}

	s := openStore(cfg)
	defer s.Close()

	srv := server.New(s, cfg)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func runSave(cfg *config.Config, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mio save <title> <content> [--type TYPE] [--project PROJECT]")
		os.Exit(1)
	}

	title := args[0]
	content := args[1]
	obsType := "discovery"
	project := ""

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				obsType = args[i+1]
				i++
			}
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		}
	}

	s := openStore(cfg)
	defer s.Close()

	obs := &store.Observation{
		Title:      title,
		Type:       obsType,
		Content:    content,
		Importance: 0.5,
		Scope:      "project",
	}
	if project != "" {
		obs.Project = &project
	}

	id, err := s.Save(obs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saved observation #%d\n", id)
}

func runSearch(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: mio search <query> [--project PROJECT] [--type TYPE] [--limit N]")
		os.Exit(1)
	}

	// Collect query words (everything before flags)
	var queryParts []string
	project := ""
	obsType := ""
	limit := 0

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "--type":
			if i+1 < len(args) {
				obsType = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		default:
			queryParts = append(queryParts, args[i])
		}
	}

	query := strings.Join(queryParts, " ")
	if query == "" {
		fmt.Fprintln(os.Stderr, "error: empty query")
		os.Exit(1)
	}

	s := openStore(cfg)
	defer s.Close()

	results, err := s.Search(query, project, obsType, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	for _, r := range results {
		preview := r.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("#%d [%s] %s (score: %.2f)\n", r.ID, r.Type, r.Title, r.Score)
		fmt.Printf("  %s\n", preview)
		fmt.Printf("  created: %s | accessed: %d times\n\n", r.CreatedAt, r.AccessCount)
	}
}

func runContext(cfg *config.Config, args []string) {
	project := ""
	limit := 0

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}

	s := openStore(cfg)
	defer s.Close()

	obs, err := s.RecentContext(project, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(obs) == 0 {
		fmt.Println("No recent context.")
		return
	}

	for _, o := range obs {
		preview := o.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("#%d [%s] %s\n  %s\n  %s\n\n", o.ID, o.Type, o.Title, preview, o.CreatedAt)
	}
}

func runTimeline(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: mio timeline <id> [--before N] [--after N]")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid id: %s\n", args[0])
		os.Exit(1)
	}

	before := 5
	after := 5
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--before":
			if i+1 < len(args) {
				before, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--after":
			if i+1 < len(args) {
				after, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}

	s := openStore(cfg)
	defer s.Close()

	entries, err := s.Timeline(id, before, after)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, e := range entries {
		marker := "  "
		if e.IsFocus {
			marker = "> "
		}
		preview := e.Content
		if len(preview) > 150 {
			preview = preview[:150] + "..."
		}
		fmt.Printf("%s#%d [%s] %s\n    %s\n    %s\n\n", marker, e.ID, e.Type, e.Title, preview, e.CreatedAt)
	}
}

func runStats(cfg *config.Config) {
	s := openStore(cfg)
	defer s.Close()

	metrics, err := s.GetMetrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(metrics, "", "  ")
	fmt.Println(string(data))
}

func runExport(cfg *config.Config, args []string) {
	project := ""
	outputFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		default:
			if outputFile == "" {
				outputFile = args[i]
			}
		}
	}

	s := openStore(cfg)
	defer s.Close()

	data, err := s.ExportAll(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")

	if outputFile != "" {
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported to %s\n", outputFile)
	} else {
		fmt.Println(string(jsonData))
	}
}

func runImport(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: mio import <file>")
		os.Exit(1)
	}

	fileData, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	var data store.ExportData
	if err := json.Unmarshal(fileData, &data); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	s := openStore(cfg)
	defer s.Close()

	if err := s.ImportData(&data); err != nil {
		fmt.Fprintf(os.Stderr, "error importing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Imported %d sessions, %d observations, %d prompts\n",
		len(data.Sessions), len(data.Observations), len(data.Prompts))
}

func runSync(cfg *config.Config, args []string) {
	s := openStore(cfg)
	defer s.Close()

	syncer, err := msync.NewSyncer(s, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(args) > 0 {
		switch args[0] {
		case "--import":
			count, err := syncer.Import()
			if err != nil {
				fmt.Fprintf(os.Stderr, "import error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Imported %d chunks\n", count)
			return
		case "--status":
			status := syncer.Status()
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
			return
		}
	}

	// Default: export
	project := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			project = args[i+1]
			i++
		}
	}

	meta, err := syncer.Export(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "export error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exported chunk %s (%d sessions, %d memories, %d prompts)\n",
		meta.ID, meta.SessionCount, meta.MemoryCount, meta.PromptCount)
}
