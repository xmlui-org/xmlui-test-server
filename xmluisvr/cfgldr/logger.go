package cfgldr

import (
	"log/slog"

	cfgstore "github.com/mikeschinkel/go-cfgstore"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

//nolint:unused
var logger *slog.Logger

func init() {
	common.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
		// This package `cfgldr` depends on `cfgstore` which is an independent package
		// which should not be coupled to `common` like this package is, so we initialize
		// its logger here.
		cfgstore.SetLogger(l)
	})
}
