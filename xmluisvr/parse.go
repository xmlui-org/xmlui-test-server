package xmluisvr

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apipkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// ParseOptions converts raw options from cfgldr.Options into
// validated localsvr.Options. This method performs validation and type conversion
// for all XMLUI Test Server options.
func ParseOptions(cfgOpts *cfgldr.Options) (opts *localsvr.Options, err error) {
	var errs []error
	var globalOpts *cliutil.GlobalOptions

	globalOpts, err = cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
		Quiet:     &cfgOpts.Quiet,
		Verbosity: &cfgOpts.Verbosity,
	})
	errs = AppendErr(errs, err)

	opts = &localsvr.Options{
		GlobalOptions:         globalOpts,
		AllowUntrustedQueries: cfgOpts.AllowUntrustedQueries,
	}
	opts.Timeout, err = localsvr.ParseTimeDurationEx(strconv.Itoa(cfgOpts.Timeout))
	errs = AppendErr(errs, err)
	opts.HTTPPort, err = localsvr.ParseServerPort(cfgOpts.HTTPPort, localsvr.ZeroOk)
	errs = AppendErr(errs, err)
	opts.DBExtensionFiles, err = localsvr.ParseFilepaths(cfgOpts.DBExtensionFiles)
	errs = AppendErr(errs, err)
	opts.APIFile, err = dt.ParseFilepath(cfgOpts.APIFile)
	errs = AppendErr(errs, err)
	opts.ConnectString, err = localsvr.ParseConnectString(cfgOpts.ConnectString)
	errs = AppendErr(errs, err)
	opts.DBPort, err = localsvr.ParseServerPort(cfgOpts.DBPort, localsvr.ZeroOk)
	errs = AppendErr(errs, err)
	opts.DBBootstrapFile, err = dt.ParseFilepath(cfgOpts.DBBootstrapFile)
	errs = AppendErr(errs, err)
	opts.ErrorStyle, err = localsvr.ParseErrorStyle(cfgOpts.ErrorStyle)
	errs = AppendErr(errs, err)
	opts.Webroot, err = dt.ParseDirPath(cfgOpts.Webroot)
	errs = AppendErr(errs, err)

	return opts, CombineErrs(errs)
}

type ParseAPIArgs struct {
	APIConfig cfgldr.APIConfig
	Database  dbpkg.Database
	Options   *localsvr.Options
	Writer    cliutil.Writer
	Logger    *slog.Logger
}

// ParseAPI loads and creates the API configuration from either a file or the root ServerConfig.
// If no API file is specified or found, it falls back to the API configuration
// embedded in the root configuration.
func ParseAPI(args ParseAPIArgs) (api *apipkg.API, err error) {
	var apiCfg cfgldr.APIConfig

	apiCfg, err = cfgldr.LoadAPIFileIfExists(args.Options.APIFile)
	if err != nil {
		goto end
	}
	if apiCfg.IsNil() {
		apiCfg = args.APIConfig
	}
	if apiCfg.IsNil() {
		err = NewErr(
			ErrNoConfigProvided,
			"config_type", "api",
		)
		goto end
	}
	api, err = apipkg.CreateAPI(apipkg.CreateAPIArgs{
		APIConfig: apiCfg,
		Database:  args.Database,
		Options:   args.Options,
		Writer:    args.Writer,
		Logger:    args.Logger,
	})
end:
	if err != nil {
		err = WithErr(err,
			ErrConfigParsingFailed,
		)
	}
	return api, err
}

type ParseDatabaseArgs struct {
	Writer       cliutil.Writer
	Logger       *slog.Logger
	Options      *localsvr.Options
	DBConfig     cfgldr.DatabaseConfig
	DirsProvider *cfgstore.DirsProvider
	DirType      cfgstore.DirType
}

// ParseDatabase initializes the database connection using the provided configuration and options.
// It sets up the database with the specified connection string and any configured extensions.
func ParseDatabase(ctx Context, args ParseDatabaseArgs) (db dbpkg.Database, err error) {

	if args.Options.ConnectString == "" {
		args.Options.ConnectString = localsvr.ConnectString(localsvr.DefaultSQLite3Database)
	}

	if localsvr.IsNil(args.DBConfig) {
		err = NewErr(
			ErrNoConfigProvided,
			"config_type", "database",
		)
		goto end
	}

	db, err = dbpkg.ParseDatabase(ctx, args.DBConfig, dbpkg.ParseDatabaseArgs{
		Options:      args.Options,
		Writer:       args.Writer,
		Logger:       args.Logger,
		DirsProvider: args.DirsProvider,
		DirType:      args.DirType,
	})
	if err != nil {
		goto end
	}

end:
	if err != nil {
		err = WithErr(err,
			ErrConfigParsingFailed,
		)
	}
	return db, err
}

// ParseServerArgs contains the dependencies needed to create a Server instance.
type ParseServerArgs struct {
	Database     dbpkg.Database      // Database connection
	Options      *localsvr.Options   // Parsed and validated options
	ServerConfig cfgldr.ServerConfig // Server configuration
	Writer       cliutil.Writer
	Logger       *slog.Logger
}

// ParseServer creates and configures a new Server instance with all dependencies.
// It validates the HTTP port and creates the server with the provided database,
// API configuration, and other settings.
func ParseServer(args ParseServerArgs) (svr *Server, err error) {
	var sourceFile dt.Filepath
	var svrCfg *cfgldr.ServerConfigV1
	var httpPort int

	if localsvr.IsNil(args.ServerConfig) {
		err = NewErr(
			ErrNoConfigProvided,
			"config_type", "server",
		)
		goto end
	}

	svrCfg, err = dtx.AssertType[*cfgldr.ServerConfigV1](args.ServerConfig)
	if err != nil {
		goto end
	}
	switch strings.ToLower(svrCfg.Host) {
	case "localhost", "127.0.0.1":
		// S'all good, man!
	default:
		args.Writer.Errorf("%s is not a valid host; %s only supports localhost; defaulting to localhost.\n", svrCfg.Host, localsvr.AppName)
	}

	httpPort = int(args.Options.HTTPPort)
	if httpPort == 0 {
		httpPort = svrCfg.Port
	}
	if httpPort == 0 {
		httpPort = localsvr.DefaultServerPort
	}
	args.Options.HTTPPort, err = localsvr.ParseServerPort(httpPort, localsvr.ZeroInvalid)
	if err != nil {
		err = NewErr(ErrInvalidServerPort, err)
		goto end
	}

	sourceFile, err = dt.ParseFilepath(svrCfg.SourceFile)

	svr = NewServer(ServerArgs{
		Database:   args.Database,
		Port:       args.Options.HTTPPort,
		SourceFile: sourceFile,
		Options:    args.Options,
		Writer:     args.Writer,
		Logger:     args.Logger,
	})
end:
	if err != nil {
		err = WithErr(err,
			ErrConfigParsingFailed,
		)
	}
	return svr, err
}

type ParseConfigArgs struct {
	Options      *localsvr.Options
	Logger       *slog.Logger
	Writer       cliutil.Writer
	DirsProvider *cfgstore.DirsProvider
}

func ParseConfig(ctx Context, cfg *cfgldr.RootConfigV1, args ParseConfigArgs) (config *Config, err error) {
	config = &Config{
		Logger: args.Logger,
		Writer: args.Writer,
	}

	config.Database, err = ParseDatabase(ctx, ParseDatabaseArgs{
		DBConfig:     cfg.DBConfig,
		Options:      args.Options,
		Writer:       args.Writer,
		Logger:       args.Logger,
		DirsProvider: args.DirsProvider,
		DirType:      cfg.DirType,
	})
	if err != nil {
		err = NewErr(
			ErrFailedToParseDatabaseConfig,
			err,
		)
		goto end
	}

	config.Server, err = ParseServer(ParseServerArgs{
		ServerConfig: cfg.ServerConfig,
		Database:     config.Database,
		Options:      args.Options,
		Writer:       args.Writer,
		Logger:       args.Logger,
	})
	if err != nil {
		err = NewErr(
			ErrFailedToParseServerConfig,
			err,
		)
		goto end
	}

	config.Server.API, err = ParseAPI(ParseAPIArgs{
		APIConfig: cfg.APIConfig(),
		Database:  config.Database,
		Options:   args.Options,
		Writer:    args.Writer,
		Logger:    args.Logger,
	})
	if err != nil {
		err = NewErr(
			ErrFailedToParseAPIConfig,
			err,
		)
		goto end
	}
end:
	if err != nil {
		err = WithErr(err,
			ErrConfigParsingFailed,
		)
	}
	return config, err
}
