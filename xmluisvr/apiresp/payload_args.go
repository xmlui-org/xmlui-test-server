package apiresp

import (
	"errors"
	"unsafe"

	"github.com/mikeschinkel/go-pathvars"
	"github.com/mikeschinkel/go-rfc9457"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"

	. "github.com/mikeschinkel/go-doterr"
)

var paType = (*PayloadArgs)(nil)

type propInfo struct {
	name     string
	zeroFunc func(*PayloadArgs) bool
}

var payloadArgsProps = map[uintptr]propInfo{
	unsafe.Offsetof(paType.Suggestion):        {"Suggestion", func(args *PayloadArgs) bool { return args.Suggestion == "" }},
	unsafe.Offsetof(paType.Location):          {"Location", func(args *PayloadArgs) bool { return args.Location == "" }},
	unsafe.Offsetof(paType.HTTPStatus):        {"HTTPStatus", func(args *PayloadArgs) bool { return args.HTTPStatus == 0 }},
	unsafe.Offsetof(paType.ErrorStyle):        {"ErrorStyle", func(args *PayloadArgs) bool { return args.ErrorStyle == "" }},
	unsafe.Offsetof(paType.Error):             {"Error", func(args *PayloadArgs) bool { return errors.Is(args.Error, ErrUnspecifiedError) }},
	unsafe.Offsetof(paType.MissingParameters): {"MissingParameters", func(args *PayloadArgs) bool { return len(args.MissingParameters) == 0 }},
	unsafe.Offsetof(paType.DBQuery):           {"DBQuery", func(args *PayloadArgs) bool { return args.DBQuery == "" }},
	unsafe.Offsetof(paType.EndpointTemplate):  {"EndpointTemplate", func(args *PayloadArgs) bool { return args.EndpointTemplate == "" }},
	unsafe.Offsetof(paType.Detail):            {"Detail", func(args *PayloadArgs) bool { return args.Detail == "" }},
	unsafe.Offsetof(paType.RFC9457):           {"RFC9457", func(args *PayloadArgs) bool { return args.RFC9457 == nil }},
	unsafe.Offsetof(paType.TemplateError):     {"TemplateError", func(args *PayloadArgs) bool { return args.TemplateError == nil }},
}

type PayloadArgs struct {
	Suggestion        string
	Location          LocationType
	HTTPStatus        int
	ErrorStyle        common.ErrorStyle
	Error             error
	MissingParameters []MissingParameter
	DBQuery           sqlparams.QueryString
	EndpointTemplate  string
	Detail            string
	RFC9457           *rfc9457.Response
	TemplateError     *pathvars.TemplateError
	PVE               *pathvars.TemplateError
	propsUsed         map[uintptr]struct{}
	errs              []error
}

func (args *PayloadArgs) clone() *PayloadArgs {
	pa := *args
	pa.propsUsed = make(map[uintptr]struct{})
	if pa.Error == nil {
		pa.Error = ErrUnspecifiedError
	}
	if pa.MissingParameters == nil {
		pa.MissingParameters = make([]MissingParameter, 0)
	}
	return &pa
}
func (args *PayloadArgs) useProp(propId uintptr) {
	if args.propsUsed == nil {
		args.errs = append(args.errs, NewErr(
			ErrNotUsingClonedPayloadArgs,
			"property_id", propId,
		))
		goto end
	}
	args.propsUsed[propId] = struct{}{}
end:
	return
}

// checkUsage checks to see if the developer passed a disallowed non-zero value
// for a PayloadArgs property whose value will not be used by the function OR did
// not pass non-zero PayloadArgs that were use and generates a fatal logging
// error if so. This makes sure the developer is made away that the function will
// not use any of these "disallowed" arg rather than allowing a potentially
// subtle bug to remain in the source code.
func (args *PayloadArgs) checkUsage(rp ResponsePayload) PayloadResult {
	var errs []error
	var err error

	for propId, info := range payloadArgsProps {
		_, used := args.propsUsed[propId]
		switch {
		case used && info.zeroFunc(args):
			errs = append(errs, NewErr(
				ErrZeroValueForExpectedProperty,
				"property_name", info.name,
			))
		case !used && !info.zeroFunc(args):
			errs = append(errs, NewErr(
				ErrNonZeroValueForUnexpectedProperty,
				"property_name", info.name,
			))
		}
	}
	// Add these errors to ones collected by useProp()
	err = CombineErrs(append(args.errs, errs...))
	if err != nil {
		err = NewErr(
			ErrInvalidPropertyUsage,
			err,
		)
	}
	return PayloadResult{
		ResponsePayload: rp,
		Error:           err,
	}
}

func (args *PayloadArgs) GetHTTPStatus() int {
	args.useProp(unsafe.Offsetof(args.HTTPStatus))
	return args.HTTPStatus
}

func (args *PayloadArgs) GetErrorStyle() common.ErrorStyle {
	args.useProp(unsafe.Offsetof(args.ErrorStyle))
	return args.ErrorStyle
}

func (args *PayloadArgs) GetError() error {
	args.useProp(unsafe.Offsetof(args.Error))
	return args.Error
}

func (args *PayloadArgs) GetLocation() LocationType {
	args.useProp(unsafe.Offsetof(args.Location))
	return args.Location
}

func (args *PayloadArgs) GetRFC9457() *rfc9457.Response {
	args.useProp(unsafe.Offsetof(args.RFC9457))
	return args.RFC9457
}

func (args *PayloadArgs) GetSuggestion() string {
	args.useProp(unsafe.Offsetof(args.Suggestion))
	return args.Suggestion
}

func (args *PayloadArgs) GetDetail() string {
	args.useProp(unsafe.Offsetof(args.Detail))
	return args.Detail
}

func (args *PayloadArgs) GetMissingParameters() []MissingParameter {
	args.useProp(unsafe.Offsetof(args.MissingParameters))
	return args.MissingParameters
}

func (args *PayloadArgs) GetDBQuery() sqlparams.QueryString {
	args.useProp(unsafe.Offsetof(args.DBQuery))
	return args.DBQuery
}

func (args *PayloadArgs) GetTemplateError() *pathvars.TemplateError {
	args.useProp(unsafe.Offsetof(args.TemplateError))
	return args.TemplateError
}
