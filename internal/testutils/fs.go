package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func CreateTempFile(t *testing.T, content string) (string, func()) {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "grafanactl_tests_")
	require.NoError(t, err)

	_, err = file.WriteString(content)
	require.NoError(t, err)

	return file.Name(), func() {
		_ = os.Remove(file.Name())
	}
}
