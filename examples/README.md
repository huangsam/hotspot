# Hotspot Configuration Examples

This directory contains example configurations and references for advanced customization.

## Quick Start

**If you're analyzing a repository locally**, we recommend using the built-in shape analysis to automatically generate a configuration:

```bash
hotspot init
```

This will analyze your repository and create a `.hotspot.yml` tailored to your codebase. Alternatively, you can specify a preset directly:
- `hotspot init --preset large` (Monorepos)
- `hotspot init --preset small` (Microservices/Tools)
- `hotspot init --preset infra` (IaC/Terraform)

## Reference Documentation

### [`reference/`](./reference/)
Comprehensive documentation and advanced templates:
- **[Complete Reference](./reference/hotspot.docs.yml)** — Every available configuration option documented.
- **[CI/CD Policy](./reference/hotspot.ci.yml)** — Example configuration for build gating and risk thresholds.
- **[Weight Tuning](./reference/hotspot.weights.yml)** — Advanced examples for custom score algorithm adjustments.

### [`mcp/`](./mcp/)
Configurations optimized for AI agent integration via the Model Context Protocol:
- **[MCP Config](./mcp/hotspot.mcp.yml)** — MCP server defaults (higher precision, structured output).

## Getting Started

The easiest way to manage settings is by using the `init` command. To see the canonical definitions for all built-in presets, refer to [schema/data/presets.yaml](../schema/data/presets.yaml).

To override specific settings from the command line:

```bash
hotspot files --limit 20 --mode risk
```

See [USERGUIDE.md](../USERGUIDE.md) for complete documentation.
