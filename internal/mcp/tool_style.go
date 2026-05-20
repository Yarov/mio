package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"mio/internal/agents"
	"mio/internal/store"
)

func (s *Server) handleMemStyleGet(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, err := s.store.GetOutputStyle()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status := "enabled"
	if !cfg.Enabled {
		status = "disabled"
	}

	return mcp.NewToolResultText(fmt.Sprintf("**Output Style:** %s", status)), nil
}

func (s *Server) handleMemStyleSet(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	enabled := boolArg(request, "enabled")

	// Apply to files first — if this fails, DB stays consistent with the old state
	if err := agents.ApplyOutputStyleToggle(enabled); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to apply style toggle: %s", err.Error())), nil
	}

	cfg := store.OutputStyleConfig{
		Version: 1,
		Enabled: enabled,
	}

	if err := s.store.SetOutputStyle(cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("applied to files but failed to save preference: %s", err.Error())), nil
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}

	return mcp.NewToolResultText(fmt.Sprintf("**Output Style:** %s (applied to settings)", status)), nil
}
