package logs

import (
	"log/slog"

	"github.com/grafana/grafana-app-sdk/logging"
)

type StaticLevelLogger struct {
	logger logging.Logger
	level  slog.Level
}

func (d *StaticLevelLogger) Print(v ...interface{}) {
	if len(v) == 0 {
		return
	}

	var logFunc func(msg string, args ...any)

	switch d.level {
	case slog.LevelDebug:
		logFunc = d.logger.Debug
	case slog.LevelInfo:
		logFunc = d.logger.Info
	case slog.LevelWarn:
		logFunc = d.logger.Warn
	case slog.LevelError:
		logFunc = d.logger.Error
	default:
		logFunc = d.logger.Debug
	}

	//nolint:forcetypeassert
	logFunc(v[0].(string))
}

func DecorateAtLevel(l logging.Logger, level slog.Level) *StaticLevelLogger {
	return &StaticLevelLogger{
		logger: l,
		level:  level,
	}
}
