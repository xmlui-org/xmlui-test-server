package dbpkg

import (
	"github.com/mikeschinkel/go-sqlparams"
)

type (
	ColumnName string
	TableCell  any
	TableRow   map[ColumnName]TableCell
	TableRows  []TableRow
)

type QueryResult TableRows

func (qr QueryResult) GetByCardinality(c sqlparams.Cardinality) (content any, err error) {
	switch c {
	case sqlparams.ManyRowsOrNone:
		if len(qr) == 0 {
			goto end
		}

	case sqlparams.ManyRows:
		if len(qr) == 0 {
			err = ErrManyRowsExpectedZeroReturned
			goto end
		}

	case sqlparams.OneRowOrNone:
		if len(qr) == 0 {
			goto end
		}
		fallthrough

	case sqlparams.OneRow:
		if len(qr) == 0 {
			err = ErrOneRowExpectedZeroReturned
			goto end
		}
		if len(qr) > 1 {
			err = NewErr(
				ErrOneRowExpectedManyReturned,
				"row_count", len(qr),
			)
			goto end
		}
		content = TableRows(qr)[0]

	default:
		content = qr
	}

end:
	if err != nil {
		err = WithErr(ErrInvalidCardinality, err)
	}
	return content, err
}
