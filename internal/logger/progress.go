package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/schollz/progressbar/v3"
)

// Progress defines an interface for tracking the progress of long-running operations.
type Progress interface {
	// Update sends a progress update.
	Update(current, total int, msg string)
	// IncrWarn increments the warning count and may change progress color.
	IncrWarn()
	// IncrError increments the error count and may change progress color.
	IncrError()
	// Complete finishes the progress tracking.
	Complete(msg string)
}

// NewProgress returns a new Progress instance.
// It returns a real progress bar if the output is an interactive terminal,
// or if the log level is INFO or lower.
func NewProgress() Progress {
	if IsQuiet() {
		return &noopProgress{}
	}

	isTerminal := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
	isInfo := Level.Level() <= slog.LevelInfo

	if isTerminal || isInfo {
		return &terminalProgress{
			interactive: isTerminal,
		}
	}
	return &noopProgress{}
}

type terminalProgress struct {
	bar         *progressbar.ProgressBar
	warnCount   int
	errorCount  int
	stateColor  string
	interactive bool
}

func (p *terminalProgress) Update(current, total int, msg string) {
	// If not interactive and log level is too high, skip update
	if !p.interactive && Level.Level() > slog.LevelInfo {
		return
	}

	currentColor := p.getColor()
	// If color state changed, we need to reset the bar to apply the new theme.
	if p.bar != nil && p.stateColor != currentColor {
		_ = p.bar.Clear()
		p.bar = nil
	}

	if p.bar == nil {
		p.stateColor = currentColor
		p.bar = p.createBar(total, msg, p.stateColor)
		// Set initial value immediately to avoid 0% flash
		_ = p.bar.Set(current)
	}

	p.bar.Describe(msg)
	_ = p.bar.Set(current)
}

func (p *terminalProgress) getColor() string {
	if p.errorCount > 0 {
		return "red"
	} else if p.warnCount > 0 {
		return "yellow"
	}
	return "green"
}

func (p *terminalProgress) IncrWarn() {
	p.warnCount++
}

func (p *terminalProgress) IncrError() {
	p.errorCount++
}

func (p *terminalProgress) createBar(total int, msg string, color string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(msg),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        fmt.Sprintf("[%s]=[reset]", color),
			SaucerHead:    fmt.Sprintf("[%s]>[reset]", color),
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

func (p *terminalProgress) Complete(msg string) {
	if !p.interactive && Level.Level() > slog.LevelInfo {
		return
	}
	if p.bar != nil {
		_ = p.bar.Clear()
		_ = p.bar.Finish()
	}

	if msg != "" {
		status := "success"
		if p.errorCount > 0 {
			status = "errors"
		} else if p.warnCount > 0 {
			status = "warnings"
		}

		if p.warnCount > 0 || p.errorCount > 0 {
			Info(fmt.Sprintf("%s (Status: %s, Warnings: %d, Errors: %d)", msg, status, p.warnCount, p.errorCount))
		} else {
			Info(msg)
		}
	}
}

type noopProgress struct{}

func (p *noopProgress) Update(_, _ int, _ string) {}
func (p *noopProgress) IncrWarn()                 {}
func (p *noopProgress) IncrError()                {}
func (p *noopProgress) Complete(_ string)         {}
