package test

import (
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/stretchr/testify/require"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// TestCfgStoreMerge tests the cfgstore merging behavior to diagnose why endpoints disappear
func TestCfgStoreMerge(t *testing.T) {
	wd, _ := dt.Getwd()
	rootFix, css := SetupFixtures(t, SetupFixturesArgs{
		TestDataDir: dt.DirPathJoin(wd, testDataDir),
		UserFile:    "./user-config.localsvr.json",
		ProjectFile: "./project-config.localsvr.json",
	})
	defer rootFix.Cleanup()

	// First, verify that the individual config stores can load their files correctly
	t.Run("verify_individual_stores_load_correctly", func(t *testing.T) {
		// Load from CLI (user) store
		cliStore := css.CLIConfigStore()
		require.True(t, cliStore.Exists(), "CLI config store should exist")

		var cliConfig cfgldr.RootConfigV1
		err := cliStore.LoadJSON(&cliConfig, nil)
		require.NoError(t, err, "Should load CLI config")

		cliAPI := cliConfig.APIConfig()
		require.NotNil(t, cliAPI, "CLI API config should not be nil")
		cliAPIV2 := cliAPI.(*cfgldr.APIConfigV2)
		// t.Logf("CLI config has %d endpoints", len(cliAPIV2.Endpoints))
		require.Len(t, cliAPIV2.Endpoints, 5, "CLI config should have 5 endpoints")

		// Load from Project store
		projectStore := css.ProjectConfigStore()
		require.True(t, projectStore.Exists(), "Project config store should exist")

		var projectConfig cfgldr.RootConfigV1
		err = projectStore.LoadJSON(&projectConfig, nil)
		require.NoError(t, err, "Should load project config")

		projectAPI := projectConfig.APIConfig()
		require.NotNil(t, projectAPI, "Project API config should not be nil")
		projectAPIV2 := projectAPI.(*cfgldr.APIConfigV2)
		// t.Logf("Project config has %d endpoints", len(projectAPIV2.Endpoints))
		require.Len(t, projectAPIV2.Endpoints, 5, "Project config should have 5 endpoints")
	})

	// Now test the merged result
	t.Run("verify_merged_config_has_endpoints", func(t *testing.T) {
		opts := cfgldr.NewOptions(cfgldr.OptionsArgs{})

		gotRc, err := cfgldr.LoadRootConfigV1(cfgldr.LoadRootConfigV1Args{
			Options:      opts,
			ConfigStores: css,
		})
		require.NoError(t, err, "LoadRootConfigV1 should not error")

		api := gotRc.APIConfig()
		require.NotNil(t, api, "Merged API config should not be nil")

		apiV2 := api.(*cfgldr.APIConfigV2)
		// t.Logf("Merged config has %d endpoints", len(apiV2.Endpoints))
		// t.Logf("Merged API Name: %s", apiV2.Name)
		// t.Logf("Merged Webroot: %s", apiV2.Webroot)

		// CRITICAL: This is the bug - merged config has 0 endpoints!
		// Both individual configs have 5 endpoints each, but the merge loses them
		require.Len(t, apiV2.Endpoints, 5, "Merged config should have 5 endpoints from merging")
	})
}
