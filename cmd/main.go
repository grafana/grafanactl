package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grafana/grafanactl/cmd/root"
	"github.com/grafana/grafanactl/internal/fail"
)

var version = "SNAPSHOT"

func main() {
	handleError(root.Command(version).Execute())
}

func handleError(err error) {
	if err == nil {
		return
	}

	exitCode := 1

	detailedErr := fail.DetailedError{}
	if !errors.As(err, &detailedErr) {
		detailedErr = fail.DetailedError{
			Parent:  err,
			Summary: "Unexpected error",
		}
	}

	fmt.Fprintln(os.Stderr, detailedErr.Error())

	if detailedErr.ExitCode != nil {
		exitCode = *detailedErr.ExitCode
	}

	os.Exit(exitCode)
}
