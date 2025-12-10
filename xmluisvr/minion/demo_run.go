package minion

import (
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// RunDemoArgs contains all arguments for running a demo
type RunDemoArgs struct {
	SourceArg      string         // User's source argument (repo/URL)
	BranchArg      string         // User's branch argument
	Reinstall      bool           // Force reinstall
	SkipValidation bool           // Skip validation
	DryRun         bool           // Dry run mode
	Writer         cliutil.Writer // For output messages
}

// DemoResult contains everything needed to run the demo
type DemoResult struct {
	InstallDir dt.DirPath  // Installation directory
	Webroot    dt.DirPath  // Webroot directory
	ConfigFile dt.Filepath // Config file path
	SiteName   string      // Site name for display
	Source     *DemoSource // Resolved source
}

// RunDemo resolves, installs, and validates a demo in one cohesive operation.
// Handles all config setup internally - no boilerplate needed from caller.
func RunDemo(args *RunDemoArgs) (result *DemoResult, err error) {
	var cs cfgstore.ConfigStore
	var configDir dt.DirPath
	var demoSource *DemoSource
	var webroot dt.DirPath
	var installResult *InstallResult

	// 1. Config setup (moved from CLI)
	cs = cfgstore.NewCLIConfigStore(localsvr.ConfigSlug, localsvr.ConfigFile)
	configDir, err = cs.ConfigDir()
	if err != nil {
		goto end
	}

	err = cs.EnsureDirs([]dt.PathSegment{localsvr.DemosPath})
	if err != nil {
		goto end
	}

	// 2. Resolve demo source
	demoSource, err = ResolveDemoSource(&ResolveDemoSourceArgs{
		SourceArg: args.SourceArg,
		BranchArg: args.BranchArg,
		ConfigDir: configDir,
		Reinstall: args.Reinstall,
		DryRun:    args.DryRun,
		Writer:    args.Writer,
	})
	if err != nil {
		goto end
	}
	if demoSource == nil {
		goto end
	}

	// 3. Find valid branch (if GitHub)
	if demoSource.Type == GitHubSourceType {
		branches := []string{string(demoSource.Ref)}
		if demoSource.Ref == "" {
			branches = []string{"main", "master", "develop"}
		}
		demoSource, err = FindValidBranch(demoSource, configDir, branches, args.Writer)
		if err != nil {
			goto end
		}
	}

	args.Writer.V2().Printf("Demo source: %s\n", FormatDemoSource(demoSource))

	// 4. Install demo
	installResult, err = InstallDemo(&InstallDemoArgs{
		Source:    demoSource,
		ConfigDir: configDir,
		Reinstall: args.Reinstall,
		DryRun:    args.DryRun,
		Writer:    args.Writer,
	})
	if err != nil && err.Error() == fmt.Sprintf("demo already installed at %s", demoSource.InstallDir) {
		// Demo already installed - provide user notification with ErrOmitUserNotify pattern
		webroot, err = DetectWebroot(demoSource.InstallDir)
		if err != nil {
			goto end
		}
		installResult = &InstallResult{
			InstallDir: demoSource.InstallDir,
			ConfigFile: dt.FilepathJoin3(demoSource.InstallDir, ".xmlui", "localsvr.json"),
			SiteName:   string(demoSource.Repo),
		}
		args.Writer.V2().Printf("Demo path: %s\n", installResult.InstallDir)
		args.Writer.Errorf("Demo already installed. Use --reinstall to re-download.\n")
		result = &DemoResult{
			InstallDir: installResult.InstallDir,
			Webroot:    webroot,
			ConfigFile: installResult.ConfigFile,
			SiteName:   installResult.SiteName,
			Source:     demoSource,
		}
		err = cliutil.ErrOmitUserNotify
		goto end
	} else if err != nil {
		goto end
	}

	// 5. Detect webroot
	webroot, err = DetectWebroot(installResult.InstallDir)
	if err != nil {
		goto end
	}

	// 6. Validate demo (unless skipped)
	if !args.SkipValidation {
		args.Writer.V2().Printf("Validating demo...\n")
		err = ValidateDemo(webroot, XMLUIMarkers{
			UMDExport:      localsvr.XMLUIMarkerUMDExport,
			CSSProps:       localsvr.XMLUIMarkerCSSProps,
			MarkupError:    localsvr.XMLUIMarkerMarkupError,
			FunctionLabel:  localsvr.XMLUIMarkerFunctionLabel,
			Version:        localsvr.XMLUIMarkerVersion,
			MinSize:        localsvr.XMLUIBundleMinSize,
			MinMarkers:     localsvr.XMLUIBundleMinMarkers,
			MarkerReadSize: localsvr.XMLUIBundleMarkerReadSize,
		}, args.Writer)
		if err != nil {
			goto end
		}
		args.Writer.V2().Printf("✓ Demo validated successfully\n")
	}

	// 7. Return consolidated result
	result = &DemoResult{
		InstallDir: installResult.InstallDir,
		Webroot:    webroot,
		ConfigFile: installResult.ConfigFile,
		SiteName:   installResult.SiteName,
		Source:     demoSource,
	}

end:
	return result, err
}
