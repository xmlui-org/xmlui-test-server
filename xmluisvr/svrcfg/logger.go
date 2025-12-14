package cfgldr

import (
	"log/slog"

	cfgstore "github.com/mikeschinkel/go-cfgstore"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

//nolint:unused
var logger *slog.Logger

func init() {
	localsvr.RegisterSetLoggerFunc(func(l *slog.Logger) {
		logger = l
		// This package `cfgldr` depends on `cfgstore` which is an independent package
		// which should not be coupled to `localsvr` like this package is, so we initialize
		// its logger here.
		cfgstore.SetLogger(l)
	})
}
