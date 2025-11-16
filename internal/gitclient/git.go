// Package gitclient has the git client.
package gitclient

import "github.com/huangsam/hotspot/internal/contract"

// GitClient defines the necessary operations for complex Git analysis.
// This allows the core analysis logic to be tested without needing a real git executable.
type GitClient = contract.GitClient
