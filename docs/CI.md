# CI/CD Policy Enforcement

The `check` command allows you to enforce risk thresholds in CI/CD pipelines, failing builds when files exceed acceptable risk levels. Default thresholds: 50.0 for all scoring modes (hot, risk, complexity, roi).

## Examples

**1. Basic comparison check:**
`hotspot check --base-ref v1.9.0 --target-ref v1.10.0`

**2. Custom thresholds via CLI:**
`hotspot check --base-ref v1.9.0 --target-ref v1.10.0 --thresholds-override "hot:75,risk:60,complexity:80"`

## Reference

| Flags | Description |
|-------|-------------|
| `--base-ref` | The BEFORE Git reference (e.g., `main`, `v1.0.0`, a commit hash). |
| `--target-ref` | The AFTER Git reference (defaults to `HEAD`). |
| `--lookback` | Time window (e.g. `6 months`) used for base and target. |
| `--thresholds-override` | Custom risk thresholds per scoring mode (format: `hot:50,risk:50,complexity:50,roi:50`). |

The [example CI config](../examples/cli/hotspot.ci.yml) shows how custom thresholds can be configured for each scoring mode and is useful for maintaining code quality standards specific to your team.
