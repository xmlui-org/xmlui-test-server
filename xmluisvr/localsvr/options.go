package localsvr

import (
	"time"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

var _ interface{ Options() } = (*Options)(nil)

type Options struct {
	Timeout               time.Duration
	HTTPPort              ServerPort
	DBExtensionFiles      []dt.Filepath
	APIFile               dt.Filepath
	ConnectString         ConnectString
	DBPort                ServerPort
	DBBootstrapFile       dt.Filepath
	ErrorStyle            ErrorStyle
	AllowUntrustedQueries bool
	GlobalOptions         *cliutil.GlobalOptions
	Webroot               dt.DirPath
}

func (Options) Options() {}
