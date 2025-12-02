package xmluisvr

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

// RunCLI is the main CLI entry point for the xmlui-localsvr application.
// It handles command-line argument parsing, configuration loading, and starts the server.
// This function sets up logging, loads configuration files, and delegates to Run().
//
// If cfgOpts is nil, it will parse command-line flags itself (standalone mode).
// If cfgOpts is provided, it will use those options (composed mode with xmlui CLI).
//
// Exit codes:
//   - 1: Options parsing failure
//   - 2: Configuration loading failure
//   - 3: Configuration parsing failure
//   - 4: Known runtime error
//   - 5: Unknown runtime error
func RunCLI(cfgOpts *cfgldr.Options) {
	var err error
	var wl cliutil.WriterLogger
	var runArgs *RunArgs

	if cfgOpts == nil {
		cfgOpts, err = cfgldr.GetOptions()
		if err != nil {
			cliutil.Stderrf("Invalid option(s): %v\n", strings.ReplaceAll(err.Error(), "\n", "; "))
			os.Exit(cliutil.ExitOptionsParseError)
		}
	}

	//goland:noinspection GoDfaErrorMayBeNotNil,GoMaybeNil
	writer := cliutil.NewWriter(&cliutil.WriterArgs{
		Quiet:     cfgOpts.Quiet,
		Verbosity: cliutil.Verbosity(cfgOpts.Verbosity),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runArgs = &RunArgs{
		AppInfo: AppInfo(),
		Config: &Config{
			Writer: writer,
		},
	}

	runArgs, err = ParseRunArgs(ctx, cfgOpts, runArgs)
	if err != nil {
		wl = cliutil.NewWriterLogger(writer, nil)
		_ = wl.ErrorError("Failed to parse run arguments", "error", err)
		os.Exit(cliutil.ExitConfigParseError)
	}
	//goland:noinspection GoMaybeNil
	defer dt.CloseOrLog(runArgs.Config.Database)

	common.SetLogger(runArgs.Config.Logger)
	wl = cliutil.NewWriterLogger(writer, runArgs.Config.Logger)

	err = Run(ctx, runArgs)

	switch {
	case err == nil:
		writer.Printf("%s terminated gracefully", common.AppName)
	case errors.Is(err, ErrServerError):
		_ = wl.ErrorError("CLI terminated with error:",
			"cli_name", common.AppName,
			"exe_name", common.ExeName,
			"error", err,
		)
		os.Exit(cliutil.ExitKnownRuntimeError)
	default:
		_ = wl.ErrorError("Server terminated with an unexpected error", err)
		os.Exit(cliutil.ExitUnknownRuntimeError)
	}
}
