package internal

import (
	"fmt"
	"os"
)

// LogFatal logs an error and exits the program.
func LogFatal(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}
