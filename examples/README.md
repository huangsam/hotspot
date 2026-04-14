# Hotspot Configuration Examples

This directory contains example configurations organized by use case and context.

## Quick Start

**If you're analyzing a repository locally**, start with a **CLI** example that matches your codebase:
- **Large monorepo?** → [`cli/hotspot.large.yml`](./cli/hotspot.large.yml)
- **Small service or module?** → [`cli/hotspot.small.yml`](./cli/hotspot.small.yml)
- **Infrastructure-as-code (IaC)?** → [`cli/hotspot.infra.yml`](./cli/hotspot.infra.yml)
- **CI/CD pipeline enforcement?** → [`cli/hotspot.ci.yml`](./cli/hotspot.ci.yml)

**If you're integrating Hotspot with an AI agent**, use:
- **MCP server** → [`mcp/hotspot.mcp.yml`](./mcp/hotspot.mcp.yml)

**If you need detailed guidance**, see:
- **Complete reference** → [`reference/hotspot.docs.yml`](./reference/hotspot.docs.yml)
- **Advanced weight tuning** → [`reference/hotspot.weights.yml`](./reference/hotspot.weights.yml)

## Directory Structure

### `cli/`
Ready-to-use configurations for different repository types and organizational contexts:
- **hotspot.large.yml** — Monorepos with many services, contributors, and deep histories
- **hotspot.small.yml** — Single-purpose tools, microservices, or libraries
- **hotspot.infra.yml** — Infrastructure-as-code: Terraform, Ansible, Helm, etc.
- **hotspot.ci.yml** — CI/CD integration and policy enforcement

### `mcp/`
Configurations optimized for AI agent integration via the Model Context Protocol:
- **hotspot.mcp.yml** — MCP server defaults (higher precision, structured output)

### `reference/`
Comprehensive documentation and advanced customization templates:
- **hotspot.docs.yml** — Complete reference listing every available option
- **hotspot.weights.yml** — Advanced examples for custom score weight tuning

## Getting Started

Copy a relevant config to your repository root or home directory as `.hotspot.yaml` or `.hotspot.yml`:

```bash
# For a large monorepo
cp examples/cli/hotspot.large.yml /path/to/repo/.hotspot.yml

# For a small service
cp examples/cli/hotspot.small.yml /path/to/repo/.hotspot.yml
```

Then run Hotspot — it will automatically load the config:

```bash
hotspot files
```

To override specific settings from the command line:

```bash
hotspot files --limit 20 --mode risk
```

See [USERGUIDE.md](../USERGUIDE.md) for complete documentation.
