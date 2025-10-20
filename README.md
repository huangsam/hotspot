# Hotspot

Hotspot is the Git analyzer that cuts through history to instantly show you which files are your greatest risk.

## Quick Start

```bash
go install github.com/huangsam/hotspot@latest
hotspot .
```

## What It Does

Hotspot analyzes your Git history and ranks files by importance using different scoring modes — helping you find:

- 🔥 Active areas — where most work happens
- ⚠️ Risky files — single-owner or volatile code
- 🧩 Complex hotspots — technical debt candidates
- 💤 Stale files — neglected but still important

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
- **With `-follow` flag**: 2-3x slower (only runs on top N results)
- **With `-start` flag**: Fast single-pass aggregation

Adjust `-workers` based on your CPU cores for optimal performance.

## Tips

- Start with `hotspot .` to get a quick snapshot
- Use `-mode risk` to find fragile ownership
- Add `-start` for sprint/release-based analysis
- Combine `-filter` and `-exclude` for focus
- Use `-explain` to understand the scoring
