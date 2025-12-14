package test

import (
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/stretchr/testify/require"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// TestCfgStoreBehavior tests what cfgstore is actually doing
func TestCfgStoreBehavior(t *testing.T) {
	wd, _ := dt.Getwd()
	rootFix, css := SetupFixtures(t, SetupFixturesArgs{
		TestDataDir: dt.DirPathJoin(wd, testDataDir),
		UserFile:    "./user-config.localsvr.json",
		ProjectFile: "./project-config.localsvr.json",
	})
	defer rootFix.Cleanup()

	// Load using LoadRootConfigV1 which is what the actual code uses
	opts := cfgldr.NewOptions(cfgldr.OptionsArgs{})

	// t.Log("=== Before LoadRootConfigV1 ===")

	gotRc, err := cfgldr.LoadRootConfigV1(cfgldr.LoadRootConfigV1Args{
		Options:      opts,
		ConfigStores: css,
	})
	require.NoError(t, err, "LoadRootConfigV1 should not error")

	// t.Log("=== After LoadRootConfigV1 ===")

	// Check what we got
	require.NotNil(t, gotRc, "RootConfig should not be nil")
	require.NotNil(t, gotRc.ServerConfig, "ServerConfig should not be nil")

	// t.Logf("ServerConfig.Host: %s", gotRc.ServerConfig.Host)
	// t.Logf("ServerConfig.Port: %d", gotRc.ServerConfig.Port)

	if gotRc.ServerConfig.APIConfig == nil {
		t.Fatal("APIConfig is nil! This is the problem!")
	}

	api := gotRc.ServerConfig.APIConfig
	// t.Logf("APIConfig.Name: %s", api.Name)
	// t.Logf("APIConfig.Webroot: %s", api.Webroot)
	// t.Logf("APIConfig.Endpoints: %d", len(api.Endpoints))

	// The bug: endpoints should be 5 but are 0
	require.Len(t, api.Endpoints, 5, "Should have 5 endpoints")
}
