package test

import (
	"testing"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cfgstore/cstest"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-fsfix"
	"github.com/mikeschinkel/go-testutil"
)

type SetupFixturesArgs struct {
	TestDataDir dt.DirPath
	UserFile    dt.Filename
	ProjectFile dt.Filename
}

// SetupFixtures sets up a root fixture with two dir fixtures, one to
// emulate the user's ~/.config/appname config directory and the other to emulate
// the project's ./.appname config directory. The userFile and projectFile should
// be absolute file paths containing the respective config files for each.
func SetupFixtures(t *testing.T, args SetupFixturesArgs) (rootFix *fsfix.RootFixture, css *cfgstore.ConfigStores) {

	rootFix = fsfix.NewRootFixture("cfgldr")

	testArgs := &cstest.TestDirsProviderArgs{
		Username:   "alanturing",
		ProjectDir: "myproject",
		ConfigSlug: "appname",
		TestRootFunc: func() dt.DirPath {
			return rootFix.Dir()
		},
	}

	// Create config stores map with default configuration
	css = cfgstore.NewConfigStores(cfgstore.ConfigStoresArgs{
		ConfigStoreArgs: cfgstore.ConfigStoreArgs{
			ConfigSlug:   "appname",
			RelFilepath:  "cfgldr.json",
			DirsProvider: cstest.NewTestDirsProvider(testArgs),
		},
	})

	// Get config file name from the config store

	cliStore := css.CLIConfigStore()
	cliDir, err := cstest.GetRelConfigDir(cliStore, testArgs)
	if err != nil {
		t.Fatal(err)
	}
	cliConfig := cliStore.GetRelFilepath()
	// Create .config directory for user config
	cliFix := rootFix.AddDirFixture(t, cliDir, nil)
	cliFix.AddFileFixture(t, cliConfig, &fsfix.FileFixtureArgs{
		Content: string(testutil.LoadFile(t, dt.FilepathJoin(testDataDir, args.UserFile), true)),
	})

	projectStore := css.ProjectConfigStore()
	projectDir, err := cstest.GetRelConfigDir(projectStore, testArgs)
	if err != nil {
		t.Fatal(err)
	}
	projectConfig := projectStore.GetRelFilepath()
	// Create project directory for project config (same content for simplicity)

	// Setup project config directory fixture
	projectFix := rootFix.AddDirFixture(t, projectDir, nil)
	projectFix.AddFileFixture(t, projectConfig, &fsfix.FileFixtureArgs{
		Content: string(testutil.LoadFile(t, dt.FilepathJoin(testDataDir, args.ProjectFile), true)),
	})

	rootFix.Create(t)

	css.CLIConfigStore().SetConfigDir(cliFix.Dir())
	css.ProjectConfigStore().SetConfigDir(projectFix.Dir())

	return rootFix, css
}
