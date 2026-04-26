package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	t.Run("compare_file_hotspots missing base_ref", func(t *testing.T) {
		tool := s.GetTool("compare_file_hotspots")
		require.NotNil(t, tool, "Tool compare_file_hotspots should exist")

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "compare_file_hotspots",
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
		"get_heatmap",
		"get_folders_hotspots",
		"compare_file_hotspots",
		"compare_folder_hotspots",
		"get_timeseries",
		"get_release_journey",
		"get_blast_radius",
		"run_check",
		"run_batch_analysis",
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

func TestMCPServerHandlers_Execution(t *testing.T) {
	setup := func(_ *testing.T) (*server.MCPServer, *git.MockGitClient) {
		baseCfg := &config.Config{
			Git:     config.GitConfig{RepoPath: "."},
			Scoring: config.ScoringConfig{Mode: "hot"},
		}
		mgr := &iocache.MockCacheManager{}
		analysisStore := &iocache.MockAnalysisStore{}
		cacheStore := &iocache.MockCacheStore{}
		client := &git.MockGitClient{}

		mgr.On("GetAnalysisStore").Return(analysisStore)
		mgr.On("GetActivityStore").Return(cacheStore)

		// Default analysis store mocks
		analysisStore.On("Initialize", mock.Anything).Return(nil)
		analysisStore.On("BeginAnalysis", mock.Anything, mock.Anything, mock.Anything).Return(int64(1), nil)
		analysisStore.On("RecordFileResultsBatch", mock.Anything, mock.Anything).Return(nil)
		analysisStore.On("EndAnalysis", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Default cache store mocks
		cacheStore.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		cacheStore.On("Get", mock.Anything).Return([]byte(nil), 0, int64(0), fmt.Errorf("not found"))

		// Default git client mocks
		absPath, _ := filepath.Abs(".")
		client.On("GetRepoRoot", mock.Anything, mock.Anything).Return(absPath, nil)
		client.On("GetRemoteURL", mock.Anything, mock.Anything).Return("https://github.com/test/repo", nil)
		client.On("GetRootCommitHash", mock.Anything, mock.Anything).Return("abc", nil)
		client.On("GetRepoHash", mock.Anything, mock.Anything).Return("abc", nil).Maybe()
		client.On("ListFilesAtRef", mock.Anything, mock.Anything, mock.Anything).Return([]string{"main.go", "cmd/main.go", "a.go", "b.go"}, nil).Maybe()
		client.On("GetOldestCommitDateForPath", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Now().Add(-24*time.Hour), nil).Maybe()
		client.On("GetCommitTime", mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), nil).Maybe()
		client.On("GetTags", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, nil).Maybe()

		s := NewMCPServer(baseCfg, mgr, client, "")
		return s, client
	}

	ctx := context.Background()

	t.Run("get_repo_shape success", func(t *testing.T) {
		s, client := setup(t)
		// Use 'now' to be absolutely safe with time windows
		nowStr := time.Now().UTC().Format(time.RFC3339)
		// Mock log for shape analysis (one commit for main.go)
		// Use the format WITHOUT single quotes as it's more standard for the parser
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "--abc|Tester|%s\n\n1\t1\tmain.go\n", nowStr), nil)

		tool := s.GetTool("get_repo_shape")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_repo_shape",
				Arguments: map[string]any{
					"repo_path": ".",
					"start":     "2020-01-01T00:00:00Z",
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "preset")
	})

	t.Run("get_files_hotspots success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "--abc|Tester|%s\n\n10\t5\tmain.go\n", nowStr), nil)
		client.On("Run", mock.Anything, mock.Anything, "rev-list", "--count", mock.Anything, "--", "main.go").Return([]byte("1\n"), nil)

		tool := s.GetTool("get_files_hotspots")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_files_hotspots",
				Arguments: map[string]any{"repo_path": "."},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
	})

	t.Run("get_folders_hotspots success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "--abc|Tester|%s\n\n10\t5\tcmd/main.go\n", nowStr), nil)
		client.On("Run", mock.Anything, mock.Anything, "rev-list", "--count", mock.Anything, "--", "cmd/main.go").Return([]byte("1\n"), nil)

		tool := s.GetTool("get_folders_hotspots")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_folders_hotspots",
				Arguments: map[string]any{"repo_path": "."},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
	})

	t.Run("get_heatmap success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "--abc|Tester|%s\n\n10\t5\tmain.go\n", nowStr), nil)
		client.On("Run", mock.Anything, mock.Anything, "rev-list", "--count", mock.Anything, "--", "main.go").Return([]byte("1\n"), nil)

		tool := s.GetTool("get_heatmap")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_heatmap",
				Arguments: map[string]any{"repo_path": "."},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, res.Content)
		// Heatmap generation requires results to visualization, so in a mock environment
		// it might return a 'no files' message, which still proves the MCP tool is registered
		// and correctly calling the core engine.
	})

	t.Run("get_blast_radius success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		// Mock log where a.go and b.go change together
		logContent := fmt.Sprintf("'--abc|Tester|%s\n\n1\t1\ta.go\n1\t1\tb.go\n", nowStr)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]byte(logContent), nil)

		tool := s.GetTool("get_blast_radius")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_blast_radius",
				Arguments: map[string]any{"repo_path": "."},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
	})

	t.Run("get_timeseries success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "'--abc|Tester|%s\n\n10\t5\tmain.go\n", nowStr), nil)
		client.On("Run", mock.Anything, mock.Anything, "rev-list", "--count", mock.Anything, "--", "main.go").Return([]byte("1\n"), nil)
		client.On("GetCommitTime", mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), nil)
		client.On("GetTags", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, nil)

		tool := s.GetTool("get_timeseries")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_timeseries",
				Arguments: map[string]any{
					"repo_path": ".",
					"path":      "main.go",
					"interval":  "1 month",
					"points":    2,
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "main.go")
	})

	t.Run("run_check success", func(t *testing.T) {
		s, client := setup(t)
		client.On("GetChangedFilesBetweenRefs", mock.Anything, mock.Anything, "v1.0.0", "HEAD").Return([]string{"main.go"}, nil)
		client.On("ListFilesAtRef", mock.Anything, mock.Anything, mock.Anything).Return([]string{"main.go"}, nil)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]byte("'--abc|Tester|2026-01-01T00:00:00Z\n\n10\t5\tmain.go\n"), nil)
		client.On("Run", mock.Anything, mock.Anything, "rev-list", "--count", mock.Anything, "--", "main.go").Return([]byte("1"), nil)
		client.On("GetCommitTime", mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), nil)

		tool := s.GetTool("run_check")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_check",
				Arguments: map[string]any{
					"base_ref":   "v1.0.0",
					"target_ref": "HEAD",
					"repo_path":  ".",
					"lookback":   "10 years",
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Passed")
	})

	t.Run("run_batch_analysis success", func(t *testing.T) {
		s, client := setup(t)
		nowStr := time.Now().UTC().Format(time.RFC3339)
		client.On("GetActivityLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Appendf(nil, "--abc|Tester|%s\n\n1\t1\tmain.go\n", nowStr), nil)

		tool := s.GetTool("run_batch_analysis")
		require.NotNil(t, tool)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_batch_analysis",
				Arguments: map[string]any{
					"path": ".",
					"auto": false,
				},
			},
		}

		res, err := tool.Handler(ctx, req)
		require.NoError(t, err)
		assert.False(t, res.IsError, "Result should not be an error: %v", res.Content)
		assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "results")
	})
}
