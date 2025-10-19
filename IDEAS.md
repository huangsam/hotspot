# Future ideas

Code Quality & Testing

- Add tests - Even basic ones for core functions:
- Error handling - Your current code has some silent errors
- Input validation for
    - Non-existent repo paths?
    - Repos with no commits?
    - Empty files?
    - Binary files?

Distribution

- GitHub Actions / CI - Automate builds and tests:
- Release automation - Use GoReleaser for multi-platform binaries
- Installation script - Make it easier for users without Go installed

Performance & Robustness

- Progress indicators - For large repos, show progress (currently just emojis at start)
- Graceful interruption - Handle Ctrl+C cleanly
- Memory optimization - Large repos might consume lots of memory with current buffering
- Git availability check - Verify git is installed before running

User Experience

- Config file support - .hotspot.yml for default settings per repo
- Better error messages - More helpful feedback when things go wrong
- Validation - Check mode names, date formats earlier with clear errors
- Shell completion - Bash/Zsh completion scripts for flags
