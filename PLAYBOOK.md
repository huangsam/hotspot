# Hotspot Remediation Playbook

High scores in Hotspot aren't "errors"—they are **signals**. This guide helps Tech Leads, Managers, and Developers translate Git metrics into healthy engineering conversations and actionable improvements.

## The Goal: From Diagnosis to Support

The core mission of `hotspot` is to surface **technical debt** and **knowledge risk**. However, these are often symptoms of systemic issues, not individual performance problems.

### Red Flag: The Blame Trap

If you use Hotspot to ask "Why did you write such complex code?" or "Why are you the only one who knows this module?", you have already failed. This leads to metric gaming, hidden debt, and developer burnout.

## Remediation Strategies

### Area 1: High Complexity Score (The "Nightmare" File)

**The Signal**: A file is large, high churn, and aging.
- **Supportive Action**:
    - **Refactor Sprint**: Explicitly allocate time to break down the file.
    - **Pair Programming**: Assign a "complexity buddy" to help simplify logic.
    - **Test Coverage**: Increase unit tests before refactoring to reduce fear.

### Area 2: High Risk Score (The "Knowledge Island")

**The Signal**: One person owns 90% of a critical path (High Bus Factor).
- **Supportive Action**:
    - **Shadowing**: Have another dev shadow the owner for a week.
    - **Rotation**: Temporarily rotate the owner out of that module to force knowledge transfer.
    - **Documentation**: Ask the owner to record "Architecture Decision Records" (ADRs) for that area.

### Area 3: High Stale Score (The "Lurking Dragon")

**The Signal**: Large, old files with high historical importance that haven't been touched recently.
- **The Risk**: These are often "load-bearing" legacy files that no one dares to touch. When they eventually *must* change, they cause outages.
- **Supportive Action**:
    - **Modernization Spike**: Dedicate a spike to analyze if the logic is still needed or can be ported to a newer service.
    - **Verification Loop**: Ensure unit tests still run and pass against current dependencies.

### Area 4: High Hot Score (The "Active Volcano")

**The Signal**: Extreme recent activity and churn.
- **The Context**: This is normal during a feature launch, but dangerous if it persists for months.
- **Supportive Action**:
    - **Complexity Check**: Run `hotspot files --mode complexity` on the same path. If both are high, the file is likely a "God Object" that needs splitting.
    - **Cool-down Period**: If the churn is driven by bug-fix loops, pause feature work to stabilize the architecture.

### Area 5: High ROI Score (The "Refactoring Goldmine")

**The Signal**: High maintenance burden on complex files where investment will yield the most impact.
- **Supportive Action**:
    - **Strategic Planning**: Use the `describe` output (`--output describe`) to generate an executive summary for stakeholders to justify refactoring time.
    - **ROI Target**: Focus on these files first to get the most "bang for your buck" in terms of improved development velocity.
    - **Impact Audit**: After refactoring, run `hotspot compare` in ROI mode to quantify the technical return on investment.

## Closing the Loop: Measuring Success

Data is only useful if it shows progress. Use these commands to quantify your impact:

- **Audit a Refactor**: Run `hotspot compare files --base-ref v1.0 --target-ref HEAD --mode complexity`. A successful refactor should show a significant delta decrease in complexity scores.
- **Track Trends**: Use `hotspot timeseries --path <path> --mode risk` to prove that "Knowledge Islands" are shrinking over time as ownership is shared.

## Hotspot in Agile Rituals

- **Sprint Planning**: Before starting work on a legacy module, run `hotspot check`. If it fails thresholds, add "Technical Debt Cleanup" as a sub-task for the story.
- **Retrospectives**: Share the `timeseries` for the team's core subsystem. Celebrate when the trend line for "Risk" or "Complexity" goes down.
- **Onboarding**: Give new joins a list of "High Risk" files from Hotspot so they know where to ask for extra review.

## Managing Upward (The "Ambiguity" Guide)

Communication with leadership can be tough. Use these scripts to handle common leadership ambiguities:

### "Why didn't we see this complexity earlier?"

**The Reframe**: "Complexity grows naturally with features. We didn't 'miss' it; we've now reached the scale where the architecture needs to evolve to support our next growth milestone. Hotspot helped us identify exactly where that investment will yield the highest ROI."

### "Make the hotspots go to zero by next month."

**The Reframe**: "A score of zero isn't the goal—balanced risk is. Some high-activity areas (Hot) are healthy during a feature launch. We are focusing our energy on the 'Complexity' and 'Risk' silos that actually threaten our stability, rather than chasing a vanity metric."

## Policy Enforcement Ethics (CI/CD)

The `hotspot check` command is a powerful tool, but it must be used with care:

1. **Start with "Soft Fails"**: When first introducing Hotspot to a pipeline, configure it to report results without failing the build. This allows the team to calibrate thresholds.
2. **Exemptions are Healthy**: Provide a mechanism (like `hotspot.yaml` excludes) for legacy files that the team has explicitly decided *not* to refactor yet.
3. **Discussion over Blocking**: Use a "High Risk" alert as a prompt for a senior engineer to provide a more detailed code review, rather than a robotic "Request Changes."

## The Empathy Ethics

1. **Be Public with Goals, Private with Growth**: Discuss subsystem risk in team meetings, but discuss individual contribution patterns in 1-on-1s.
2. **Context is King**: A score of 95.0 in a legacy "legacy-adapter.go" might be acceptable; the same score in a new "core-logic.go" is a signal to stop and pivot.
3. **Data is a Mirror, Not a Hammer**: Use it to reflect the current state of the system, not to pound people into submission.
