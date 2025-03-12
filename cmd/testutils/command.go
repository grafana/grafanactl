package testutils

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type CommandAssertion func(*testing.T, CommandResult)

func CommandSuccess() CommandAssertion {
	return func(t *testing.T, result CommandResult) {
		t.Helper()

		require.NoError(t, result.Err)
	}
}

func CommandErrorContains(message string) CommandAssertion {
	return func(t *testing.T, result CommandResult) {
		t.Helper()

		require.Error(t, result.Err)
		require.ErrorContains(t, result.Err, message)
	}
}

func CommandOutputContains(expected string) CommandAssertion {
	return func(t *testing.T, result CommandResult) {
		t.Helper()

		require.Contains(t, result.Stdout, expected)
	}
}

type CommandResult struct {
	Err    error
	Stdout string
}

type CommandTestCase struct {
	Cmd     *cobra.Command
	Command []string

	Assertions []CommandAssertion

	Stdin io.Reader
}

func (testCase CommandTestCase) Run(t *testing.T) {
	t.Helper()

	var stdin io.Reader = &bytes.Buffer{}
	if testCase.Stdin != nil {
		stdin = testCase.Stdin
	}
	stdout := &bytes.Buffer{}

	// To avoid polluting the tests output
	testCase.Cmd.SilenceErrors = true

	testCase.Cmd.SetIn(stdin)
	testCase.Cmd.SetOut(stdout)
	testCase.Cmd.SetArgs(testCase.Command)

	err := testCase.Cmd.Execute()
	result := CommandResult{
		Err:    err,
		Stdout: stdout.String(),
	}

	for _, assertion := range testCase.Assertions {
		assertion(t, result)
	}
}

func WithTempFile(t *testing.T, content string, with func(*testing.T, string)) {
	t.Helper()

	file, err := os.CreateTemp("", "grafanactl_tests_")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.Write([]byte(content))
	require.NoError(t, err)

	with(t, file.Name())
}
