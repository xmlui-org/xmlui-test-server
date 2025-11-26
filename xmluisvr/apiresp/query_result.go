package apiresp

import (
	"fmt"
	"net/http"

	"github.com/mikeschinkel/go-rfc9457"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"

	. "github.com/mikeschinkel/go-doterr"
)

var _ ResponsePayload = (*QueryResult)(nil)

type QueryResult dbpkg.QueryResult

func (qr QueryResult) GetByCardinality(c sqlparams.Cardinality) (content any, err error) {
	switch c {
	case sqlparams.ManyRowsOrNone:
		if len(qr) == 0 {
			goto end
		}

	case sqlparams.ManyRows:
		if len(qr) == 0 {
			err = dbpkg.ErrManyRowsExpectedZeroReturned
			goto end
		}

	case sqlparams.OneRowOrNone:
		if len(qr) == 0 {
			goto end
		}
		fallthrough

	case sqlparams.OneRow:
		if len(qr) == 0 {
			err = dbpkg.ErrOneRowExpectedZeroReturned
			goto end
		}
		if len(qr) > 1 {
			err = NewErr(
				dbpkg.ErrOneRowExpectedManyReturned,
				"row_count", len(qr),
			)
			goto end
		}
		content = dbpkg.TableRows(qr)[0]

	default:
		content = qr
	}

end:
	if err != nil {
		err = WithErr(err, dbpkg.ErrInvalidCardinality)
	}
	return qr, err
}

func NewQueryResult(qr dbpkg.QueryResult) QueryResult {
	return QueryResult(qr)
}

func (QueryResult) ResponsePayload() {}
func (qr QueryResult) MIMEType() rfc9457.MIMEType {
	return rfc9457.ApplicationJSON
}
func (qr QueryResult) Error() string {
	return fmt.Sprintf("QueryResult\nrow_count=%d", len(qr))
}

func (qr QueryResult) HTTPStatusCode() int {
	if len(qr) == 0 {
		return http.StatusNotFound
	}
	return http.StatusOK
}
