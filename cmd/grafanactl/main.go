package main

import (
	"fmt"
	"os"

	"github.com/grafana/grafanactl/cmd/grafanactl/fail"
	"github.com/grafana/grafanactl/cmd/grafanactl/root"
)

// Version variables which are set at build time.
var (
	version string
	//nolint:gochecknoglobals
	commit string
	//nolint:gochecknoglobals
	date string
)

func main() {
	handleError(root.Command(formatVersion()).Execute())
}

func handleError(err error) {
	if err == nil {
		return
	}

	exitCode := 1
	detailedErr := fail.ErrorToDetailedError(err)

	fmt.Fprintln(os.Stderr, detailedErr.Error())

	if detailedErr.ExitCode != nil {
		exitCode = *detailedErr.ExitCode
	}

	os.Exit(exitCode)
}

func formatVersion() string {
	if version == "" {
		version = "SNAPSHOT"
	}

	return fmt.Sprintf("%s built from %s on %s", version, commit, date)
}
