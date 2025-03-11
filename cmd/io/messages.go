package io

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

func Success(stdout io.Writer, message string, args ...any) {
	green := color.New(color.FgGreen).SprintFunc()
	msg := fmt.Sprintf(message, args...)

	fmt.Fprintln(stdout, green("✓ ")+msg)
}
