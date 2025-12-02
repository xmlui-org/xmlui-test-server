package apiresp

import (
	"fmt"

	"github.com/mikeschinkel/go-rfc9457"
)

type PayloadResult struct {
	ResponsePayload ResponsePayload
	Error           error
}

func (r PayloadResult) NewErr(parts ...any) error {
	var errs []any
	// If generating the payload generated an error, show it first
	if r.Error != nil {
		errs = append(errs, r.Error)
	}
	// Add the ResponsePayload as an error so it can be found with FindErr
	if r.ResponsePayload != nil {
		errs = append(errs, r.ResponsePayload)
	}
	// Add all the parts passed in
	errs = append(errs, parts...)

	// Now make a new error
	return NewErr(errs...)
}

type ResponsePayload interface {
	ResponsePayload()
	HTTPStatusCode() int
	MIMEType() rfc9457.MIMEType
	error
}

var _ ResponsePayload = (*responsePayload)(nil)
var _ rfc9457.ContentGetter = (*responsePayload)(nil)

type responsePayload struct {
	content    any
	httpStatus int
	mimeType   rfc9457.MIMEType
}

type ResponsePayloadArgs struct {
	Content    any
	HTTPStatus int
	MIMEType   rfc9457.MIMEType
}

func NewResponsePayload(args ResponsePayloadArgs) ResponsePayload {
	return &responsePayload{
		content:    args.Content,
		httpStatus: args.HTTPStatus,
		mimeType:   args.MIMEType,
	}
}

func (rp responsePayload) Content() any {
	return rp.content
}

func (responsePayload) ResponsePayload() {}

func (rp responsePayload) HTTPStatusCode() int {
	return rp.httpStatus
}

func (rp responsePayload) MIMEType() rfc9457.MIMEType {
	return rp.mimeType
}
func (rp responsePayload) Error() string {
	return fmt.Sprintf("responsePayload\ncontent=%v\nhttp_status=%d\nmime_type=%v",
		rp.content,
		rp.httpStatus,
		rp.mimeType,
	)
}
