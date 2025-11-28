package cfgldr_test

import (
	"os"
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/stretchr/testify/require"
)

// TestCfgStoreFileLocation tests where cfgstore is looking for config files
func TestCfgStoreFileLocation(t *testing.T) {
	wd, _ := dt.Getwd()
	rootFix, css := SetupFixtures(t, SetupFixturesArgs{
		TestDataDir: dt.DirPathJoin(wd, testDataDir),
		UserFile:    "./user-config.localsvr.json",
		ProjectFile: "./project-config.localsvr.json",
	})
	defer rootFix.Cleanup()

	// Check where fixtures are created
	// t.Logf("Test root dir: %s", rootFix.Dir())

	// Check CLI store
	cliStore := css.CLIConfigStore()
	// t.Logf("CLI store exists: %v", cliStore.Exists())

	cliPath, err := cliStore.GetFilepath()
	require.NoError(t, err)
	// t.Logf("CLI store path: %s", cliPath)

	// Check if file actually exists
	_, statErr := os.Stat(string(cliPath))
	// t.Logf("CLI file exists via os.Stat: %v (error: %v)", statErr == nil, statErr)

	if statErr == nil {
		//data, readErr := os.ReadFile(string(cliPath))
		_, readErr := os.ReadFile(string(cliPath))
		require.NoError(t, readErr)
		// t.Logf("CLI file size: %d bytes", len(data))
		// t.Logf("CLI file first 200 chars: %s...", string(data[:min(200, len(data))]))
	}

	// Check project store
	projectStore := css.ProjectConfigStore()
	// t.Logf("Project store exists: %v", projectStore.Exists())

	projectPath, err := projectStore.GetFilepath()
	require.NoError(t, err)
	// t.Logf("Project store path: %s", projectPath)

	// Check if file actually exists
	_, statErr = os.Stat(string(projectPath))
	// t.Logf("Project file exists via os.Stat: %v (error: %v)", statErr == nil, statErr)

	if statErr == nil {
		//data, readErr := os.ReadFile(string(projectPath))
		_, readErr := os.ReadFile(string(projectPath))
		require.NoError(t, readErr)
		// t.Logf("Project file size: %d bytes", len(data))
	}
}
