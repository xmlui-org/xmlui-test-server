package xmluisvr

import (
	"log/slog"

	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// logger is the package-level logger instance.
//
//nolint:unused
var logger *slog.Logger

func init() {
	localsvr.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
	})
}
