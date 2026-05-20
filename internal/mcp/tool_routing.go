package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"mio/internal/store"
)

func (s *Server) handleMemRoutingGet(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, err := s.store.GetModelRouting()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("**Model Routing Configuration**\n\n")
	sb.WriteString(fmt.Sprintf("Available models: %s\n", strings.Join(cfg.AvailableModels, ", ")))
	sb.WriteString(fmt.Sprintf("Default planning:  %s\n", cfg.Preferences.DefaultPlanning))
	sb.WriteString(fmt.Sprintf("Default execution: %s\n", cfg.Preferences.DefaultExecution))
	sb.WriteString(fmt.Sprintf("Cost mode:         %s\n", cfg.Preferences.CostMode))

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleMemRoutingSet(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, err := s.store.GetModelRouting()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read current config: %s", err.Error())), nil
	}

	// Merge available_models if provided
	args := request.GetArguments()
	if raw, ok := args["available_models"]; ok {
		switch v := raw.(type) {
		case []interface{}:
			models := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					models = append(models, s)
				}
			}
			cfg.AvailableModels = models
		}
	}

	// Merge preference fields if provided
	if val := strArg(request, "default_planning"); val != "" {
		cfg.Preferences.DefaultPlanning = val
	}
	if val := strArg(request, "default_execution"); val != "" {
		cfg.Preferences.DefaultExecution = val
	}
	if val := strArg(request, "cost_mode"); val != "" {
		cfg.Preferences.CostMode = val
	}

	if err := store.ValidateModelRoutingConfig(cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("validation failed: %s", err.Error())), nil
	}

	if err := s.store.SetModelRouting(cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("save failed: %s", err.Error())), nil
	}

	var sb strings.Builder
	sb.WriteString("**Model Routing Updated**\n\n")
	sb.WriteString(fmt.Sprintf("Available models: %s\n", strings.Join(cfg.AvailableModels, ", ")))
	sb.WriteString(fmt.Sprintf("Default planning:  %s\n", cfg.Preferences.DefaultPlanning))
	sb.WriteString(fmt.Sprintf("Default execution: %s\n", cfg.Preferences.DefaultExecution))
	sb.WriteString(fmt.Sprintf("Cost mode:         %s\n", cfg.Preferences.CostMode))

	return mcp.NewToolResultText(sb.String()), nil
}
