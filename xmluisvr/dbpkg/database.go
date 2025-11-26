package dbpkg

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-doterr"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/pathvars"

	. "github.com/mikeschinkel/go-doterr"
)

//type ConnectStyle string
//const (
//	FileConnect ConnectStyle = "file"
//	URLConnect ConnectStyle = "url"
//	DSNConnect ConnectStyle = "dsn"
//	URLOrDSNConnect ConnectStyle = "url|dsn"
//)
//ConnectStyle() ConnectStyle

type DBExtensionConfig interface {
	DBExtensionConfig()
}

type FormatParamFunc = func(int) string

type Database interface {
	Type() DatabaseType
	TypeName() string
	ConnectString() string
	SetConnectString(string)
	SourceFile() dt.Filepath
	SetBaseDatabase(db *BaseDatabase)
	Open(Context) error
	Close() error
	Query(Context, string, ...any) (*sql.Rows, error)
	ValidatedConnection(Context, DatabaseType, common.ConnectString) error
	ParseConnectString(string) (common.ConnectString, error)
	ParseQueryString(query string) (sqlparams.QueryString, error)
	QueryFileExt() string
	ParseExtension(DBExtensionConfig) (DBExtension, error)
	CreateNew(DatabaseArgs) (Database, error)
	Extensions() []DBExtension
	LoadExtension(DBExtension) error
	GetFormatParamFunc() FormatParamFunc
	Options() common.Options
	ConvertValue(value any, dt sqlparams.DBDataType) any
	fmt.Stringer
}

type DatabaseArgs struct {
	DatabaseType     DatabaseType
	ConnectString    string
	Port             int
	Extensions       []DBExtension
	BootstrapQueries *MultipartQuery
	OnOpenQueries    *MultipartQuery
	Options          *common.Options
	AccessMode       AccessMode
	SourceFile       dt.Filepath
	CLIWriter        CLIWriter
	Logger           *slog.Logger
	Config           cfgldr.DatabaseConfig
}

func (args *DatabaseArgs) Normalize() (err error) {
	var entry dt.EntryPath
	var errs []error

	opts := args.Options
	entry = dt.EntryPath(args.ConnectString)
	errs = AppendErr(errs, args.absolutize(&entry, opts.Webroot))
	args.ConnectString = string(entry)
	//entry = dt.EntryPath(args..DBBootstrapFile)
	//errs = AppendErr(errs, args.absolutize(&entry, opts.Webroot))
	//opts.DBBootstrapFile = dt.Filepath(entry)

	return CombineErrs(errs)
}

func (args *DatabaseArgs) absolutize(ep *dt.EntryPath, base dt.DirPath) error {
	abs, err := ep.Abs()
	if err != nil {
		goto end
	}
	if abs != *ep {
		*ep = dt.EntryPathJoin(base, *ep)
	}
end:
	return err
}

type ParseQueriesArgs struct {
	Database       Database
	BaseFilename   string
	ConfigSource   dt.Filepath
	DirsProvider   *cfgstore.DirsProvider
	PrimaryDirType cfgstore.DirType
}

func ParseQueries(queries []string, args ParseQueriesArgs) (mpq *MultipartQuery, err error) {
	var queryBytes []byte
	var fileQuery string
	var elemCnt, lineCnt int
	//var csFilepath dt.Filepath
	var baseDir dt.DirPath

	mpq = NewMultipartQuery()
	elemCnt = len(queries)
	for i, qs := range queries {
		mpq.AddQuerySource(
			NewQuerySource(i+1, i+1, common.QueryString(qs), args.ConfigSource),
		)
	}

	db := args.Database
	var fp dt.Filepath
	filename := args.BaseFilename + db.QueryFileExt()
	if args.PrimaryDirType == cfgstore.ProjectConfigDirType {
		baseDir, err = args.DirsProvider.ProjectDirFunc()
		fp = dt.FilepathJoin3(baseDir, ConfigSlug, filename)
	} else {
		baseDir, err = args.DirsProvider.CLIConfigDirType()
		fp = dt.FilepathJoin4(baseDir, ConfigSlug, db.Type(), filename)
	}
	queryBytes, err = fp.ReadFile()
	if os.IsNotExist(err) {
		err = nil
		goto end
	}
	if err != nil {
		goto end
	}
	fileQuery = strings.TrimSpace(string(queryBytes))
	if len(fileQuery) == 0 {
		goto end
	}
	lineCnt = strings.Count(fileQuery, "\n") + 1
	//csFilepath, err = cs.GetFilepath()
	//if err != nil {
	//	goto end
	//}
	mpq.AddQuerySource(
		NewQuerySource(
			elemCnt+1,
			elemCnt+lineCnt,
			common.QueryString(fileQuery),
			fp,
			//csFilepath,
		),
	)
end:
	return mpq, err
}

type ParseDatabaseArgs struct {
	Options      *common.Options
	Writer       CLIWriter
	Logger       *slog.Logger
	DirsProvider *cfgstore.DirsProvider
	DirType      cfgstore.DirType
}

var ErrNoDatabaseConnectString = errors.New("no database connection string")

func ParseDatabase(ctx Context, cfg cfgldr.DatabaseConfig, args ParseDatabaseArgs) (db Database, err error) {
	var dbType DatabaseType
	var exts []DBExtension
	var bootstrapQueries, onOpenQueries *MultipartQuery
	var sourceFile dt.Filepath

	switch {
	case cfg.DatabaseType() != "":
		dbType = DatabaseType(cfg.DatabaseType())
	case args.Options.ConnectString != "":
		dbType, err = ParseDatabaseType(ctx, string(args.Options.ConnectString))
	case cfg.ConnectString() != "":
		dbType, err = ParseDatabaseType(ctx, cfg.ConnectString())
	default:
		err = NewErr(ErrNoDatabaseConnectString)
	}
	if err != nil {
		goto end
	}

	db, err = GetRegisteredDatabase(dbType)
	if db == nil {
		err = NewErr(ErrUnsupportedDBType, "database_type", dbType, err)
		goto end
	}

	exts, err = ParseExtensions(db, cfg.DBExtensions())
	if err != nil {
		goto end
	}

	sourceFile, err = dt.ParseFilepath(cfg.SourceFile())
	if err != nil {
		goto end
	}

	bootstrapQueries, err = ParseQueries(cfg.BootstrapQueries(), ParseQueriesArgs{
		Database:       db,
		BaseFilename:   "bootstrap",
		ConfigSource:   sourceFile,
		DirsProvider:   args.DirsProvider,
		PrimaryDirType: args.DirType,
	})
	if err != nil {
		goto end
	}

	onOpenQueries, err = ParseQueries(cfg.OnOpenQueries(), ParseQueriesArgs{
		Database:       db,
		BaseFilename:   "on_open",
		ConfigSource:   sourceFile,
		DirsProvider:   args.DirsProvider,
		PrimaryDirType: args.DirType,
	})
	if err != nil {
		goto end
	}

	db, err = db.CreateNew(DatabaseArgs{
		DatabaseType:     dbType,
		ConnectString:    cfg.ConnectString(),
		Port:             cfg.Port(),
		Extensions:       exts,
		BootstrapQueries: bootstrapQueries,
		OnOpenQueries:    onOpenQueries,
		AccessMode:       ReadOnlyMode, //TODO: Make this configurable somehow
		SourceFile:       sourceFile,
		Config:           cfg,
		Options:          args.Options,
		CLIWriter:        args.Writer,
		Logger:           args.Logger,
	})
end:
	return db, err
}

type DBExtension interface {
	Name() string
	DBExtension()
}

func ParseExtensions(db Database, exts []cfgldr.DBExtensionConfig) (dbExts []DBExtension, err error) {
	var errs []error
	var dbExt DBExtension
	dbExts = make([]DBExtension, 0, len(exts))
	for _, ext := range exts {
		dbExt, err = db.ParseExtension(ext)
		if err != nil {
			errs = append(errs, NewErr(
				pathvars.ErrParseFailed,
				pathvars.ErrParsingDBExtensionFailed,
				"database_type", db.Type(),
				"extension", ext,
				err,
			))
			continue
		}
		dbExts = append(dbExts, dbExt)
	}
	return dbExts, doterr.CombineErrs(errs)
}
