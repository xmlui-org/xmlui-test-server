package dbpkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

type QuerySource struct {
	Lines    [2]int
	Source   localsvr.QueryString
	Filepath dt.Filepath
}

func NewQuerySource(start, end int, src localsvr.QueryString, fp dt.Filepath) *QuerySource {
	return &QuerySource{
		Lines:    [2]int{start, end},
		Source:   src,
		Filepath: fp,
	}
}
