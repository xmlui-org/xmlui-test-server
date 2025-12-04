package apipkg

import (
	"log/slog"

	"github.com/mikeschinkel/go-rfc9457"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

// logger provides package-level logging functionality using the common logger instance.
//
//nolint:unused
var logger *slog.Logger

func init() {
	common.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
		rfc9457.SetLogger(l)
	})
}
