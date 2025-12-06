package dbpkg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

type BaseDatabase struct {
	*sql.DB
	cliutil.WriterLogger
	dbType           DatabaseType
	conn             string
	parent           Database
	BootstrapQueries *MultipartQuery
	OnOpenQueries    *MultipartQuery
	extensions       []DBExtension
	sourceFile       dt.Filepath
	options          *localsvr.Options
	AccessMode       AccessMode
	Initialized      bool
}

func (db *BaseDatabase) Options() localsvr.Options {
	return *db.options
}

func (db *BaseDatabase) ConvertValue(value any, dt sqlparams.DBDataType) any {
	return value
}

func (db *BaseDatabase) String() string {
	return fmt.Sprintf("%s://%s", db.dbType, db.conn)
}

func NewBaseDatabase(parent Database, args DatabaseArgs) *BaseDatabase {
	if args.AccessMode == UnspecifiedAccessMode {
		args.AccessMode = ReadWriteMode
	}
	return &BaseDatabase{
		parent:           parent,
		dbType:           args.DatabaseType,
		conn:             args.ConnectString,
		BootstrapQueries: args.BootstrapQueries,
		OnOpenQueries:    args.OnOpenQueries,
		extensions:       args.Extensions,
		options:          args.Options,
		AccessMode:       args.AccessMode,
		sourceFile:       args.SourceFile,
		WriterLogger:     cliutil.NewWriterLogger(args.CLIWriter, args.Logger),
	}
}

func (db *BaseDatabase) Close() error {
	return db.DB.Close()
}

func (db *BaseDatabase) SourceFile() dt.Filepath {
	return db.sourceFile
}

func (db *BaseDatabase) QueryFileExt() string {
	return ".sql"
}

func (db *BaseDatabase) HasExtensions() bool {
	return len(db.extensions) > 0
}

func (db *BaseDatabase) LoadExtensions() error {
	var errs = make([]error, 0)

	for _, ext := range db.Extensions() {
		errs = append(errs, db.parent.LoadExtension(ext))
	}
	return CombineErrs(errs)
}

func (db *BaseDatabase) LoadExtension(_ DBExtension) error {
	db.checkForExtensions()
	// Stub for those databases for which we do not current support extensions.
	return NewErr(ErrExtensionsUnsupportedForDBType, "database_type", db.dbType)
}

func (db *BaseDatabase) ParseExtension(_ DBExtensionConfig) (DBExtension, error) {
	db.checkForExtensions()
	// Stub for those databases for which we do not current support extensions.
	return nil, NewErr(ErrExtensionsUnsupportedForDBType, "database_type", db.dbType)
}

func (db *BaseDatabase) checkForExtensions() {
	var msg string
	if len(db.extensions) == 0 {
		goto end
	}
	msg = "LoadExtension() not implemented for database type"
	db.writeErrorf("%s '%s'. Did you forget to implement?  Extensions:", msg, db.dbType)
	for _, ext := range db.extensions {
		db.writeErrorf("- %s", ext.Name())
	}
	db.logError(msg,
		slog.String("database_type", string(db.dbType)),
		slog.Any("extensions", db.extensions),
	)
end:
}

func (db *BaseDatabase) Initialize(ctx Context) (err error) {
	if db.Initialized {
		goto end
	}
	err = db.parent.Open(ctx)
	db.Initialized = true
end:
	return err
}

func (db *BaseDatabase) ConnectString() string {
	return db.conn
}

func (db *BaseDatabase) SetConnectString(cs string) {
	db.conn = cs
}

func (db *BaseDatabase) Query(ctx Context, q string, params ...any) (*sql.Rows, error) {
	return db.QueryContext(ctx, q, params...)
}

// ValidateFileConnection checks for file connections which work for SQLite3 and DuckDB.
func (db *BaseDatabase) ValidateFileConnection(ctx Context, dbType DatabaseType, cs dt.Filepath) (err error) {
	var status dt.EntryStatus
	status, err = cs.Status()
	switch status {
	case dt.IsEntryError:
		err = NewErr(err)
	case dt.IsFileEntry:
		// What we are looking for; carry on!
		err = db.PingDB(ctx, dbType, localsvr.ConnectString(cs))
	case dt.IsSymlinkEntry:
		// Follow the symlink
		var newCS dt.Filepath
		newCS, err = cs.Readlink()
		if err != nil {
			goto end
		}
		err = db.ValidateFileConnection(ctx, dbType, newCS)
	case dt.IsMissingEntry:
		err = dt.ErrFileDoesNotExist
	default:
		err = dtx.EntryStatusError(status)
	}
end:
	if err != nil {
		err = WithErr(err,
			dt.ErrConnectFailed,
		)
	}
	return err
}

// PingDB checks for file connections which work for SQLite3 and DuckDB.
func (db *BaseDatabase) PingDB(_ Context, dbType DatabaseType, cs localsvr.ConnectString) (err error) {
	var sqlDB *sql.DB

	defer func() {
		if err != nil {
			return
		}
		e := recover()
		if e != nil {
			err = e.(error)
		}
	}()
	sqlDB, err = sql.Open(string(dbType), string(cs))
	if err != nil {
		goto end
	}
	err = sqlDB.Ping()
	if err != nil {
		goto end
	}
	dt.CloseOrLog(sqlDB) // DO NOT move this up and defer it; that will cascade errors
end:
	return err
}

func (db *BaseDatabase) HomeRelativeFile() string {
	absPath, err := filepath.Abs(db.conn)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path of '%s': %v", db.conn, err))
	}
	return localsvr.HomeRelative(absPath)
}

func (db *BaseDatabase) Extensions() []DBExtension {
	return db.extensions
}

// writeErrorf allows calling db.Logger.Error() where it is obvious that this
// is writing to the stdio without the linter complaining that I can remove
// .Writer if called directly.
func (db *BaseDatabase) writeErrorf(format string, args ...any) {
	db.Errorf(format, args...)
}

// logError allows calling db.Logger.Error() where it is obvious that this
// is writing to the stdio without the linter complaining that I can remove
// .Writer if called directly.
func (db *BaseDatabase) logError(msg string, attrs ...any) {
	db.Error(msg, attrs...)
}
