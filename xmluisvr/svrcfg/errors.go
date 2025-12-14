package cfgldr

import (
	"errors"
)

var (
	ErrAPIEndpointHasNoQueryOrFile = errors.New("APIConfig endpoint has no query or query file")
	ErrAPIEndpointMustNotBeEmpty   = errors.New("APIConfig endpoint must not be empty")
	ErrInvalidAPIEndpoint          = errors.New("invalid APIConfig endpoint")
	ErrInvalidAPIEndpointMethod    = errors.New("invalid APIConfig endpoint HTTP method")
	ErrInvalidAPIEndpointPath      = errors.New("invalid APIConfig endpoint path")
	ErrParseFailed                 = errors.New("parse failed")
	ErrReadFailed                  = errors.New("read failed")
)

var (
	ErrAPIParamsMapCannotBeNested     = errors.New("API parameter map cannot be nested JSON")
	ErrAPIParamsMapCannotContainArray = errors.New("API parameter map cannot contain a JSON array")
	ErrAPIParamsMapExpectedObject     = errors.New("API parameter map expects a JSON object")
	ErrAPIParamsMapStringsOnly        = errors.New("API parameter map values must be JSON strings")
	ErrAPIParamsMapDuplicateKey       = errors.New("API parameter map contains a duplicate key")
	ErrAPIParamsMapTrailingData       = errors.New("API parameter map has trailing data after object")
	ErrAPIParamsMapInvalidComments    = errors.New("API parameter map contains invalid comments; must contain only strings")
	ErrAPIParamsIsAnInvalidDataType   = errors.New("API params is an invalid data type")
)

var (
	ErrFailedToLoadAPIConfigFile      = errors.New("failed to load APIConfig config file")
	ErrFailedToUnmarshalAPIConfigFile = errors.New("failed to unmarshal APIConfig config file")
	ErrFailedToLoadDBSchemaFile       = errors.New("failed to load DB schema file")
)

// ErrFailedToReadQueryFile indicates that an SQL file referenced by an endpoint could not be read.
var ErrFailedToReadQueryFile = errors.New("failed to read query file")
var ErrEitherQueryOrQueryFile = errors.New("both query file and query cannot have values")
