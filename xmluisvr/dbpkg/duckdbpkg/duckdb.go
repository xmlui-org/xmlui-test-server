package duckdbpkg

import (
	"context"
	"fmt"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

func init() {
	dbpkg.RegisterDatabase(&DuckDB{})
}

var _ dbpkg.Database = (*DuckDB)(nil)

type database = dbpkg.BaseDatabase

type DuckDB struct {
	*database
}

func (d *DuckDB) GetFormatParamFunc() dbpkg.FormatParamFunc {
	return func(index int) string {
		return fmt.Sprintf("$%d", index)
	}
}

func (d *DuckDB) SetBaseDatabase(db *dbpkg.BaseDatabase) {
	d.database = db
}

func (d *DuckDB) Open(_ context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (*DuckDB) ParseQueryString(query string) (_ sqlparams.QueryString, err error) {
	// Add SQL Query validation
	return sqlparams.QueryString(query), err
}

func (d *DuckDB) String() string {
	return d.HomeRelativeFile()
}

func (d *DuckDB) TypeName() string {
	return "DuckDB"
}

func (d *DuckDB) ValidatedConnection(ctx dbpkg.Context, dbType dbpkg.DatabaseType, connStr common.ConnectString) (err error) {
	var fp dt.Filepath
	fp, err = dt.ParseFilepath(string(connStr))
	if err != nil {
		goto end
	}
	err = d.ValidateFileConnection(ctx, dbType, fp)
end:
	return err
}

// ParseConnectString injects or overrides the port in a DuckDB connection string (URL or DSN format)
func (d *DuckDB) ParseConnectString(cs string) (_ common.ConnectString, err error) {
	// TODO Add validation
	return common.ConnectString(cs), err
}

func (*DuckDB) Type() dbpkg.DatabaseType {
	return dbpkg.DuckDBDatabase
}

func NewDuckDB(args dbpkg.DatabaseArgs) *DuckDB {
	db := &DuckDB{}
	db.database = dbpkg.NewBaseDatabase(db, args)
	return db
}
func (*DuckDB) CreateNew(args dbpkg.DatabaseArgs) (_ dbpkg.Database, err error) {
	return NewDuckDB(args), err
}
