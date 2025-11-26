package dbpkg

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-sqlparams"
)

var mutex sync.Mutex

// ExecuteQuery and return results as a map of any
func ExecuteQuery(ctx Context, db Database, query sqlparams.QueryString, params []any) (result QueryResult, err error) {
	var rows *sql.Rows
	var columns []string

	result = make(QueryResult, 0)

	mutex.Lock()
	defer mutex.Unlock()

	// Log the SQL query (just once)
	cliutil.Printf("Query: %s %v\n", query, formatParams(db, params))

	// Execute the query
	rows, err = db.Query(ctx, string(query), params...)
	if err != nil {
		goto end
	}
	defer closeOrLog(rows)

	// Get column information
	columns, err = rows.Columns()
	if err != nil {
		goto end
	}

	// Process result rows
	for rows.Next() {
		// Create values slice with appropriate length
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into values
		err := rows.Scan(valuePtrs...)
		if err != nil {
			goto end
		}

		// Create a map for this row
		entry := make(TableRow)
		for i, col := range columns {
			var v any
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[ColumnName(col)] = v
		}

		// Add the row to the result
		result = append(result, entry)
	}

	// Check for errors after iteration
	err = rows.Err()

end:
	return result, err
}

func formatParams(db Database, params []any) (out []string) {
	fn := db.GetFormatParamFunc()
	out = make([]string, len(params))
	for i, p := range params {
		out[i] = fmt.Sprintf("%s=%v", fn(i), p)
	}
	return out
}
