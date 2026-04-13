// Package contract provides utilities for the hotspot tool.
package contract

import (
	"log"
)

// LogFatal logs a fatal error message to stderr and exits the program.
func LogFatal(msg string, err error) {
	log.Fatalf("%s: %v", msg, err)
}

// LogWarn logs a warning message to stderr.
func LogWarn(msg string, err error) {
	log.Printf("%s: %v", msg, err)
}
