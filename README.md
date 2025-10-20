# Hotspot

Hotspot is the Git analyzer that cuts through history to instantly show you which files are your greatest risk.

- 🔍 **See what matters** — rank files by activity, ownership, or complexity
- ⚡ **Fast results** — analyze thousands of files in seconds
- 🧮 **Rich insights** — contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** — narrow down by folder, exclude noise, or track trends over time

Perfect for:

- 🧑‍💻 **Developers** tracking sprint or release activity
- 🧹 **Tech leads** prioritizing refactors and risk
- 🧾 **Managers** monitoring bus factor and maintenance debt

## Quick start

```bash
go install github.com/huangsam/hotspot@latest
hotspot .
```

## Common Use Cases

### Daily & Sprint Workflows

```bash
# Current Activity Hotspots
hotspot -mode hot -start 2024-10-01T00:00:00Z -limit 15 .

# Immediate Refactoring Targets
hotspot -mode complexity -limit 20 .
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk
hotspot -mode risk -output csv -csv-file bus-factor.csv .

# Maintenance Debt Audit
hotspot -mode stale -exclude "test/,vendor/" .

# Complex Files with History
hotspot -mode complexity -limit 10 -follow .
```

## Performance

- **Typical repo (1k files)**: 2-5 seconds
- **Large repo (10k+ files)**: 15-30 seconds

Adjust `-workers` based on your CPU cores for optimal performance.

## Tips

- Start with `hotspot .` for a quick snapshot
- Combine filters and excludes for focus
- Use CSV export for tracking trends
- Try `-explain` to see how each metric affects ranking
