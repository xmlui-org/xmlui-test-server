package xmluisvr

import (
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

var appInfo = appinfo.New(appinfo.Args{
	Name:        localsvr.AppName,
	Description: localsvr.AppDescr,
	Version:     localsvr.AppVer,
	AppSlug:     localsvr.AppSlug,
	ConfigSlug:  localsvr.ConfigSlug,
	ConfigFile:  localsvr.ConfigFile,
	InfoURL:     localsvr.InfoURL,
	ExeName:     localsvr.ExeName,
	LogFile:     localsvr.LogFile,
	LogPath:     localsvr.LogPath,
	ExtraInfo:   localsvr.ExtraInfo,
})

func AppInfo() appinfo.AppInfo {
	return appInfo
}
