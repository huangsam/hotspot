package mcp

import (
	"context"
	"testing"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerHandlers_ValidationErrors(t *testing.T) {
	baseCfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: ".",
		},
		Scoring: config.ScoringConfig{
			Mode: "hot",
		},
	}

	// Create a dummy manager and mock client
	var mgr iocache.CacheManager
	client := &git.MockGitClient{}
	s := NewMCPServer(baseCfg, mgr, client, "")

	ctx := context.Background()

	t.Run("compare_hotspots missing base_ref", func(t *testing.T) {
		tool := s.GetTool("compare_hotspots")
		require.NotNil(t, tool, "Tool compare_hotspots should exist")

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "compare_hotspots",
				Arguments: map[string]any{
					"base_ref": "", // Missing required
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err, "The MCP handler should not return a raw error for tool logic failures")
		assert.True(t, res.IsError, "The response should indicate an error state")
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "--base-ref is required")
	})

	t.Run("get_timeseries invalid interval", func(t *testing.T) {
		tool := s.GetTool("get_timeseries")
		require.NotNil(t, tool, "Tool get_timeseries should exist")

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_timeseries",
				Arguments: map[string]any{
					"path":     "main.go",
					"points":   5.0,
					"interval": "invalid_interval", // Invalid
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.True(t, res.IsError, "The response should indicate an error state")
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "invalid interval")
	})

	t.Run("get_timeseries invalid points", func(t *testing.T) {
		tool := s.GetTool("get_timeseries")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_timeseries",
				Arguments: map[string]any{
					"path":     "main.go",
					"points":   0.0, // Invalid
					"interval": "1 week",
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "--points must be at least 1")
	})
}

func TestMCPServer_ToolRegistration(t *testing.T) {
	s := NewMCPServer(&config.Config{}, nil, &git.MockGitClient{}, "")
	tools := s.ListTools()

	expectedTools := []string{
		"get_repo_shape",
		"get_files_hotspots",
		"get_folders_hotspots",
		"compare_hotspots",
		"get_timeseries",
		"get_release_journey",
		"get_blast_radius",
	}

	// Verify the count matches to ensure the test list is updated when new tools are added.
	assert.Equal(t, len(expectedTools), len(tools), "Tool count mismatch! If you added a tool, update TestMCPServer_ToolRegistration.")

	for _, name := range expectedTools {
		t.Run(name, func(t *testing.T) {
			_, ok := tools[name]
			assert.True(t, ok, "Tool %s should be registered", name)
		})
	}
}
