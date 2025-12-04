package xmluisvr

import (
	"context"
	"log/slog"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/go-logutil"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

// RunArgs contains all the configuration and dependencies needed to run the server.
// This struct is used to pass configuration from the CLI layer to the core server logic.
type RunArgs struct {
	CLIArgs      []string               // Command-line arguments (currently unused)
	AppInfo      appinfo.AppInfo        // Developer-maintained application information
	Config       *Config                // Parsed configuration from files
	Options      *common.Options        // Parsed command-line options
	DirsProvider *cfgstore.DirsProvider // Optional custom directory provider for config loading
}

func ensureDirsProvider(dp *cfgstore.DirsProvider) (_ *cfgstore.DirsProvider) {
	// Determine log file location using DirsProvider
	if dp != nil {
		goto end
	}
	dp = cfgstore.DefaultDirsProvider()
end:
	return dp
}
func getProjectDir(dp *cfgstore.DirsProvider) (projectDir dt.DirPath, err error) {
	// Determine log file location using DirsProvider
	if dp != nil {
		// Use custom project directory (e.g., demo install dir)
		projectDir, err = dp.ProjectDirFunc()
	}
	if err != nil {
		goto end
	}
	if projectDir != "" {
		goto end
	}
	// Use current working directory
	projectDir, err = dt.Getwd()
	if err != nil {
		goto end
	}
end:
	return projectDir, err
}

// ParseRunArgs creates a complete RunArgs from a partial RunArgs and cfgldr.Options.
// This function extracts the RunArgs construction logic from RunCLI
// so it can be reused by commands that need to call Run() directly.
//
// The input args should contain:
//   - AppInfo: Application information
//   - Config.Writer: Writer for output
//   - DirsProvider: (optional) Custom directory provider for config loading
//
// Returns a new RunArgs with:
//   - Config: Complete server configuration including logger
//   - Options: Parsed common options
func ParseRunArgs(ctx context.Context, cfgOpts *cfgldr.Options, args *RunArgs) (runArgs *RunArgs, err error) {
	var cfg *cfgldr.RootConfigV1
	var opts *common.Options
	var config *Config
	var logger *slog.Logger
	var projectDir dt.DirPath
	var logFile dt.Filepath

	writer := args.Config.Writer

	projectDir, err = getProjectDir(args.DirsProvider)
	if err != nil {
		goto end
	}

	args.DirsProvider = ensureDirsProvider(args.DirsProvider)

	// Create log file path: <projectDir>/<logPath>/<logFile>
	// e.g., ~/.config/xmlui/demos/invoice/logs/xmlui-localsvr.log
	logFile = dt.FilepathJoin3(projectDir, args.AppInfo.LogPath(), args.AppInfo.LogFile())

	// Create logger
	logger, err = logutil.CreateJSONFileLogger(logFile)
	if err != nil {
		writer.Printf("Warning: Failed to create log file %s: %v\n", logFile, err)
		writer.Printf("Continuing without writing logs to disk\n")
		// Continue without logger - server will handle nil logger
	}

	// Load root configuration
	cfg, err = cfgldr.LoadRootConfigV1(cfgldr.LoadRootConfigV1Args{
		AppInfo:      args.AppInfo,
		Options:      cfgOpts,
		DirsProvider: args.DirsProvider,
	})
	if err != nil {
		goto end
	}

	// Parse options
	opts, err = ParseOptions(cfgOpts)
	if err != nil {
		goto end
	}

	cfg.ServerConfig.APIConfig.Webroot = cfgOpts.Webroot

	// Parse configuration
	config, err = ParseConfig(ctx, cfg, ParseConfigArgs{
		Options:      opts,
		Logger:       logger,
		Writer:       writer,
		DirsProvider: args.DirsProvider,
	})
	if err != nil {
		goto end
	}

	// Return new RunArgs with complete Config and Options
	runArgs = &RunArgs{
		CLIArgs:      args.CLIArgs,
		AppInfo:      args.AppInfo,
		Config:       config,
		Options:      opts,
		DirsProvider: args.DirsProvider,
	}

end:
	return runArgs, err
}
