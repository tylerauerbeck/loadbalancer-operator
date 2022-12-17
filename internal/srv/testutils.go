package srv

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func setupTestLogger(t *testing.T, name string) *zap.SugaredLogger {
	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
	return logger.Sugar().With("test", name)
}
