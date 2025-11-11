# VHS demo script for Hotspot CLI
#
# This script demonstrates running Hotspot locally on the current repository.
# It builds the tool and runs basic analysis commands.

Output images/demo.gif

Require go

Set Shell "zsh"
Set FontSize 18
Set Width 1280
Set Height 720
Set TypingSpeed 100ms

# --- 1. Diagnose Technical Debt (Complexity Mode) ---
Type "hotspot files --mode complexity --limit 8" Enter
Sleep 1.5s

# --- 2. Trend Analysis (Show Risk Evolution) ---
Type "hotspot timeseries --path main.go --interval '10 days' --points 3" Enter
Sleep 2s

# --- 3. Transparency (Show the Scoring Formulas) ---
Type "hotspot metrics" Enter
Sleep 2s

Type "exit" Enter
Sleep 0.5s
