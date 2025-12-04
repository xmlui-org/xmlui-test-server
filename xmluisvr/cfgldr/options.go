package cfgldr

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

const (
	DefaultTimeout  = 3
	DefaultHTTPPort = 8080
	DefaultAPIFile  = ""
	DefaultDBPort   = 0

	DefaultQuiet                 = false
	DefaultAllowUntrustedQueries = false
	DefaultVerbosity             = cliutil.DefaultVerbosity
	DefaultDBAccessMode          = int(sqlparams.DBReadWriteMode)
)

const (
	DefaultErrorStyle = string(common.DefaultErrorStyle)
)

const (
	AllowUntrustedQueriesFlag = "dangerously-allow-untrusted-db-queries"
)

var (
	DefaultConnectString = common.DefaultSQLite3Database
)

type Options struct {
	Timeout               int
	HTTPPort              int
	APIFile               string
	Webroot               string
	ConnectString         string
	DBPort                int
	DBBootstrapFile       string
	Quiet                 bool
	Verbosity             int
	ErrorStyle            string
	AllowUntrustedQueries bool
	DBAccessMode          int
	DBExtensionFiles      []string
}

func (*Options) Options() {}
func (opts *Options) appendToWebroot(value, defaultValue string) string {
	switch {
	case value == "":
		value = filepath.Join(opts.Webroot, defaultValue)
	case !filepath.IsAbs(value):
		value = filepath.Join(opts.Webroot, value)
	}
	return value
}

type OptionsArgs struct {
	Timeout               *int
	HTTPPort              *int
	APIFile               *string
	Webroot               *string
	ConnectString         *string
	DBPort                *int
	DBBootstrapFile       *string
	Quiet                 *bool
	Verbosity             *int
	ErrorStyle            *string
	DBAccessMode          *int
	AllowUntrustedQueries *bool
	DBExtensionFiles      []string
}

func NewOptions(args OptionsArgs) *Options {
	opts := &Options{}

	if args.Timeout != nil {
		opts.Timeout = *args.Timeout
	}
	if args.HTTPPort != nil {
		opts.HTTPPort = *args.HTTPPort
	}
	if args.APIFile != nil {
		opts.APIFile = *args.APIFile
	}
	if args.Webroot != nil {
		opts.Webroot = *args.Webroot
	} else {
		opts.Webroot = common.DefaultWebroot
	}
	if len(opts.Webroot) == 0 {
		print()
	}
	if len(opts.Webroot) >= 1 && opts.Webroot[0] == '.' {
		opts.Webroot = string(dt.DirPathJoin(workingDir, opts.Webroot))
	}
	if args.ConnectString != nil {
		opts.ConnectString = *args.ConnectString
	}
	if args.DBPort != nil {
		opts.DBPort = *args.DBPort
	}
	if args.DBBootstrapFile != nil {
		opts.DBBootstrapFile = *args.DBBootstrapFile
	}
	if args.Quiet != nil {
		opts.Quiet = *args.Quiet
	}
	if args.Verbosity != nil {
		opts.Verbosity = *args.Verbosity
	}
	if args.ErrorStyle != nil {
		opts.ErrorStyle = *args.ErrorStyle
	}
	if args.DBAccessMode != nil {
		opts.DBAccessMode = *args.DBAccessMode
	}
	if args.DBExtensionFiles != nil {
		opts.DBExtensionFiles = args.DBExtensionFiles
	}
	return opts
}

// OptionsFlagSet encapsulates flag parsing for server options
type OptionsFlagSet struct {
	fs                    *flag.FlagSet
	timeout               *int
	port                  *int
	apiFile               *string
	webroot               *string
	connStr               *string
	dbPort                *int
	dbBootstrapFile       *string
	dbExtensions          stringSliceFlag
	quiet                 *bool
	verbosity             *int
	errorStyle            *string
	dbAccessMode          *int
	allowUntrustedQueries *bool
}

// NewOptionsFlagSet creates a new OptionsFlagSet with all flags configured
func NewOptionsFlagSet(name string) *OptionsFlagSet {
	ofs := &OptionsFlagSet{
		fs:                    flag.NewFlagSet(name, flag.ContinueOnError),
		timeout:               new(int),
		port:                  new(int),
		apiFile:               new(string),
		webroot:               new(string),
		connStr:               new(string),
		dbPort:                new(int),
		dbBootstrapFile:       new(string),
		dbExtensions:          stringSliceFlag{},
		quiet:                 new(bool),
		verbosity:             new(int),
		errorStyle:            new(string),
		dbAccessMode:          new(int),
		allowUntrustedQueries: new(bool),
	}

	// Set up command line flags with long and short versions
	ofs.fs.IntVar(ofs.port, "port", DefaultHTTPPort, "port to run the server on")
	ofs.fs.IntVar(ofs.port, "p", DefaultHTTPPort, "port to run the server on (shorthand)")

	ofs.fs.IntVar(ofs.timeout, "timeout", DefaultTimeout, "Timeout(in seconds) (TODO explain what this controls)")

	ofs.fs.StringVar(ofs.webroot, "webroot", common.DefaultWebroot, "Directory to serve from")
	ofs.fs.StringVar(ofs.apiFile, "api", DefaultAPIFile, "Path to API description file")
	ofs.fs.StringVar(ofs.connStr, "db", DefaultConnectString, "Path to SQLite database file or PostgreSQL connection string or DB description file")
	ofs.fs.StringVar(ofs.dbBootstrapFile, "db-bootstrap", common.DefaultDBBootstrapFilepath,
		fmt.Sprintf("Path to database query file containing idempotent queries to run on start of server (default %s)", common.DefaultDBBootstrapFilepath),
	)
	ofs.fs.IntVar(ofs.dbPort, "db-port", 0, "PostgreSQL port (optional, overrides port in --db if provided)")
	ofs.fs.Var(&ofs.dbExtensions, "db-ext", "One or more paths to database extensions to load (currently only SQLite3.)")
	ofs.fs.IntVar(ofs.dbAccessMode, "db-access", DefaultDBAccessMode, "Mode for API access the database (1=Read-only,2=Read-Write,3=Admin,4=SuperAdmin, default 2)")

	ofs.fs.BoolVar(ofs.quiet, "quiet", DefaultQuiet, "Disable display of most command line output")
	ofs.fs.BoolVar(ofs.quiet, "q", DefaultQuiet, "Disable display of most command line output (shorthand)")
	ofs.fs.BoolVar(ofs.allowUntrustedQueries, AllowUntrustedQueriesFlag, false, "Allow UNTRUSTED Database Queries to be submitted via the API")

	ofs.fs.IntVar(ofs.verbosity, "verbosity", DefaultVerbosity, "Verbosity of most command line output (1 to 3, default 1)")
	ofs.fs.IntVar(ofs.verbosity, "v", DefaultVerbosity, "Verbosity of most command line output (shorthand, 1 to 3, default 1)")

	ofs.fs.StringVar(ofs.errorStyle, "err-style", DefaultErrorStyle, "Errors style can be 'dev' for Developer style, or 'pres' for Presentation style")

	return ofs
}

// FlagSet returns the underlying flag.FlagSet for composition with other CLIs
func (ofs *OptionsFlagSet) FlagSet() *flag.FlagSet {
	return ofs.fs
}

// Parse parses the provided arguments and returns the constructed Options
func (ofs *OptionsFlagSet) Parse(args []string) error {
	return ofs.fs.Parse(args)
}

// Options constructs and returns the Options from parsed flag values
func (ofs *OptionsFlagSet) Options() (opts *Options, err error) {
	var verbosity cliutil.Verbosity

	// Parse Verbosity because it is the only one that gets used immediately that
	// needs to be parsed.
	verbosity, err = cliutil.ParseVerbosity(*ofs.verbosity)
	if err != nil {
		goto end
	}

	opts = NewOptions(OptionsArgs{
		HTTPPort:              ofs.port,
		APIFile:               ofs.apiFile,
		Webroot:               ofs.webroot,
		ConnectString:         ofs.connStr,
		DBPort:                ofs.dbPort,
		DBBootstrapFile:       ofs.dbBootstrapFile,
		Quiet:                 ofs.quiet,
		Verbosity:             intPtr(int(verbosity)),
		DBExtensionFiles:      ofs.dbExtensions.values(),
		Timeout:               ofs.timeout,
		ErrorStyle:            ofs.errorStyle,
		AllowUntrustedQueries: ofs.allowUntrustedQueries,
	})

end:
	return opts, err
}

var options *Options

func GetOptions() (opts *Options, err error) {
	var ofs *OptionsFlagSet

	if options != nil {
		opts = options
		goto end
	}

	ofs = NewOptionsFlagSet(os.Args[0])

	// Set custom flag usage to display double dashes for word options
	ofs.fs.Usage = func() {
		fprintf(ofs.fs.Output(), "Usage of %s:\n", os.Args[0])
		ofs.fs.VisitAll(func(f *flag.Flag) {
			prefix := "-"
			// Use double dash for multi-character flags
			if len(f.Name) > 1 {
				prefix = "--"
			}
			fprintf(ofs.fs.Output(), "  %s%s: %s\n", prefix, f.Name, f.Usage)
		})
	}

	err = ofs.Parse(os.Args[1:])
	if err != nil {
		goto end
	}

	opts, err = ofs.Options()
	if err != nil {
		goto end
	}

	options = opts
end:
	return opts, err
}
func intPtr(n int) *int {
	return &n
}

type stringSliceFlag []string

func (f *stringSliceFlag) values() (v []string) {
	v = make([]string, 0, len(*f))
	for _, s := range *f {
		v = append(v, strings.TrimSpace(s))
	}
	return v
}

func (f *stringSliceFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func fprintf(w io.Writer, format string, a ...any) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}
