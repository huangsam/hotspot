package mcp_test

import (
	"context"
	"testing"

	"github.com/huangsam/hotspot/internal/contract"
	mcp_internal "github.com/huangsam/hotspot/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerHandlers_ValidationErrors(t *testing.T) {
	baseCfg := &contract.Config{
		RepoPath: ".",
		Mode:     "hot",
	}

	// Create a dummy manager, though we shouldn't hit it because we test validation errors
	var mgr contract.CacheManager
	s := mcp_internal.NewMCPServer(baseCfg, mgr)

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
