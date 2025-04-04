package testutils

import (
	"io"
	"log/slog"

	"github.com/grafana/grafanactl/internal/logs"
)

func NullLogger() *slog.Logger {
	return slog.New(logs.NewHandler(io.Discard, &logs.Options{}))
}
