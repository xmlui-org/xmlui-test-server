package dbpkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

var ErrInvalidConnectString = dt.ErrInvalidConnectString

type DatabaseType string

func ParseDatabaseType(ctx Context, connStr string) (dt DatabaseType, err error) {
	var errs []error
	var cs common.ConnectString
	for dbType, db := range databaseMap {
		cs, err = db.ParseConnectString(connStr)
		if err != nil {
			err = NewErr(ErrInvalidConnectString)
			goto end
		}
		err := db.ValidatedConnection(ctx, dbType, cs)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		dt = db.Type()
		goto end
	}
	err = NewErr(
		ErrConnectStringNotSupported,
		CombineErrs(errs),
	)
end:
	if err != nil {
		err = WithErr(err,
			"connect_string", connStr,
		)
	}
	return dt, err
}
