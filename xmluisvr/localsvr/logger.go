package localsvr

import (
	"log/slog"
)

var logger *slog.Logger

func Logger() *slog.Logger {
	return logger
}

type setLoggerFunc = func(*slog.Logger)

var setLoggerFuncs = make([]setLoggerFunc, 0)

func RegisterSetLoggerFunc(fn setLoggerFunc) {
	setLoggerFuncs = append(setLoggerFuncs, fn)
}

func SetLogger(l *slog.Logger) {
	logger = l
	for _, fn := range setLoggerFuncs {
		fn(logger)
	}
}

func EnsureLogger() *slog.Logger {
	if logger == nil {
		panic("Must call localsvr.SetLogger() with a *slog.Logger before reaching this check.")
	}
	return logger
}
