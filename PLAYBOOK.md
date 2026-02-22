# Hotspot Remediation Playbook

High scores in Hotspot aren't "errors"—they are **signals**. This guide helps Tech Leads, Managers, and Developers translate Git metrics into healthy engineering conversations and actionable improvements.

## The Goal: From Diagnosis to Support

The core mission of `hotspot` is to surface **technical debt** and **knowledge risk**. However, these are often symptoms of systemic issues, not individual performance problems.

### Red Flag: The Blame Trap

If you use Hotspot to ask "Why did you write such complex code?" or "Why are you the only one who knows this module?", you have already failed. This leads to metric gaming, hidden debt, and developer burnout.

---

## Remediation Strategies

### Area 1: High Complexity Score (The "Nightmare" File)

**The Signal**: A file is large, high churn, and aging.
- **Supportive Action**:
    - **Refactor Sprint**: Explicitly allocate time to break down the file.
    - **Pair Programming**: Assign a "complexity buddy" to help simplify logic.
    - **Test Coverage**: Increase unit tests before refactoring to reduce fear.

### Area 2: High Risk Score (The "Knowledge Island")

**The Signal**: One person owns 90% of a critical path.
- **Supportive Action**:
    - **Shadowing**: Have another dev shadow the owner for a week.
    - **Rotation**: Temporarily rotate the owner out of that module to force knowledge transfer.
    - **Documentation**: Ask the owner to record "Architecture Decison Records" (ADRs) for that area.

---

## Managing Upward (The "Ambiguity" Guide)

Communication with leadership can be tough, especially when they demand "zero risk" or "perfect scores." Use these scripts to handle common leadership ambiguities:

### "Why didn't we see this complexity earlier?"

**The Reframe**: "Complexity grows naturally with features. We didn't 'miss' it; we've now reached the scale where the architecture needs to evolve to support our next growth milestone. Hotspot helped us identify exactly where that investment will yield the highest ROI."

### "Make the hotspots go to zero by next month."

**The Reframe**: "A score of zero isn't the goal—balanced risk is. Some high-activity areas (Hot) are healthy during a feature launch. We are focusing our energy on the 'Complexity' and 'Risk' silos that actually threaten our stability, rather than chasing a vanity metric."

---

## The Empathy Ethics

1. **Be Public with Goals, Private with Growth**: Discuss subsystem risk in team meetings, but discuss individual contribution patterns in 1-on-1s.
2. **Context is King**: A score of 95.0 in a legacy "legacy-adapter.go" might be acceptable; the same score in a new "core-logic.go" is a signal to stop and pivot.
3. **Data is a Mirror, Not a Hammer**: Use it to reflect the current state of the system, not to pound people into submission.
