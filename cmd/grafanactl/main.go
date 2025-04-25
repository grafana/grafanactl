package main

import (
	"fmt"
	"os"

	"github.com/grafana/grafanactl/cmd/grafanactl/fail"
	"github.com/grafana/grafanactl/cmd/grafanactl/root"
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
	detailedErr := fail.ErrorToDetailedError(err)

	fmt.Fprintln(os.Stderr, detailedErr.Error())

	if detailedErr.ExitCode != nil {
		exitCode = *detailedErr.ExitCode
	}

	os.Exit(exitCode)
}
