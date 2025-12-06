// Package apipkg provides the API endpoint management system for xmlui-localsvr.
//
// This package handles the configuration-driven API endpoint system that allows
// defining HTTP endpoints through JSON configuration files. It supports:
//
//   - Dynamic endpoint routing based on URL patterns
//   - Path parameter extraction and validation
//   - SQL query execution with parameter binding
//   - JSON request/response handling
//   - File-based or inline SQL queries
//   - Type checking and constraint validation
//
// # Configuration Format
//
// API endpoints are defined in JSON configuration files with this structure:
//
//	{
//	  "name": "My API",
//	  "basePath": "/api/v1",
//	  "webroot": "./public",
//	  "endpoints": [
//	    {
//	      "endpoint": "GET /users/:id",
//	      "query": "SELECT * FROM users WHERE id = :id",
//	      "params": ["id:integer"]
//	    }
//	  ]
//	}
//
// # Usage Example
//
//	cfg := &cfgldr.APIConfigV2{...}
//	api, err := apipkg.CreateAPI(apipkg.CreateAPIArgs{
//		APIConfig: cfg,
//		Writer: writer,
//		Logger: logger,
//	})
//	if err != nil {
//		return err
//	}
//	err = api.Initialize(ctx)
//	handler := api.HandleAPIFunc(ctx, database)
package apipkg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-pathvars"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// API represents a configured API instance with endpoints, routing, and metadata.
// It manages HTTP endpoints that execute SQL queries based on JSON configuration.
type API struct {
	Name                 string           // Human-readable name for the API
	Webroot              dt.DirPath       // Root directory for static file serving // TODO: Move this to Server 🤦‍♂️
	SourceFile           dt.Filepath      // Path to the configuration file
	BasePath             localsvr.URLPath // Common URL prefix for all endpoints
	Endpoints            []*Endpoint      // List of configured API endpoints
	Verbose              bool             // Enable verbose logging
	Router               *pathvars.Router // URL routing and path parameter extraction
	Options              *localsvr.Options
	initialized          bool // Whether Initialize() has been called
	cliutil.WriterLogger      // Embedded logging functionality
}

// APIArgs contains the configuration needed to create a new API instance.
type APIArgs struct {
	Name       string           // API name
	Webroot    dt.DirPath       // Static file root directory
	SourceFile dt.Filepath      // Configuration file path
	BasePath   localsvr.URLPath // URL prefix for endpoints
	Endpoints  []*Endpoint      // Parsed endpoint configurations
	Options    *localsvr.Options
	CLIWriter  cliutil.Writer // CLI output writer
	Logger     *slog.Logger   // Structured logger
}

// CreateAPIArgs contains dependencies needed to create an API from configuration.
type CreateAPIArgs struct {
	Database  dbpkg.Database
	APIConfig cfgldr.APIConfig // Loaded API configuration
	Options   *localsvr.Options
	Writer    cliutil.Writer // CLI writer for output
	Logger    *slog.Logger   // Logger instance
}

// CreateAPI creates a new API instance from the provided configuration.
// It parses the configuration, validates settings, and creates endpoints.
// Currently only supports APIConfigV2 format.
func CreateAPI(args CreateAPIArgs) (api *API, err error) {
	var basePath localsvr.URLPath
	var sourceFile dt.Filepath
	var webroot dt.DirPath
	var endpoints []*Endpoint

	cfg := args.APIConfig

	cfgV2, ok := cfg.(*cfgldr.APIConfigV2)
	if !ok {
		panic(fmt.Sprintf("Cannot type assert APIConfig config value of type %T to type %T", cfg, (*cfgldr.APIConfigV2)(nil)))
	}
	basePath, err = localsvr.ParseURLPath(cfgV2.BasePath)
	if err != nil {
		goto end
	}
	sourceFile, err = dt.ParseFilepath(cfgV2.SourceFile)
	if err != nil {
		goto end
	}
	endpoints, err = ParseEndpoints(cfgV2.Endpoints, basePath, args.Database)
	if err != nil {
		goto end
	}
	webroot, err = localsvr.ParseDirPath(cfgV2.Webroot)
	if err != nil {
		goto end
	}

	api = NewAPI(APIArgs{
		Name:       cfgV2.Name,
		Webroot:    webroot,
		SourceFile: sourceFile,
		BasePath:   basePath,
		Endpoints:  endpoints,
		Options:    args.Options,
		CLIWriter:  args.Writer,
		Logger:     args.Logger,
	})
end:
	return api, err
}

// NewAPI creates a new API instance with the provided arguments.
// The API is created with an empty router that must be initialized
// before use by calling Initialize().
func NewAPI(args APIArgs) (api *API) {
	return &API{
		Name:         args.Name,
		Webroot:      args.Webroot,
		SourceFile:   args.SourceFile,
		BasePath:     args.BasePath,
		Endpoints:    args.Endpoints,
		Options:      args.Options,
		Router:       pathvars.NewRouter(),
		WriterLogger: cliutil.NewWriterLogger(args.CLIWriter, args.Logger),
	}
}

// Initialize prepares the API for use by setting up the routing system.
// This method must be called before using HandleAPIFunc().
// It parses all endpoint path patterns and builds the internal router.
func (api *API) Initialize(_ context.Context) (err error) {
	if api.initialized {
		goto end
	}
	err = api.initializeRouter()
	api.initialized = true
end:
	return err
}

// initializeRouter configures the internal router with all endpoint patterns.
// It parses path variables from each endpoint and registers them with the router.
func (api *API) initializeRouter() (err error) {
	var errs []error
	for i, ep := range api.Endpoints {
		var pp []pathvars.Parameter
		pp, err = ep.ParsePathVarParameters()
		if err != nil {
			errs = append(errs, err)
		}
		err = api.Router.AddRoute(pathvars.HTTPMethod(ep.method), ep.path, &pathvars.RouteArgs{
			Parameters:  pp,
			Index:       i,
			Description: ep.Description,
			Cardinality: pathvars.Cardinality(ep.Cardinality),
			RowType:     pathvars.DBRowType(ep.RowType),
			ColumnTypes: pathvars.DBDataTypes(ep.ColumnTypes),
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		err = CombineErrs(errs)
	}
	return err
}

//// extractBodyJSON extracts JSON from the request body into a localsvr.JSONBytes
//// variable. It uses a TeeReader to preserve the request body for potential
//// future use. On error it returns nil result and a populated error.
//func extractBodyJSON(r *http.Request) (json localsvr.JSONBytes, err error) {
//	var buffer bytes.Buffer
//	var jsonBytes []byte
//	if r.Body == nil {
//		goto end
//	}
//
//	jsonBytes, err = io.ReadAll(io.TeeReader(r.Body, &buffer))
//	if err != nil {
//		goto end
//	}
//
//	// Reset r.Body for potential future use
//	r.Body = io.NopCloser(&buffer)
//	json = jsonBytes
//
//end:
//	return json, err
//}
