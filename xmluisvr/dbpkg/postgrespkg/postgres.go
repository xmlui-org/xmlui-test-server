package postgrespkg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

func init() {
	dbpkg.RegisterDatabase(&Postgres{})
}

var _ dbpkg.Database = (*Postgres)(nil)

type database = dbpkg.BaseDatabase
type Postgres struct {
	*database
	port   int
	writer cliutil.Writer
	logger *slog.Logger
}

func (d *Postgres) GetFormatParamFunc() dbpkg.FormatParamFunc {
	return func(index int) string {
		return fmt.Sprintf("$%d", index)
	}
}

func (p *Postgres) CreateNewFromConfig(config cfgldr.DatabaseConfig) (dbpkg.Database, error) {
	//TODO implement me
	panic("implement me")
}

func (*Postgres) ParseQueryString(query string) (_ sqlparams.QueryString, err error) {
	// Add SQL Query validation
	return sqlparams.QueryString(query), err
}
func (p *Postgres) CreateNew(args dbpkg.DatabaseArgs) (_ dbpkg.Database, err error) {
	return NewPostgres(args), err
}

func (p *Postgres) TypeName() string {
	return "Postgres"
}

func (p *Postgres) String() string {
	//TODO parse out database name from URL or DSN
	return p.TypeName()
}

func (*Postgres) Type() dbpkg.DatabaseType {
	return dbpkg.PostgresDatabase
}

func NewPostgres(args dbpkg.DatabaseArgs) *Postgres {
	pdb := &Postgres{}
	if args.Port != 0 {
		pdb.port = args.Port
	}
	pdb.database = dbpkg.NewBaseDatabase(pdb, args)
	return pdb
}

func (p *Postgres) IsConnectString(cs string) bool {
	_, err := p.ParseConnectString(cs)
	return err == nil
}

func (p *Postgres) Open(_ context.Context) (err error) {
	var cs common.ConnectString

	p.writer.Printf("Using PostgreSQL database\n")
	p.logger.Info("Opening PostgreSQL database")
	cs, err = p.ParseConnectString(p.ConnectString())
	if err != nil {
		err = NewErr(dt.ErrInvalidConnectString, err)
		goto end
	}
	p.DB, err = sql.Open("postgres", string(cs))
	if err != nil {
		err = NewErr(dt.ErrConnectFailed, err)
		goto end
	}
end:
	return err
}
func (p *Postgres) SetBaseDatabase(db *dbpkg.BaseDatabase) {
	p.database = db
}

func (p *Postgres) Query(ctx dbpkg.Context, q string, params ...any) (*sql.Rows, error) {
	return p.database.Query(ctx, FormatQueryForPostgres(q), params...)
}

func (p *Postgres) ValidatedConnection(ctx dbpkg.Context, dbType dbpkg.DatabaseType, connStr common.ConnectString) (err error) {
	return p.PingDB(ctx, dbType, connStr)
}

// ParseConnectString injects or overrides the port in a Postgres connection string (URL or DSN format)
func (p *Postgres) ParseConnectString(cs string) (common.ConnectString, error) {
	return ParsePGConnectString(cs, p.port)
}

var postgresPrefixRE = regexp.MustCompile(`^\s*postgres(ql)?://`)
var dsnFormatRE = regexp.MustCompile(`port=\\d+`)

// ParsePGConnectString injects or overrides the port in a Postgres connection string (URL or DSN format)
func ParsePGConnectString(cs string, port int) (_ common.ConnectString, err error) {
	if port == 0 {
		goto end
	}
	if postgresPrefixRE.MatchString(cs) {
		var u *url.URL
		u, err = url.Parse(cs)
		if err != nil {
			goto end
		}
		if u == nil {
			err = fmt.Errorf("invalid PostgreSQL connection string: '%s'", cs)
			goto end
		}
		if u.Port() == "" {
			goto end
		}
		if u.Port() == strconv.Itoa(port) {
			goto end
		}
		u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
		cs = u.String()
		goto end
	}

	// DSN format: add or replace port=...
	// TODO Can we do more to validate here?
	if dsnFormatRE.MatchString(cs) {
		cs = dsnFormatRE.ReplaceAllString(cs, fmt.Sprintf("port=%d", port))
		goto end
	}
	cs = fmt.Sprintf("%s port=%d", cs, port)
end:
	return common.ConnectString(cs), err
}

// FormatQueryForPostgres replaces ? in query w/numbered params in $n format
func FormatQueryForPostgres(q string) string {
	newQ := make([]byte, len(q)*2)
	pNum := 1
	i, j := 0, 0
	for i < len(q) {
		newQ[j] = q[i]
		if q[i] != '?' {
			i++
			j++
			continue
		}
		i++
		newQ[j] = '$'
		j++
		for n, c := range strconv.Itoa(pNum) {
			newQ[j+n] = byte(c)
			j++
		}
		pNum++
	}
	return string(newQ[:j])
}
