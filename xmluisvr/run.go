package xmluisvr

import (
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"

	"github.com/mikeschinkel/go-cliutil"
)

// Run starts the xmlui-localsvr with the provided configuration and context.
// This is the main server execution function that initializes all components
// and starts the HTTP server.
//
// The function follows this execution flow:
//  1. Initialize global settings (writer, logger)
//  2. ParseBytes and validate options
//  3. Initialize database connection
//  4. Load API configuration
//  5. Create and configure server
//  6. Start HTTP server and listen for requests
//
// Returns ErrServerError if the server terminates with an error condition.
func Run(ctx Context, args *RunArgs) (err error) {
	var server *Server

	writer := args.Config.Writer

	writer.Loud().Printf("%s starting\n", args.AppInfo.Name())

	err = Initialize(ctx, args)
	if err != nil {
		goto end
	}

	server = args.Config.Server

	err = server.Initialize(ctx)
	if err != nil {
		goto end
	}

	server.showConfig()

	server.V2().InfoPrint("Starting server")
	err = server.ListenAndServe(ctx)
	if err != nil {
		err = NewErr(ErrServerError, err)
	}
end:
	return err
}

// Initialize sets up global package state including the CLI writer and logger.
// This must be called before other package functions to ensure proper logging
// and output formatting.
func Initialize(_ Context, args *RunArgs) (err error) {
	cfg := args.Config
	// Setting the writer allows the shorthand of being able to call cliutil.Printf()
	// and cliutil.Errorf() without having a writer injected into every func.
	cliutil.SetWriter(cfg.Writer)

	// Setting the logger sets the package level logger variable so it is accessible
	// throughout the package.
	common.SetLogger(cfg.Logger)

	apiresp.SetGitHubRepoURL(common.GitHubRepoURL)

	return err
}
