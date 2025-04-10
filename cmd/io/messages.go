package io

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

//nolint:gochecknoglobals
var (
	Green = color.New(color.FgGreen).SprintfFunc()
	Blue  = color.New(color.FgBlue).SprintfFunc()
)

func Success(stdout io.Writer, message string, args ...any) {
	green := color.New(color.FgGreen).SprintFunc()
	msg := fmt.Sprintf(message, args...)

	fmt.Fprintln(stdout, green("✓ ")+msg)
}

func Info(stdout io.Writer, message string, args ...any) {
	msg := fmt.Sprintf(message, args...)

	fmt.Fprintln(stdout, Blue("🛈 ")+msg)
}
