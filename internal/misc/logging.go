package misc

import (
	"go.uber.org/zap"
)

var bblogger *zap.SugaredLogger

func GetLogger() *zap.SugaredLogger {
	if bblogger != nil {
		return bblogger
	}
	zlogger, _ := zap.NewDevelopment()
	zsugar := zlogger.Sugar()

	bblogger = zsugar
	return bblogger
}

func GracefulShutdown() {
	bblogger.Sync()
}
