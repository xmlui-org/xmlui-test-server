package sqlite3pkg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

func init() {
	dbpkg.RegisterDatabase(&SQLite3{})
}

var _ dbpkg.Database = (*SQLite3)(nil)

type database = dbpkg.BaseDatabase

var registerDriverOnce sync.Once

type SQLite3 struct {
	*database
	JournalMode    JournalMode    // default true
	Synchronous    Synchronous    // "NORMAL" (default) or "FULL"
	ForeignKeyMode ForeignKeyMode // default ignore foreign keys
	BusyTimeout    time.Duration  // default 5s
	AutoCheckpoint int            // default 1000 pages (WAL mode only)
}

func (s *SQLite3) allowVTable(extName string) (allow bool) {
	for _, ext := range s.Extensions() {
		sExt, ok := ext.(*Extension)
		if !ok {
			s.WarnError(dbpkg.ErrFailedToTypeAssertToExtensionType.Error(),
				"name", ext.Name(),
				"type", fmt.Sprintf("%T", ext),
				"target_type", fmt.Sprintf("%T", (*Extension)(nil)))
			continue
		}
		if sExt.Name() != extName {
			continue
		}
		allow = sExt.allowVTable
		goto end
	}
end:
	return allow
}

func (*SQLite3) ParseQueryString(query string) (_ sqlparams.QueryString, err error) {
	// TODO: Add SQL Query validation
	return sqlparams.QueryString(query), err
}

type SQLite3Args struct {
	DatabaseArgs   dbpkg.DatabaseArgs
	JournalMode    JournalMode    // default true
	Synchronous    Synchronous    // "NORMAL" (default) or "FULL"
	ForeignKeyMode ForeignKeyMode // default ignore foreign keys
	BusyTimeout    time.Duration  // default 5s
	AutoCheckpoint int            // default 1000 pages (WAL mode only)
}

func NewSQLite3(args SQLite3Args) *SQLite3 {
	dbArgs := args.DatabaseArgs
	exts := make([]dbpkg.DBExtension, 0, len(dbArgs.Extensions))
	for _, ext := range dbArgs.Extensions {
		exts = append(exts, ext.(*Extension))
	}
	dbArgs.Extensions = exts
	db := &SQLite3{
		JournalMode:    args.JournalMode,
		Synchronous:    args.Synchronous,
		ForeignKeyMode: args.ForeignKeyMode,
		BusyTimeout:    args.BusyTimeout,
		AutoCheckpoint: args.AutoCheckpoint,
	}
	db.database = dbpkg.NewBaseDatabase(db, dbArgs)
	return db
}

func (*SQLite3) CreateNew(args dbpkg.DatabaseArgs) (ndb dbpkg.Database, err error) {
	var errs []error
	slCfg, ok := args.Config.(*cfgldr.SQLite3ConfigV1)
	if !ok {
		panic(fmt.Sprintf("Failed to type assert value of type %T to type %T",
			args.Config,
			(*cfgldr.SQLite3ConfigV1)(nil),
		))
	}

	db := &SQLite3{}
	db.JournalMode, err = ParseJournalMode(slCfg.JournalMode)
	errs = AppendErr(errs, err)
	db.Synchronous, err = ParseSynchronous(slCfg.Synchronous)
	errs = AppendErr(errs, err)
	db.ForeignKeyMode, err = ParseForeignKeyMode(slCfg.ForeignKeys)
	errs = AppendErr(errs, err)
	db.AutoCheckpoint, err = ParseAutoCheckpoint(slCfg.AutoCheckpoint)
	errs = AppendErr(errs, err)
	db.BusyTimeout, err = ParseBusyTimeout(slCfg.BusyTimeout)
	errs = AppendErr(errs, err)
	args.AccessMode, err = dbpkg.ParseAccessMode(slCfg.AccessMode)
	errs = AppendErr(errs, err)

	errs = AppendErr(errs, args.Normalize())

	err = CombineErrs(errs)
	if err != nil {
		db = nil
		goto end
	}

	ndb = NewSQLite3(SQLite3Args{
		DatabaseArgs:   args,
		JournalMode:    db.JournalMode,
		Synchronous:    db.Synchronous,
		ForeignKeyMode: db.ForeignKeyMode,
		AutoCheckpoint: db.AutoCheckpoint,
		BusyTimeout:    db.BusyTimeout,
	})
end:
	return ndb, err
}

func (*SQLite3) Type() dbpkg.DatabaseType {
	return dbpkg.SQLite3Database
}

func (s *SQLite3) SetBaseDatabase(db *dbpkg.BaseDatabase) {
	s.database = db
}

func (s *SQLite3) TypeName() string {
	return "SQLite3"
}

func (s *SQLite3) ParseExtension(dbExtCfg dbpkg.DBExtensionConfig) (dbExt dbpkg.DBExtension, err error) {
	var fp dt.Filepath
	var status dt.EntryStatus

	sExtCfg, ok := dbExtCfg.(*cfgldr.SQLite3ExtensionConfigV1)
	if !ok {
		err = NewErr(
			dbpkg.ErrFailedToTypeAssertToExtensionType,
			"expected_type", fmt.Sprintf("%T", (*cfgldr.SQLite3ExtensionConfigV1)(nil)),
		)
		goto end
	}
	fp, err = dt.ParseFilepath(sExtCfg.Filepath)
	if err != nil {
		goto end
	}
	status, err = fp.Status()
	if err != nil {
		goto end
	}
	if status == dt.IsFileEntry {
		dbExt = NewExtension(fp, ExtensionArgs{
			// TODO Assign sExtCfg args
		})
	}
end:
	return dbExt, err
}

func (s *SQLite3) ValidatedConnection(ctx dbpkg.Context, dbType dbpkg.DatabaseType, connStr localsvr.ConnectString) (err error) {
	var fp dt.Filepath
	fp, err = dt.ParseFilepath(string(connStr))
	if err != nil {
		goto end
	}
	err = s.ValidateFileConnection(ctx, dbType, fp)
end:
	return err
}

func (s *SQLite3) String() string {
	return s.HomeRelativeFile()
}

func (s *SQLite3) Open(ctx context.Context) (err error) {
	var cancel context.CancelFunc
	var timeout time.Duration

	denyUnlessAuthorized = false

	s.V2().InfoPrint("Opening SQLite database", "database_file", s.HomeRelativeFile())

	// Register the SQLite driver only once using sync.Once
	registerDriverOnce.Do(func() {
		sql.Register("sqlite3_ext", &sqlite3.SQLiteDriver{
			ConnectHook: s.ConnectHook(),
		})
	})

	cs := s.ConnectString()
	err = dt.Filepath(cs).Dir().MkdirAll(0755)
	if err != nil {
		err = WithErr(
			dt.ErrFailedToMakeDirectory,
			err,
		)
		goto end
	}

	// Simple connection string with extension loading enabled
	s.DB, err = sql.Open("sqlite3_ext", cs+"?_allow_load_extension=1")
	if err != nil {
		err = WithErr(
			dt.ErrFailedToOpenDatabase,
			err,
		)
		goto end
	}

	// SQLite specific configurations Limiting to 1 connection means we don't have to
	// manage concurrent database connections, and since this is intended as a local
	// dev server and not a production server that should be more than sufficient.
	s.V3().Printf("Setting MaxOpenConnections to 1\n")
	s.SetMaxOpenConns(1)
	s.V3().Printf("Setting MaxIdleConnections to 1\n")
	s.SetMaxIdleConns(1)

	// Sanity ping with deadline
	timeout = s.Options().Timeout
	s.V3().Printf("Setting Timeout to %d seconds\n", timeout/time.Second)
	if timeout == 0 {
		timeout = time.Hour * 24 * 365 * 100 // 100 years
	}
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	s.V3().Printf("Pinging database to confirm connection\n")
	err = s.PingContext(ctx)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToPingDatabase,
			err,
		)
		goto end
	}
	s.V3().Printf("Connection confirmed\n")

	err = s.execQueriesIfExists("bootstrap", s.BootstrapQueries)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToExecuteQueries,
			"query_type", "bootstrap",
			err,
		)
		goto end
	}

	err = s.execQueriesIfExists("on_open", s.OnOpenQueries)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToExecuteQueries,
			"query_type", "on_open",
			err,
		)
		goto end
	}

	s.V2().InfoPrint("Database opened")

end:
	denyUnlessAuthorized = true
	if err != nil {
		err = WithErr(err,
			dt.ErrFailedToOpenDatabase,
			"db_file", s.HomeRelativeFile(),
		)
	}
	return err
}

func (s *SQLite3) execQueriesIfExists(qt string, q *dbpkg.MultipartQuery) (err error) {
	if !q.HasQueries() {
		goto end
	}
	s.V2().InfoPrint("Running queries", "query_type", qt)
	// TODO Split out individual queries and run the separately to allow for more
	//      targeted error messages.
	_, err = s.Exec(string(q.Source()))
	if err != nil {
		// TODO: Do we want to fail to run the server or allow failed initialization SQL?
		s.WarnError("Failed to run query", "type", qt, "error", err, "query", q.Source())
	}
end:
	return err
}

func (s *SQLite3) ConnectHook() func(*sqlite3.SQLiteConn) error {
	return func(conn *sqlite3.SQLiteConn) (err error) {
		var errs []error

		for p, fn := range GetPragmasFuncs {
			pv := fn(s)
			pragma := fmt.Sprintf("PRAGMA %s=%s", p, pv)
			_, err := conn.Exec(pragma, nil)
			if err != nil {
				s.WarnError("Failed to execute SQLite3 PRAGMA", "pragma", p, "value", pv, "error", err)
			}
		}

		// Create memory database for extensions
		// Concatenation used to stop IDE from flagging this an an error
		_, err = conn.Exec(`ATTACH `+`DATABASE ':memory:' AS mem`, nil)
		if err != nil {
			msg := "Failed to attach memory database 'mem'."
			s.WarnError(msg, "error", err)
			// TODO: Do we want to fail to run the server or allow without extension?
		}

		_, err = conn.Exec(string(s.OnOpenQueries.Source()), nil)
		if err != nil {
			// TODO Elaborate on s.OnOpenQueries for context
			errs = append(errs, err)
		}

		// authorizer: deny engine mutation / native-code abuse; mode-specific extras
		conn.RegisterAuthorizer(s.authorizer())

		if !s.HasExtensions() {
			goto end
		}

		// If extension is provided, try to load it
		err = s.LoadExtensions()
		if err != nil {
			s.WarnError("Failed to load SQLite3 extensions", "error", err)
			// TODO Should we fail here, or continue on?
		}

	end:
		return CombineErrs(errs)
	}
}

var denyUnlessAuthorized bool

type authorizerFunc = func(int, string, string, string) int

// authorizer tests access modes to determine if user is authorized to run specified Sqlite3 operations
func (s *SQLite3) authorizer() authorizerFunc {
	return func(op int, funcName, extName, arg3 string) (decision int) {

		if !denyUnlessAuthorized {
			decision = sqlite3.SQLITE_OK
			goto end
		}
		// Always deny these special cases
		decision = sqlite3.SQLITE_DENY

		// Test for virtual tables per extension
		if op == sqlite3.SQLITE_CREATE_VTABLE && !s.allowVTable(extName) {
			goto end
		}

		// Test to see if an op is recognized
		if !isRecognizedOp(op) {
			s.WarnError("Unrecognized SQLite operation", "op", op)
		}

		// Test the cases where mode+ops are the only criteria
		if s.IsAuthorizedSQLite3Operation(op, funcName) {
			decision = sqlite3.SQLITE_OK
			goto end
		}
	end:
		return decision
	}
}

func (s *SQLite3) GetFormatParamFunc() dbpkg.FormatParamFunc {
	return func(_ int) string {
		return "?"
	}
}

func (s *SQLite3) LoadExtension(dbExt dbpkg.DBExtension) (err error) {
	var filePath, absPath, loadSQL string
	var mu sync.Mutex

	ext, ok := dbExt.(*Extension)
	if !ok {
		err = NewErr(ErrFailedToTypeAssertToSQLite3Extension, "extension_name", dbExt.Name())
		goto end
	}

	mu.Lock()
	defer mu.Unlock()

	// TODO Check dt.Filepath for URL and download if applicable

	filePath = string(ext.filePath)
	// Get the absolute path to the extension file
	absPath, err = filepath.Abs(filePath)
	if err != nil {
		s.WarnError("Failed to get absolute path for extension", "error", err)
		absPath = filepath.Join("./", filePath)
	}

	// Ensure file has execute permissions (required for Linux)
	err = os.Chmod(absPath, 0755)
	if err != nil {
		s.WarnError("Failed to set execute permissions on extension", "error", err)
	}

	// Log extension loading attempt
	s.InfoPrint("Loading extension", "extension", absPath)

	// TODO: This is vulnerable to SQL injection; we should harden it
	loadSQL = fmt.Sprintf("SELECT load_extension('%s')", strings.ReplaceAll(absPath, "'", "''"))
	_, err = s.Exec(loadSQL)
	if err != nil {
		s.WarnError("Extension loading failed with", "error", err)
		goto end
	}
	s.V2().Printf("Extension loaded successfully\n")

end:
	return err
}

// ParseConnectString injects or overrides the port in a Postgres connection string (URL or DSN format)
func (s *SQLite3) ParseConnectString(cs string) (_ localsvr.ConnectString, err error) {
	// TODO Add validation here
	return localsvr.ConnectString(cs), err
}

func (s *SQLite3) IsAuthorizedSQLite3Operation(op int, funcName string) (allowed bool) {
	var ok bool
	var deniedOpMode dbpkg.AccessMode
	// Test these special cases first
	if op == sqlite3.SQLITE_FUNCTION &&
		s.AccessMode < dbpkg.SuperAdminMode &&
		strings.EqualFold(funcName, "load_extension") {
		goto end
	}
	deniedOpMode, ok = accessModeOpsDenied[op]
	if !ok {
		goto end
	}
	allowed = s.AccessMode > deniedOpMode
end:
	return allowed
}

func (s *SQLite3) ConvertValue(value any, dt sqlparams.DBDataType) any {
	switch dt {
	case sqlparams.IntegerDBDataType:
		s, ok := value.(string)
		if !ok {
			goto end
		}
		switch strings.ToLower(s) {
		case "true":
			value = 1
		case "false":
			value = 0
		default:
			// Keep original value if it's not a recognized boolean string
		}
	}
end:
	return value
}
