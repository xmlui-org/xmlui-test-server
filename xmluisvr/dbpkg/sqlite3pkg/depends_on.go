package sqlite3pkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

type DependsOn struct {
	ExtensionId localsvr.ExtensionId
	Filepath    dt.Filepath
}

func (d DependsOn) String() string {
	if d.ExtensionId == "" {
		return string(d.Filepath)
	}
	return string(d.ExtensionId)
}
