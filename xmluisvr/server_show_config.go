package xmluisvr

import (
	"fmt"
	"os"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

func (svr *Server) showConfig() {

	cliutil.Printf("\n")
	cliutil.Printf("Configuration:\n")
	cliutil.Printf("- Current Dir:  %v\n", svr.displayDir())
	cliutil.Printf("- Server:       %s\n", svr.Host())
	cliutil.Printf("  - Webroot:    %v\n", svr.displayWebroot())
	cliutil.Printf("  - Config:     %s\n", svr.displayServerSourceFile())
	cliutil.Printf("- Database:     %s\n", svr.displayDBTypeName())
	cliutil.Printf("  - Connection: %v\n", svr.displayDBName())
	cliutil.Printf("  - Config:     %s\n", svr.displayDBSourceFile())
	if svr.API != nil {
		cliutil.Printf("- API:          %s\n", svr.API.Name)
		cliutil.Printf("  - URL Path:   %s\n", svr.API.BasePath)
		cliutil.Printf("  - Config:     %s\n", svr.displayAPISourceFile())
	}
	if len(svr.Database.Extensions()) != 0 {
		cliutil.Printf("- Extension:   %s\n", svr.displayExtensionPaths())
	}
	cliutil.Loud().Printf("\n")

}

func (svr *Server) displayHost() string {
	return fmt.Sprintf("localhost:%d", svr.port)
}

func (svr *Server) displayWebroot() (wr string) {
	dp := svr.API.Webroot
	if !dp.IsAbs() {
		// Print current working directory
		wd, err := os.Getwd()
		if err != nil {
			wr = fmt.Sprintf("%s (ERROR: %s)", wr, err.Error())
			goto end
		}
		if wd == "" {
			wr = fmt.Sprintf("%s (ERROR: Working directory unavailable)", wr)
			goto end
		}
		dp = dt.DirPathJoin(wd, svr.API.Webroot)
	}
	wr = localsvr.HomeRelative(string(dp))
end:
	return wr
}

func (svr *Server) displayAPISourceFile() string {
	return localsvr.HomeRelative(string(svr.API.SourceFile))
}
func (svr *Server) displayDBSourceFile() string {
	return localsvr.HomeRelative(string(svr.Database.SourceFile()))
}
func (svr *Server) displayServerSourceFile() string {
	return localsvr.HomeRelative(string(svr.SourceFile))
}

func (svr *Server) displayDir() (d string) {
	// Print current working directory
	wd, err := os.Getwd()
	if err != nil {
		d = err.Error()
		goto end
	}
	if wd == "" {
		d = "Working directory unavailable"
		goto end
	}
	d = localsvr.HomeRelative(wd)
end:
	return d
}

func (svr *Server) displayDBName() (name string) {
	if svr.Database == nil {
		return ""
	}
	return svr.Database.String()
}
func (svr *Server) displayDBTypeName() (name string) {
	return svr.Database.TypeName()
}

func (svr *Server) displayExtensionPaths() (paths string) {
	var sb strings.Builder
	for _, ext := range svr.Database.Extensions() {
		sb.WriteString(ext.Name())
		sb.WriteString(". ")
	}
	paths = sb.String()
	if len(paths) != 0 {
		paths = paths[:len(paths)-2]
	}
	return paths
}
