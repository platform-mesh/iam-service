package logger

import (
	"context"
	"time"

	commonslogger "github.com/platform-mesh/golang-commons/logger"
	"gorm.io/gorm/logger"
)

// Logger implements the GORM logger interface for dxplogger
type Logger struct {
	commonslogger *commonslogger.Logger
}

func NewFromLogger(logger *commonslogger.Logger) Logger {
	return Logger{
		commonslogger: logger,
	}
}

func (l Logger) LogMode(level logger.LogLevel) logger.Interface { // nolint: ireturn
	return Logger{commonslogger: l.commonslogger.Level(commonslogger.Level(level))} // nolint: gosec // no risk of integer overflow
}

func (l Logger) Info(ctx context.Context, s string, i ...interface{}) {
	l.commonslogger.Info().Fields(i).Msgf(s, i...)
}

func (l Logger) Warn(ctx context.Context, s string, i ...interface{}) {
	l.commonslogger.Warn().Fields(i).Msgf(s, i...)
}

func (l Logger) Error(ctx context.Context, s string, i ...interface{}) {
	l.commonslogger.Error().Fields(i).Msgf(s, i...)
}

func (l Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()
	l.commonslogger.Info().Msgf("%s [%s]", sql, elapsed)
}
