package apipkg

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mikeschinkel/go-pathvars/pvtypes"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"

	"github.com/mikeschinkel/go-jsonxtractr"
	"github.com/mikeschinkel/go-rfc9457"

	"github.com/mikeschinkel/go-pathvars"
)

// ParseEndpoints converts a slice of configuration endpoint definitions
// into parsed Endpoint structs. Each endpoint is validated during parsing.
func ParseEndpoints(cfgEPs []*cfgldr.APIEndpointV2, basePath localsvr.URLPath, db dbpkg.Database) (eps []*Endpoint, err error) {
	var errs []error
	eps = make([]*Endpoint, len(cfgEPs))
	for i, cfgEP := range cfgEPs {
		eps[i], err = ParseEndpoint(cfgEP, basePath, db)
		if err != nil {
			errs = append(errs, NewErr(
				ErrInvalidAPIEndpointParameter,
				err,
			))
		}
	}
	err = CombineErrs(errs)
	if err != nil {
		err = NewErr(
			ErrParsingFailed,
			ErrParsingOfMultipleEndpointsFailed,
			CombineErrs(errs),
		)
	}
	return eps, err
}

func ParseQuery(query localsvr.QueryString, db dbpkg.Database) (pq sqlparams.ParsedQuery, err error) {

	// QueryFileExt is a proxy for Query Type.
	// TODO: Maybe add a first-class QueryType later
	qt := strings.ToLower(db.QueryFileExt())
	switch qt {
	case ".sql":
		pq, err = sqlparams.ParseSQL(sqlparams.SQLQuery(query), db.GetFormatParamFunc())
	default:
		err = NewErr(ErrQueryTypeParsingNotYetSupported, "query_type", qt)
	}
	return pq, err
}

// ParseEndpoint converts a configuration endpoint into a parsed Endpoint struct.
// It validates all fields including HTTP method, URL path, parameters, and SQL configuration.
func ParseEndpoint(cfg *cfgldr.APIEndpointV2, basePath localsvr.URLPath, db dbpkg.Database) (ep *Endpoint, err error) {
	var errs []error
	var q string
	ep = &Endpoint{
		Description: cfg.Description,
	}
	q, err = cfg.GetQuery()
	if err == nil {
		ep.ParsedQuery, err = ParseQuery(localsvr.QueryString(q), db)
	}
	errs = AppendErr(errs, err)

	// TODO: Allow or disallow defining endpoints without explicitly specifying a method? Maybe we should require "ANY"?
	ep.method, err = localsvr.ParseHTTPMethod(cfg.Method, localsvr.EmptyOk)
	errs = AppendErr(errs, err)
	var relPath *pathvars.ParsedTemplate
	relPath, err = pathvars.ParseTemplate(cfg.Path)
	errs = AppendErr(errs, err)

	// TODO Allow or disallow root-based URLs that ignore basepath; which to choose?
	//      Need override setting to explicitly allow
	// 			UNTIL THEN, we strip any leading slash (`/`)
	// Only normalize and use relPath if parsing succeeded
	if relPath != nil {
		relPath.Normalize()
		ep.path = pathvars.Template(fmt.Sprintf("%s/%s", basePath, relPath))
		// Only parse params if we have a valid path
		ep.Params, err = ParseEndpointParams(cfg.Params, ep.path)
	} else {
		// Path parsing failed - use empty params to avoid cascading errors
		ep.Params = []EndpointParam{}
	}
	errs = AppendErr(errs, err)
	ep.pathParsed = true
	ep.Cardinality, err = sqlparams.ParseCardinality(cfg.Cardinality)
	errs = AppendErr(errs, err)
	ep.RowType, err = sqlparams.ParseDBRowType(cfg.RowType)
	errs = AppendErr(errs, err)
	ep.ColumnTypes, err = sqlparams.ParseColumnTypes(cfg.ColumnTypes)
	errs = AppendErr(errs, err)
	err = CombineErrs(errs)
	if err != nil {
		ep = nil
		err = NewErr(
			ErrParsingFailed,
			ErrEndpointParsingFailed,
			err,
			"endpoint", cfg.Endpoint(),
		)
	}
	return ep, err
}

// EndPointString represents a string representation of an HTTP endpoint (e.g., "GET /users/:id").
type EndPointString string

// Endpoint represents a parsed API endpoint configuration with all validation complete.
// It contains the HTTP method, URL path, SQL query, parameters, and response formatting options.
type Endpoint struct {
	Description string                // Human-readable description of the endpoint
	ParsedQuery sqlparams.ParsedQuery // Query to execute parsed by sqlparams.ParseBytes()
	//queryFilepath dt.Filepath            // Resolved absolute path to SQL file
	Params      []EndpointParam        // Parameters that can be extracted from requests
	Cardinality sqlparams.Cardinality  // Expected number of result rows (one, many, etc.)
	RowType     sqlparams.DBRowType    // Format for returning results (json, columns, etc.)
	ColumnTypes []sqlparams.DBDataType // Expected data types for result columns
	method      localsvr.HTTPMethod    // HTTP method (GET, POST, etc.)
	path        pathvars.Template      // URL path pattern with parameter placeholders
	pathParsed  bool
}

func (ep *Endpoint) GetBodyValuesMap(r io.Reader, selectors []jsonxtractr.Selector) (valuesMap jsonxtractr.ValuesMap, notFound []jsonxtractr.Selector, err error) {
	dbq := ep.ParsedQuery
	// Get the pathValuesMap needed for the SQL query from the URL path and query variables
	valuesMap, notFound, err = jsonxtractr.ExtractValuesFromReader(r, selectors)
	if errors.Is(err, jsonxtractr.ErrJSONValueSelectorCannotBeEmpty) {
		err = nil
		goto end
	}
	if err != nil {
		// TODO: Replace with an Response
		err = NewErr(ErrExtractingFromReader,
			err,
			"endpoint", ep.Endpoint(),
			"sql_query", dbq.QueryString(),
			"sql_params", dbq.Parameters(),
			"body_matched", valuesMap,
			"not_matched", notFound,
		)
		goto end
	}
end:
	return valuesMap, notFound, err
}

type ParameterValuesArgs struct {
	ValuesMap  pvtypes.ValuesMap
	BodyReader io.Reader
	Headers    http.Header // TODO: Not yet supported
	Database   dbpkg.Database
}

func (ep *Endpoint) parameterTypeMap() (tm map[string]pathvars.PVDataType) {
	// Create a map of parameter names to their types for quick lookup
	tm = make(map[string]pathvars.PVDataType)
	for _, param := range ep.Params {
		tm[string(param.Name)] = param.Type
	}
	return tm
}

func (ep *Endpoint) GetParameterValues(args ParameterValuesArgs) (queryValues []any, missing []apiresp.MissingParameter, err error) {
	var pathValuesMap pvtypes.ValuesMap
	var namesNotFound []pathvars.Identifier
	var jsonValuesMap jsonxtractr.ValuesMap
	var notFound []jsonxtractr.Selector
	var selectors []sqlparams.Selector
	var epParams []EndpointParam

	dbq := ep.ParsedQuery
	parameters := dbq.Parameters()
	occurrences := dbq.Occurrences()

	// Get the pathValuesMap needed for the SQL query from the URL path and query variables
	ids := pathvars.Identifiers(parameters.Identifiers())
	pathValuesMap, namesNotFound = args.ValuesMap.GetValues(ids)

	// Get the selectors to search JSON - only dotted selectors (e.g., task.title) should
	// be extracted from the body. Query parameters and path parameters are already in pathValuesMap.
	selectors = parameters.DottedSelectors()
	// NOTE: We do NOT add namesNotFound to selectors here because those are simple parameter names
	// (not dotted) that should come from path/query, not the JSON body.

	switch {
	case args.BodyReader != nil:
		// We got a reader for the JSON body - extract dotted body parameters
		jsonValuesMap, notFound, err = ep.GetBodyValuesMap(args.BodyReader, jsonxtractr.ToSelectors(selectors))
		if err != nil {
			err = NewErr(ErrExtractingJSONBodyValues, err)
			goto end
		}
		// Combine path/query params not found with body params not found
		notFound = combineStringsAsY(namesNotFound, notFound)
	default:
		// We did NOT get a reader for the JSON body
		// Any parameters not found in path/query are missing
		notFound = combineStringsAsY(namesNotFound, []jsonxtractr.Selector{})
	}

	// Build queryValues array based on ALL occurrences (including duplicates)
	// For SQL binding, we need one value per placeholder, even if the same parameter appears multiple times
	queryValues = make([]any, len(occurrences))
	for i, token := range occurrences {
		qv, ok := pathValuesMap.Get(pathvars.Identifier(token.Name))
		if ok {
			queryValues[i] = qv
			continue
		}
		if args.BodyReader == nil {
			// We did not get a body ready so no jsonValuesMap to look at
			continue
		}
		qv, ok = jsonValuesMap[jsonxtractr.Selector(token.Name)]
		if ok {
			queryValues[i] = qv
			continue
		}
	}

	// Convert values based on parameter types for SQL compatibility
	queryValues = ep.convertValuesForSQL(occurrences.Parameters(), queryValues, args)

	// Build list of missing parameters for error reporting
	epParams = EndpointParams(ep.Params).FilterByNames(jsonxtractr.Selectors(notFound).Strings())
	missing = make([]apiresp.MissingParameter, len(epParams))
	for i, p := range epParams {
		missing[i] = apiresp.MissingParameter{
			// TODO: Converting an Identifier to a Selector. p.Name should probably be a Selector
			Selector: rfc9457.Selector(p.Name),
			Location: apiresp.LocationType(p.Location),
			Expected: "", // TODO Can we populate this?
			Received: "", // TODO Can we populate this?
			Message:  "", // TODO Can we populate this?
		}
	}
end:
	return queryValues, missing, err
}

//// getParameterQueryValue retrieves a parameter value from the unified valuesMap.
//// The valuesMap now contains ALL parameters (path, template query, and Params-defined query).
//// Returns the first value if the parameter exists, empty string otherwise.
//func (ep *Endpoint) getParameterQueryValue(epp EndpointParam, valuesMap pvtypes.ValuesMap) (value string) {
//	var rawValue any
//	var strValue string
//	var ok bool
//
//	// Skip path params - already validated by Router.Match()
//	if epp.Location == pathvars.PathLocation {
//		goto end
//	}
//
//	// Skip body params - handled separately by GetParameterValues
//	if epp.Location != pathvars.QueryLocation {
//		goto end
//	}
//
//	// Get value from unified valuesMap (contains path + template query + Params-defined query)
//	// All query params are now stored as strings (first value from HTTP query)
//	rawValue, ok = valuesMap.Get(epp.Name)
//	if !ok {
//		goto end
//	}
//
//	// Convert to string (all query params should be strings)
//	strValue, ok = rawValue.(string)
//	if ok {
//		value = strValue
//		goto end
//	}
//end:
//	return value
//}

// ValidateQueryParameters validates ALL HTTP query parameters defined in ep.Params against
// the actual HTTP request query string. This is separate from SQL parameter extraction.
// Path parameters are already validated by Router.Match() and are skipped here.
// Returns a TemplateError if validation fails, allowing consistent error handling with
// template-defined parameter validation.
func (ep *Endpoint) ValidateQueryParameters(matchResult pathvars.MatchResult) (err error) {
	type paramValidationError struct {
		param    pathvars.Parameter
		value    string
		validErr error
		location pathvars.LocationType
	}
	var validationErrors []paramValidationError
	var parsedTemplate *pathvars.ParsedTemplate
	var parsedQuery *pathvars.ParsedQuery

	// Get the parsed query from the matched route's template
	parsedTemplate = matchResult.Route.ParsedTemplate
	if parsedTemplate != nil {
		parsedQuery = parsedTemplate.ParsedQuery()
	}

	// If no parsedQuery, nothing to validate
	if parsedQuery == nil {
		return nil
	}

	valuesMap := matchResult.ValuesMap()

	// Build a ValuesMap with ONLY user-provided parameters per ADR-018
	// This means: path params + query params the user actually sent (not defaults)
	// We exclude optional query params that got default values but weren't in the HTTP request
	userProvidedParams := pvtypes.NewValuesMap(parsedQuery.Len())

	// Add path parameters from valuesMap (these are always user-provided via the URL path)
	pathParamNames := make(map[pathvars.Identifier]bool)
	for _, epParam := range ep.Params {
		if epParam.Location == pathvars.PathLocation {
			pathParamNames[epParam.Name] = true
		}
	}

	for name, value := range valuesMap.Iterator() {
		if pathParamNames[name] {
			userProvidedParams.Set(name, value)
		}
	}

	// Add ONLY query parameters that were in the HTTP request (not template defaults)
	// parsedQuery contains exactly what the user sent, so this implements ADR-018's
	// "only show parameters provided by the user" rule
	for paramName, values := range parsedQuery.Iterator() {
		if len(values) > 0 {
			userProvidedParams.Set(pathvars.Identifier(paramName), values[0])
		}
	}

	// Validate ALL query params from ep.Params against the HTTP request query string
	for _, epParam := range ep.Params {
		// Skip non-query parameters
		if epParam.Location != pathvars.QueryLocation {
			continue
		}

		// Get the value directly from parsedQuery (HTTP request query string)
		paramName := string(epParam.Name)
		values, found := parsedQuery.Get(paramName)
		if !found || len(values) == 0 {
			// Parameter not provided in HTTP request - skip validation
			// (missing required params are handled elsewhere)
			continue
		}
		value := values[0] // Use first value if multiple provided

		// Convert EndpointParam to pathvars.Parameter for validation
		paramType := epParam.Type
		if paramType == pathvars.UnspecifiedDataType {
			paramType, _ = pathvars.ParseParameterDataType(string(epParam.Name), string(epParam.Type.Slug()))
		}

		param := pathvars.NewParameter(pathvars.ParameterArgs{
			NameProps:   epParam.Props,
			Location:    epParam.Location,
			DataType:    paramType,
			Constraints: epParam.Constraints,
			Original:    epParam.RawValue(),
		})

		// Validate the parameter value
		validErr := param.Validate(value)
		if validErr != nil {
			validationErrors = append(validationErrors, paramValidationError{
				param:    param,
				value:    value,
				validErr: validErr,
				location: epParam.Location,
			})
		}
	}

	// If we have validation errors, wrap them in a TemplateError
	if len(validationErrors) > 0 {
		// Return the first error (similar to how ParsedTemplate handles it)
		ve := validationErrors[0]
		exampleURL := parsedTemplate.Example(&pvtypes.ExampleArgs{
			ProblematicParam:   ve.param,
			UserProvidedParams: &userProvidedParams,
			ValidationErr:      ve.validErr,
		})
		err = pathvars.NewTemplateError(ve.validErr, pathvars.TemplateErrorArgs{
			Endpoint:   string(ep.path),
			Example:    exampleURL,
			Source:     string(ep.path),
			Location:   ve.location,
			Parameter:  ve.param,
			Suggestion: ve.param.ErrorSuggestion(ve.validErr, ve.value, exampleURL),
		})
	}

	return err
}

// ParsePathVarParameters converts endpoint parameters into pathvars.Parameter instances
// for use with the routing system. This enables path parameter extraction and validation.
func (ep *Endpoint) ParsePathVarParameters() (params []pathvars.Parameter, err error) {
	var errs []error
	params = make([]pathvars.Parameter, 0, len(ep.Params))
	for i, p := range ep.Params {
		var dataType pathvars.PVDataType
		props := p.Props
		if props.DataType != nil {
			dataType = *props.DataType
		}
		if dataType == pathvars.UnspecifiedDataType {
			dataType, err = pathvars.ParseParameterDataType(string(props.Name), string(p.Type.Slug()))
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if p.RawValue() == "" {
			// TODO Remove this after we ensure RawValue is set
			panic("PARAMETER RAW VALUE NOT SET")
		}
		params = append(params, pathvars.NewParameter(pathvars.ParameterArgs{
			Position:    i,
			NameProps:   props,
			Location:    p.Location,
			DataType:    dataType,
			Constraints: p.Constraints,
			Original:    p.RawValue(),
		}))
	}
	return params, CombineErrs(errs)
}

//// GetQuery returns the SQL query for this endpoint, loading from a file if necessary.
//// If QueryFile is specified, it loads the SQL from the file relative to the provided directory.
//// Otherwise, it returns the inline Query string.
//func (ep *Endpoint) GetQuery(dir dt.DirPath) (q localsvr.QueryString, queryFile dt.Filepath, err error) {
//	panic("FIX THIS")
//	return q, "", err
//}

// Endpoint returns a string representation of the endpoint in "METHOD /path" format.
func (ep *Endpoint) Endpoint() EndPointString {
	return EndPointString(fmt.Sprintf("%s %s",
		strings.ToUpper(string(ep.method)),
		ep.path),
	)
}

// Path returns the URL path pattern for this endpoint.
func (ep *Endpoint) Path() pathvars.Template {
	return ep.path
}

// Method returns the HTTP method for this endpoint, with ANY method converted to empty string.
func (ep *Endpoint) Method() (m localsvr.HTTPMethod) {
	m = ep.RawMethod()
	if m == localsvr.ANYMethod {
		m = ""
		goto end
	}
end:
	return m
}

// RawMethod returns the raw HTTP method without any processing.
func (ep *Endpoint) RawMethod() localsvr.HTTPMethod {
	return ep.method
}

// convertValuesForSQL converts parameter values to SQL-compatible types.
// Currently handles boolean to integer conversion for SQLite compatibility.
func (ep *Endpoint) convertValuesForSQL(parameters sqlparams.Parameters, values []any, args ParameterValuesArgs) (_ []any) {
	var converted []any

	// Create a map of parameter names to their types for quick lookup
	tm := ep.parameterTypeMap()

	// Convert values based on their parameter types
	converted = make([]any, len(values))
	for i, param := range parameters {
		value := values[i]
		if value == nil {
			converted[i] = value
			continue
		}

		// Look up the parameter type
		dataType, exists := tm[string(param.Name)]
		if !exists {
			// If type not found, keep original value
			converted[i] = value
			continue
		}
		// Convert if needed
		converted[i] = convertValueForSQL(value, dataType, args)
	}
	return converted
}

func convertValueForSQL(value any, dt pathvars.PVDataType, args ParameterValuesArgs) any {
	switch dt {
	case pathvars.BooleanType:
		value = args.Database.ConvertValue(value, sqlparams.IntegerDBDataType)
	}
	return value
}
