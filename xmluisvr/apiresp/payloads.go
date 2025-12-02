package apiresp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mikeschinkel/go-rfc9457"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

func NoResultsPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.NoResultsErrorType,
		Title:    "No Results Error",
		Status:   http.StatusNotFound,
		Detail:   use.GetDetail(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      DBQueryLocation,
				Suggestion:    use.GetSuggestion(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

func UnauthorizedPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.UnauthorizedErrorType,
		Title:    "Not Authorized Error",
		Status:   http.StatusUnauthorized,
		Detail:   use.GetDetail(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Suggestion:    use.GetSuggestion(),
				Location:      "",  // TODO: Can we populate this?,
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

func CardinalityMismatchPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()

	suggest := ""
	switch {
	case errors.Is(args.Error, dbpkg.ErrOneRowExpectedZeroReturned):
		suggest = "from 'one?' to 'one'"
	case errors.Is(args.Error, dbpkg.ErrOneRowExpectedManyReturned):
		suggest = "from 'one' (or 'one?') to 'many' (or 'many?')"
	case errors.Is(args.Error, dbpkg.ErrManyRowsExpectedZeroReturned):
		suggest = "from 'many' to 'many?'"
	}

	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.CardinalityMismatchErrorType,
		Title:    "Cardinality mismatch in API configuration",
		Status:   http.StatusInternalServerError,
		Detail:   use.GetError().Error(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Location:      "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
				Suggestion: fmt.Sprintf("Maybe change the cardinality in your API's configuration for route %s %s or revisit your database query to ensure it matched your the specified cardinality.",
					args.EndpointTemplate,
					suggest,
				),
			},
		},
	}))
}

func EndpointNotMatchedPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.EndpointNotMatchedErrorType,
		Title:    "Endpoint Not Matched",
		Status:   http.StatusNotFound,
		Detail:   "Request URL did not match a configured endpoint",
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Location:      "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
				Suggestion:    "",  // TODO: Can we populate this?
			},
		},
	}))
}

func UnprocessableEntityPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	if args.RFC9457 == nil {
		pr = InternalServerErrorPayload(req, args)
		goto end
	}
	pr = use.checkUsage(args.RFC9457)
end:
	return pr
}

func CurrentlyUnhandledErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()

	if use.HTTPStatus == 0 {
		use.HTTPStatus = http.StatusInternalServerError
	}
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.CurrentlyUnhandledErrorType,
		Title:    "Unexpected Server Error",
		Status:   use.GetHTTPStatus(),
		Detail:   fmt.Sprintf("Currently unhandled error: %v.\nPlease %s.", use.GetError(), ReportOnGithubMessageFunc()),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      use.GetLocation(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
				Suggestion:    "",  // TODO: Can we populate this?
			},
		},
	}))
}

func InternalServerErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.InternalServerErrorType,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   fmt.Sprintf(UnexpectedErrorMatchingURLFileOnGithub, ReportOnGithubMessageFunc()),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      use.Location,   // Optional: Use directly, don't call getter
				Suggestion:    use.Suggestion, // Optional: Use directly, don't call getter
				Parameter:     "",             // TODO: Can we populate this?
				ExpectedType:  "",             // TODO: Can we populate this?
				ReceivedValue: "",             // TODO: Can we populate this?
				Constraint:    nil,            // TODO: Can we populate this?
			},
		},
	}))
}

func QueryFailedPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	detail := "Database or API configuration error"
	switch use.GetErrorStyle() {
	case common.DevelopmentStyle:
		detail = fmt.Sprintf("%s; %s", detail, use.GetError().Error())
	case common.PresentationStyle:
		detail = fmt.Sprintf("%s; check logs if you have server access.", detail)
	}

	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.QueryFailedErrorType,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      DBQueryLocation,
				Suggestion:    use.GetSuggestion(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

func InvalidBodyFormatErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()

	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.InvalidBodyFormatErrorType,
		Title:    "Invalid Body Format",
		Status:   http.StatusBadRequest,
		Detail:   use.GetDetail(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      use.GetLocation(),
				Suggestion:    use.GetSuggestion(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

func InvalidDBQueryErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()

	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.InvalidDBQueryErrorType,
		Title:    "Invalid Database Query",
		Status:   http.StatusBadRequest,
		Detail:   use.GetDetail(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      use.GetLocation(),
				Suggestion:    use.GetSuggestion(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

type MissingParameter struct {
	rfc9457.Selector
	Location LocationType
	Expected string
	Received string
	Message  string
}

func MissingParametersPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	var rp ResponsePayload

	use := args.clone()
	missing := use.GetMissingParameters()

	rfc := rfc9457.ResponseArgs{
		Type:     rfc9457.MissingParametersErrorType,
		Status:   http.StatusBadRequest,
		Detail:   " — URL path, query, HTTP body or headers — while attempting to match requested URL. Check server logs if you have access.",
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Location:      "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
				Suggestion:    "",  // TODO: Can we populate this?
			},
		},
	}

	switch len(missing) {
	case 0:
		suggest := fmt.Sprintf("MissingParametersPayload() was called with no missing parameters. This is a bug; please submit an issue or a PR at %s",
			GitHubRepoURL(),
		)
		pr = InternalServerErrorPayload(req, PayloadArgs{
			Suggestion: suggest,
		})
		goto end

	case 1:
		rfc.Title = "Missing required parameter"
		rfc.Detail = rfc.Title + rfc.Detail
		p := missing[0]
		rfc.AddExtension(RFC9457Extension{
			Parameter:     string(p.Selector),
			ExpectedType:  p.Expected,
			ReceivedValue: p.Received,
			Location:      p.Location,
			Constraint:    p.Message,
		})
	default:
		var ext RFC9457Extension
		rfc.Title = "Missing required parameters"
		rfc.Detail = rfc.Title + rfc.Detail
		for _, p := range missing {
			ext.ValidationErrors = append(ext.ValidationErrors, NewValidationError(ValidationErrorArgs{
				Parameter: string(p.Selector),
				Location:  p.Location,
				Expected:  p.Expected,
				Received:  p.Received,
				Message:   p.Message,
			}))
		}
		rfc.AddExtension(ext)
	}
	rp = rfc9457.NewResponse(rfc)
	pr = use.checkUsage(rp)
end:
	return pr
}

func InvalidURLFormatErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()

	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.InvalidURLFormatErrorType,
		Title:    "Invalid URL Format",
		Status:   http.StatusBadRequest,
		Detail:   use.GetDetail(),
		Instance: req.RequestURI,
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Location:      use.GetLocation(),
				Suggestion:    use.GetSuggestion(),
				Parameter:     "",  // TODO: Can we populate this?
				ExpectedType:  "",  // TODO: Can we populate this?
				ReceivedValue: "",  // TODO: Can we populate this?
				Constraint:    nil, // TODO: Can we populate this?
			},
		},
	}))
}

func InvalidURLParameterErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	te := use.GetTemplateError()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.InvalidURLParameterErrorType,
		Title:    "Invalid URL Parameter",
		Instance: req.RequestURI,
		Status:   getHTTPStatusFromFaultSource(te.FaultSource()),
		Detail:   te.Detail(),
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Suggestion:    te.Suggestion,
				Location:      LocationType(te.Location),
				Parameter:     te.Parameter(),
				ExpectedType:  te.ExpectedType(),
				ReceivedValue: te.ReceivedValue(),
				Constraint:    nil,
			},
		},
	}))
}

func ConstraintViolationErrorPayload(req *http.Request, args PayloadArgs) (pr PayloadResult) {
	use := args.clone()
	te := use.GetTemplateError()
	return use.checkUsage(rfc9457.NewResponse(rfc9457.ResponseArgs{
		Type:     rfc9457.ConstraintViolationErrorType,
		Title:    "Constraint Violation",
		Instance: req.RequestURI,
		Status:   getHTTPStatusFromFaultSource(te.FaultSource()),
		Detail:   te.Detail(),
		Extensions: []rfc9457.Extension{
			RFC9457Extension{
				Suggestion:    te.Suggestion,
				Location:      LocationType(te.Location),
				Parameter:     te.Parameter(),
				ExpectedType:  te.ExpectedType(),
				ReceivedValue: te.ReceivedValue(),
				Constraint:    te.ConstraintType(),
			},
		},
	}))
}
