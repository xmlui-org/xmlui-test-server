package apipkg

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mikeschinkel/go-pathvars"
	"github.com/mikeschinkel/go-pathvars/pvtypes"
	"github.com/mikeschinkel/go-rfc9457"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

// HandleAPIFunc returns an HTTP handler function that processes API requests.
// The handler:
//  1. Matches the request path against configured endpoints
//  2. Extracts path parameters and request body
//  3. Loads the SQL query for the matched endpoint
//  4. Executes the query against the database
//  5. Returns the results as JSON
//
// Returns 404 for unmatched routes and 500 for server errors.
func (api *API) HandleAPIFunc(db dbpkg.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var result pathvars.MatchResult
		var err error
		var args HandlerHelperArgs

		api.Writer.Printf("Handling API Request: %s %s\n", r.Method, r.URL.Path)

		args = HandlerHelperArgs{
			HTTPRequest: r,
			MatchResult: result,
			Database:    db,
			APIResponse: apiresp.NewResponse(apiresp.ResponseArgs{
				HTTPWriter: w,
				Request:    r,
				CLIWriter:  api.Writer,
				Logger:     api.Logger,
			}),
		}

		// Find the matching endpoint
		args.Endpoint, args.MatchResult, err = api.getMatchingEndpoint(args)
		if err != nil {
			goto end
		}

		// Now get the query values
		args.QueryValues, err = api.getQueryValues(args)
		if err != nil {
			goto end
		}

		// Now call the SQL query
		args.QueryResult, err = api.GetQueryResult(r.Context(), args)
		if err != nil {
			goto end
		}

		// Now get the response content
		args.Content, err = api.GetResponseContent(args)
		if err != nil {
			goto end
		}

		// Finally, send the success response. `args.Content` is expected to
		// contain the value to return as JSON.
		api.SendSuccessResponse(args.SendResponseArgs(nil))
	end:
		if err != nil {
			// Or, send the error if it was an error This assumes that err will have been
			// joined with a `ResponsePayload` — which by declaration is also an `error` —
			// one example being rfc9457.Response.
			api.SendErrorResponse(args.SendResponseArgs(err))
		}
	}
}

func (api *API) getQueryValues(args HandlerHelperArgs) (queryValues []any, err error) {
	var missing []apiresp.MissingParameter
	var r *http.Request

	result := args.MatchResult
	endpoint := args.Endpoint

	// Validate Params-defined query parameters BEFORE extracting values for SQL
	// Template-defined query params are already validated by Router.Match()
	err = endpoint.ValidateQueryParameters(result)
	if err != nil {
		goto end
	}
	r = args.HTTPRequest

	queryValues, missing, err = endpoint.GetParameterValues(ParameterValuesArgs{
		ValuesMap:  result.ValuesMap(),
		BodyReader: r.Body,
		Headers:    nil, // TODO: Not yet supported
		Database:   args.Database,
	})
	if err == nil {
		goto end
	}
	// Capture the cause
	switch {
	case len(missing) != 0:
		err = apiresp.MissingParametersPayload(r, apiresp.PayloadArgs{
			MissingParameters: missing,
			Error:             err,
		}).NewErr(
			ErrMissingQueryParameters,
			"missing_parameters", missing, // TODO Does this need to be converted to string?
			err,
		)
	default:
		err = apiresp.CurrentlyUnhandledErrorPayload(r, apiresp.PayloadArgs{
			Location: "CHANGE ME", // TODO: Determine appropriate value by breakpoint debugging during tests
			Error:    err,
		}).NewErr(
			ErrUnhandledParameterError,
			err,
		)
	}
	err = WithErr(err,
		ErrQueryValuesExtractionFailed,
		// TODO What of any of this would add value if returned as error meta?
		// 	HTTPRequest *http.Request
		// 	MatchResult pathvars.MatchResult
		// 	APIResponse *apiresp.Response
		// 	Database    dbpkg.Database
		// 	Endpoint    *Endpoint
		// 	QueryValues []any
		// 	QueryResult apiresp.QueryResult
		// 	DBQuery     sqlparams.QueryString
		// 	RequestBody bytes.Buffer
		// 	Content     any
		// 	TargetURL   *url.URL
		// 	URLPath     common.URLPath
	)
end:
	return queryValues, err
}

func (api *API) GetQueryResult(ctx Context, args HandlerHelperArgs) (dbResult apiresp.QueryResult, err error) {
	var rows dbpkg.QueryResult

	endpoint := args.Endpoint
	dbq := endpoint.ParsedQuery
	qs := dbq.QueryString()
	rows, err = dbpkg.ExecuteQuery(ctx, args.Database, qs, args.QueryValues)
	if err != nil {
		// Generate a response payload
		err = apiresp.QueryFailedPayload(args.HTTPRequest, apiresp.PayloadArgs{
			ErrorStyle: api.Options.ErrorStyle,
			Error:      err,
		}).NewErr(
			ErrQueryValuesExtractionFailed,
			err,
		)
		goto end
	}
	dbResult = apiresp.NewQueryResult(rows)

	api.V3().InfoPrint("Database query submitted.",
		"requestor_ip", args.HTTPRequest.RemoteAddr,
		"query", strings.ReplaceAll(string(args.DBQuery), "\n", " "),
	)

end:
	return dbResult, err
}

func (api *API) getMatchingEndpoint(args HandlerHelperArgs) (ep *Endpoint, mr pathvars.MatchResult, err error) {
	// Find the matching endpoint
	mr, err = api.tryMatchingRequest(args)
	if err != nil {
		goto end
	}
	// Assign the endpoint to the helper args
	ep = api.Endpoints[mr.Index]
end:
	return ep, mr, err
}

func (api *API) handleFailedMatch(err error, args HandlerHelperArgs) error {
	var httpStatus int
	var resp *rfc9457.Response
	var hasErrors bool

	r := args.HTTPRequest

	// Extract RFC9457 response using FindErr
	resp, _ = FindErr[*rfc9457.Response](err)
	if resp != nil {
		httpStatus = resp.Status
	}

	// Extract http_status from metadata using ErrValue (type-safe!)
	if httpStatus == 0 {
		httpStatus, _ = ErrValue[int](err, "http_status")
	}

	// Simple nil check
	hasErrors = err != nil

	// TODO There are probably more cases we need to add
	switch {
	case httpStatus == 0 && !hasErrors:
		goto end
	case httpStatus == http.StatusUnprocessableEntity:
		err = apiresp.UnprocessableEntityPayload(r, apiresp.PayloadArgs{
			RFC9457: resp,
		}).NewErr(
			ErrRouteMatchingFailed,
			err,
		)
		goto end
	case hasErrors && httpStatus != 0:
		err = NewErr(
			ErrRouteMatchingFailed,
			// TODO Verify that "matching_request" is appropriate for "location"
			apiresp.CurrentlyUnhandledErrorPayload(r, apiresp.PayloadArgs{
				Location:   "CHANGE ME", // TODO: Use debugging to identify what values are useful here
				HTTPStatus: httpStatus,
				Error:      err,
			}),
			err,
		)
	default:
		err = NewErr(
			ErrRouteMatchingFailed,
			apiresp.InternalServerErrorPayload(r, apiresp.PayloadArgs{}),
			err,
		)
		goto end
	}
end:
	return err
}

func (api *API) tryMatchingRequest(args HandlerHelperArgs) (result pathvars.MatchResult, err error) {
	var te *pathvars.TemplateError
	var pr apiresp.PayloadResult

	r := args.HTTPRequest

	result, err = api.Router.Match(r)
	if errors.As(err, &te) {
		// Choose appropriate error payload based on error type
		if te.ConstraintType() != "" {
			pr = apiresp.ConstraintViolationErrorPayload(r, apiresp.PayloadArgs{TemplateError: te})
		} else {
			pr = apiresp.InvalidURLParameterErrorPayload(r, apiresp.PayloadArgs{TemplateError: te})
		}
		err = pr.NewErr(
			pvtypes.ErrInvalidParameter,
			te.Err, // TODO RESOLVE THIS!
			err,
		)
		goto end
	}
	if errors.Is(err, pathvars.ErrNoMatch) {
		err = apiresp.EndpointNotMatchedPayload(r, apiresp.PayloadArgs{}).NewErr(
			ErrRouteNotMatched,
			err,
		)
		goto end
	}
	if err != nil {
		err = api.handleFailedMatch(err, args)
	}
end:
	if err != nil {
		err = WithErr(err,
			"http_method", r.Method,
			"url_path", r.URL.Path,
		)
	}
	return result, err
}

type SendResponseArgs struct {
	Content     any
	HTTPRequest *http.Request
	APIResponse *apiresp.Response
	Error       error
}

func (api *API) SendSuccessResponse(args SendResponseArgs) {
	// TODO Validate result types and column types
	//      OR MAYBE THAT IS ALREADY BEING HANDLED UPSTREAM?
	args.APIResponse.Send(apiresp.NewResponsePayload(apiresp.ResponsePayloadArgs{
		Content:    args.Content,
		HTTPStatus: http.StatusOK,
		MIMEType:   rfc9457.ApplicationJSON,
	}))
}

func (api *API) SendErrorResponse(args SendResponseArgs) {
	var rp apiresp.ResponsePayload
	var location string
	var found bool

	// Extract ResponsePayload using FindErr
	rp, found = FindErr[apiresp.ResponsePayload](args.Error)
	if !found || rp == nil {
		// Extract location using ErrValue (type-safe!)
		location, _ = ErrValue[string](args.Error, "location")
		rp = apiresp.CurrentlyUnhandledErrorPayload(args.HTTPRequest, apiresp.PayloadArgs{
			Error:    NewErr(ErrNoResponsePayloadFound, "error", args.Error.Error(), args.Error),
			Location: apiresp.LocationType(location),
		}).ResponsePayload
	}
	args.APIResponse.Send(rp)
}
