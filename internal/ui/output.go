package ui

import (
	"fmt"
	"io"
	"os"
)

// UI handles user interface output with verbosity levels
type UI struct {
	Out       io.Writer
	Err       io.Writer
	Verbosity int
}

// New creates a new UI instance
func New(out, err io.Writer, verbosity int) *UI {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &UI{
		Out:       out,
		Err:       err,
		Verbosity: verbosity,
	}
}

// Comment prints a comment (requires verbosity >= 1)
func (u *UI) Comment(format string, args ...interface{}) {
	if u.Verbosity >= 1 {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(u.Out, "# %s\n", msg)
	}
}

// Info prints informational output (requires verbosity >= 0)
func (u *UI) Info(format string, args ...interface{}) {
	if u.Verbosity >= 0 {
		fmt.Fprintf(u.Out, format+"\n", args...)
	}
}

// Warn prints a warning message (requires verbosity >= -1)
func (u *UI) Warn(format string, args ...interface{}) {
	if u.Verbosity >= -1 {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(u.Err, "warning: %s\n", msg)
	}
}

// Error prints an error message (always shown)
func (u *UI) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(u.Err, "error: %s\n", msg)
}

// Emit prints raw output without formatting (requires verbosity >= 0)
func (u *UI) Emit(format string, args ...interface{}) {
	if u.Verbosity >= 0 {
		fmt.Fprintf(u.Out, format, args...)
	}
}

// EmitLn prints raw output with newline (requires verbosity >= 0)
func (u *UI) EmitLn(format string, args ...interface{}) {
	if u.Verbosity >= 0 {
		fmt.Fprintf(u.Out, format+"\n", args...)
	}
}

// Confirm prompts for yes/no confirmation
func (u *UI) Confirm(prompt string) bool {
	fmt.Fprintf(u.Out, "%s [y/N] ", prompt)
	var response string
	fmt.Fscanln(os.Stdin, &response)
	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}
