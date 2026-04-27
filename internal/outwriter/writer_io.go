package outwriter

import (
	"fmt"
	"io"
	"os"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// selectOutputFile returns the appropriate file handle for output, based on the provided
// file path and format type. It falls back to os.Stdout on error.
func selectOutputFile(filePath string) (*os.File, error) {
	if filePath == "" {
		return os.Stdout, nil
	}
	return os.Create(filePath)
}

// WriteWithOutputFile handles the common pattern of opening a file, writing to it, and cleaning up.
// It accepts a writer function that takes an io.Writer and returns an error.
func WriteWithOutputFile(output config.OutputSettings, writer func(io.Writer) error, successMsg string) error {
	file, err := selectOutputFile(output.GetOutputFile())
	if err != nil {
		return err
	}
	// Only close if it's not stdout
	if file != os.Stdout {
		defer func() { _ = file.Close() }()
	}

	if err := writer(file); err != nil {
		return err
	}

	if file != os.Stdout && output.GetFormat() != schema.NoneOut {
		_, _ = fmt.Fprintf(os.Stderr, "%s to %s\n", successMsg, output.GetOutputFile())
	}
	return nil
}
