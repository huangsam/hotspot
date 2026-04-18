# Model Context Protocol (MCP) Server

Hotspot can run as an [MCP](https://modelcontextprotocol.io/) server, exposing its diagnostic capabilities via `mcp-go` to compatible AI agents (like Claude Desktop, Cursor, or Zed). This allows an LLM to automatically explore technical debt and risk without you having to manually run CLI commands.

## Starting the Server

The server communicates via standard input/output (stdio), which is standard for local MCP clients.

```bash
hotspot mcp
```

## Example: Claude Desktop Configuration

To use Hotspot with Claude Desktop, add it to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "hotspot": {
      "command": "hotspot",
      "args": ["mcp"]
    }
  }
}
```

*Note: Ensure `hotspot` is in your system `$PATH`, or provide the absolute path to the binary in the `command` field.*

## Supported MCP Tools

The server exposes the following tools to the AI agent:
- `get_repo_shape`: Characterize the repository and get a recommended preset (lightweight aggregation pass).
- `get_files_hotspots`: Rank files by hot, risk, complexity, or roi modes.
- `get_folders_hotspots`: Same as above, but aggregated at the folder level.
- `compare_file_hotspots`: Compare changes in technical debt at the file level between two Git references.
- `compare_folder_hotspots`: Same as above, but aggregated at the folder level.
- `get_timeseries`: Track the trend of a specific file or folder over time.
- `get_release_journey`: Compute repository trajectory by analyzing successive release tags.
- `get_blast_radius`: Identify files that historically change together.
- `run_check`: Run a policy check for CI/CD gating using risk thresholds.

All analysis tools support an optional `preset` parameter to auto-configure scoring mode, worker count, result limit, and time window based on the recommended preset family. Tools are annotated with `ReadOnly` and `Idempotent` hints to assist agent reasoning.

## Native Resources & Prompts

As of v1.16.0, the MCP server is self-documenting and provides guided workflows:

### 1. Documentation Resources
Agents can read core documentation directly from the tool using standard URIs:
- `hotspot://docs/agents`: Architectural context and scoring mode principles.
- `hotspot://docs/metrics`: Machine-readable JSON definition of scoring modes and weights.

### 2. Guided Playbooks (Prompts)
The server provides pre-defined analysis workflows via the `prompts/list` capability:
- `release-readiness`: A specialized workflow for assessing release safety by comparing HEAD against the last tag.
- `refactor-prioritization`: A specialized workflow using ROI mode to identify high-return targets.
