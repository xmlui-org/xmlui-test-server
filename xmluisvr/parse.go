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
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

// ParseOptions converts raw options from cfgldr.Options into
// validated common.Options. This method performs validation and type conversion
// for all XMLUI Test Server options.
func ParseOptions(cfgOpts *cfgldr.Options) (opts *common.Options, err error) {
	var errs []error
	var cliOpts *cliutil.CLIOptions

	cliOpts, err = cliutil.NewCLIOptions(cliutil.CLIOptionsArgs{
		Quiet:     &cfgOpts.Quiet,
		Verbosity: &cfgOpts.Verbosity,
	})
	errs = AppendErr(errs, err)

	opts = &common.Options{
		CLIOptions:            cliOpts,
		AllowUntrustedQueries: cfgOpts.AllowUntrustedQueries,
	}
	opts.Timeout, err = common.ParseTimeDurationEx(strconv.Itoa(cfgOpts.Timeout))
	errs = AppendErr(errs, err)
	opts.HTTPPort, err = common.ParseServerPort(cfgOpts.HTTPPort, common.ZeroOk)
	errs = AppendErr(errs, err)
	opts.DBExtensionFiles, err = common.ParseFilepaths(cfgOpts.DBExtensionFiles)
	errs = AppendErr(errs, err)
	opts.APIFile, err = dt.ParseFilepath(cfgOpts.APIFile)
	errs = AppendErr(errs, err)
	opts.ConnectString, err = common.ParseConnectString(cfgOpts.ConnectString)
	errs = AppendErr(errs, err)
	opts.DBPort, err = common.ParseServerPort(cfgOpts.DBPort, common.ZeroOk)
	errs = AppendErr(errs, err)
	opts.DBBootstrapFile, err = dt.ParseFilepath(cfgOpts.DBBootstrapFile)
	errs = AppendErr(errs, err)
	opts.ErrorStyle, err = common.ParseErrorStyle(cfgOpts.ErrorStyle)
	errs = AppendErr(errs, err)
	opts.Webroot, err = dt.ParseDirPath(cfgOpts.Webroot)
	errs = AppendErr(errs, err)

	return opts, CombineErrs(errs)
}

type ParseAPIArgs struct {
	APIConfig cfgldr.APIConfig
	Database  dbpkg.Database
	Options   *common.Options
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
	Options      *common.Options
	DBConfig     cfgldr.DatabaseConfig
	DirsProvider *cfgstore.DirsProvider
	DirType      cfgstore.DirType
}

// ParseDatabase initializes the database connection using the provided configuration and options.
// It sets up the database with the specified connection string and any configured extensions.
func ParseDatabase(ctx Context, args ParseDatabaseArgs) (db dbpkg.Database, err error) {

	if args.Options.ConnectString == "" {
		args.Options.ConnectString = common.ConnectString(common.DefaultSQLite3Database)
	}

	if common.IsNil(args.DBConfig) {
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
	Options      *common.Options     // Parsed and validated options
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

	if common.IsNil(args.ServerConfig) {
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
		args.Writer.Errorf("%s is not a valid host; %s only supports localhost; defaulting to localhost.\n", svrCfg.Host, common.AppName)
	}

	httpPort = int(args.Options.HTTPPort)
	if httpPort == 0 {
		httpPort = svrCfg.Port
	}
	if httpPort == 0 {
		httpPort = common.DefaultServerPort
	}
	args.Options.HTTPPort, err = common.ParseServerPort(httpPort, common.ZeroInvalid)
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
	Options      *common.Options
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
