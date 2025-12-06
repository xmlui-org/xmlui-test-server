package localsvr

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type HTTPMethod string

const (
	ANYMethod     HTTPMethod = "ANY"
	GETMethod     HTTPMethod = http.MethodGet
	POSTMethod    HTTPMethod = http.MethodPost
	HEADMethod    HTTPMethod = http.MethodHead
	PUTMethod     HTTPMethod = http.MethodPut
	PATCHMethod   HTTPMethod = http.MethodPatch
	DELETEMethod  HTTPMethod = http.MethodDelete
	CONNECTMethod HTTPMethod = http.MethodConnect
	OPTIONSMethod HTTPMethod = http.MethodOptions
	TRACEMethod   HTTPMethod = http.MethodTrace
)

var HTTPMethodRegexpString = "(" +
	GETMethod + "|" +
	POSTMethod + "|" +
	HEADMethod + "|" +
	PUTMethod + "|" +
	PATCHMethod + "|" +
	DELETEMethod + "|" +
	CONNECTMethod + "|" +
	OPTIONSMethod + "|" +
	TRACEMethod +
	")"

var HTTPMethods = []HTTPMethod{
	GETMethod,
	POSTMethod,
	HEADMethod,
	PUTMethod,
	PATCHMethod,
	DELETEMethod,
	CONNECTMethod,
	OPTIONSMethod,
	TRACEMethod,
}
var HttpMethodRegexp = regexp.MustCompile(fmt.Sprintf(`^(%s)$`, HTTPMethodRegexpString))
var ErrHTTPMethodCannotBeEmpty = errors.New("http method can not be empty")
var ErrNotAValidHTTPMethod = errors.New("not a valid HTTP method")
var ErrEmptyHandlingNotSpecified = errors.New("empty handling not specified")

type EmptyHandling int

const (
	UnspecifiedEmptyHandling EmptyHandling = iota
	EmptyOk
	EmptyInvalid
	EmptyRequired
)

func ParseHTTPMethod(m string, eh EmptyHandling) (hm HTTPMethod, err error) {
	var matches []string
	if m == "" {
		switch eh {
		case EmptyOk, EmptyRequired:
			hm = ANYMethod
			goto end
		case EmptyInvalid:
			err = ErrHTTPMethodCannotBeEmpty
			goto end
		case UnspecifiedEmptyHandling:
			fallthrough
		default:
			err = NewErr(ErrEmptyHandlingNotSpecified, "method", m)
			goto end
		}
	}
	matches = HttpMethodRegexp.FindStringSubmatch(strings.ToUpper(m))
	if len(matches) == 0 {
		err = NewErr(ErrNotAValidHTTPMethod, "method", m)
		goto end
	}
	hm = HTTPMethod(matches[1])
end:
	return hm, err
}
