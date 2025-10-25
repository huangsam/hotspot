package internal

import (
	"fmt"
	"os"
)

// FatalError logs an error and exits the program.
func FatalError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}

// Warning logs a warning.
func Warning(msg string) {
	fmt.Fprintf(os.Stderr, "⚠️  %s\n", msg)
}
