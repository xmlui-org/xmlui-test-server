package apipkg

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mikeschinkel/go-pathvars"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

type HandlerHelperArgs struct {
	HTTPRequest *http.Request
	MatchResult pathvars.MatchResult
	APIResponse *apiresp.Response
	Database    dbpkg.Database
	Endpoint    *Endpoint
	QueryValues []any
	QueryResult apiresp.QueryResult
	DBQuery     sqlparams.QueryString
	RequestBody bytes.Buffer
	Content     any
	TargetURL   *url.URL
	URLPath     common.URLPath
}

func (args HandlerHelperArgs) GetQueryString() (qs sqlparams.QueryString, err error) {
	var dbq sqlparams.ParsedQuery
	if args.DBQuery != "" {
		qs = args.DBQuery
	}
	if args.Endpoint == nil {
		err = apiresp.ErrNeitherDBQueryNorEndpointSet
		goto end
	}
	dbq = args.Endpoint.ParsedQuery
	if dbq == nil {
		err = apiresp.ErrNeitherDBQueryNorEndpointSet
		goto end
	}
	qs = dbq.QueryString()
end:
	return qs, err
}

func (args HandlerHelperArgs) APIEndpointRequested() string {
	r := args.HTTPRequest
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

func (args HandlerHelperArgs) SendResponseArgs(err error) SendResponseArgs {
	return SendResponseArgs{
		HTTPRequest: args.HTTPRequest,
		APIResponse: args.APIResponse,
		Content:     args.Content,
		Error:       err,
	}
}

func (args HandlerHelperArgs) ErrorMeta(err error) []any {
	dbq := args.Endpoint.ParsedQuery
	meta := []any{
		"endpoint", args.Endpoint.Endpoint(),
		"url_params", args.MatchResult.ValuesMap(),
		"sql_query", dbq.QueryString(),
		"sql_params", dbq.Parameters(),
		// TODO Add more properties?
	}
	if err != nil {
		meta = append(meta, err)
	}
	return meta
}

func (api *API) GetResponseContent(args HandlerHelperArgs) (content any, err error) {

	// Now get the response content
	result := args.MatchResult
	dbResult := args.QueryResult

	content, err = args.QueryResult.GetByCardinality(sqlparams.Cardinality(result.Route.Cardinality))
	switch {
	case errors.Is(err, dbpkg.ErrManyRowsExpectedZeroReturned):
		fallthrough

	case errors.Is(err, dbpkg.ErrOneRowExpectedZeroReturned):
		err = apiresp.NoResultsPayload(args.HTTPRequest, apiresp.PayloadArgs{
			Error:            err,
			ErrorStyle:       api.Options.ErrorStyle,
			DBQuery:          args.Endpoint.ParsedQuery.QueryString(),
			EndpointTemplate: result.Route.Endpoint(),
		}).NewErr(
			ErrQueryValuesExtractionFailed,
			err,
		)
		goto end

	case errors.Is(err, dbpkg.ErrOneRowExpectedManyReturned):
		err = apiresp.NoResultsPayload(args.HTTPRequest, apiresp.PayloadArgs{
			Error:            err,
			ErrorStyle:       api.Options.ErrorStyle,
			DBQuery:          args.Endpoint.ParsedQuery.QueryString(),
			EndpointTemplate: result.Route.Endpoint(),
		}).NewErr(
			ErrQueryValuesExtractionFailed,
			"rows_returned", len(dbResult),
			err,
		)

	case err != nil:
		err = NewErr(
			ErrQueryValuesExtractionFailed,
			apiresp.CurrentlyUnhandledErrorPayload(args.HTTPRequest, apiresp.PayloadArgs{
				Location:         "CHANGE ME", // TODO: Determine appropriate value by breakpoint debugging during tests
				Error:            err,
				EndpointTemplate: result.Route.Endpoint(),
			}),
			err,
		)

	default:
		// S'all good, man!

	}
	if err != nil {
		err = WithErr(err, ErrGettingResponseContent)
	}
end:
	return content, err
}
