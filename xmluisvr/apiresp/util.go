package apiresp

import (
	"encoding/json/jsontext"
	"net/http"

	cliutil "github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-pathvars"
)

// Format JSON is a pretty manner
func prettifyJSON(responseJSON []byte) (prettyJSON jsontext.Value) {
	var err error

	prettyJSON = responseJSON
	err = prettyJSON.Indent(jsontext.WithIndent("  "))
	if err != nil {
		cliutil.Errorf("Error prettifying JSON for logging: %v", err)
		cliutil.Errorf("Raw response: %s", string(responseJSON))
		goto end
	}
end:
	return prettyJSON
}

func getHTTPStatusFromFaultSource(fs pathvars.FaultSource) int {
	switch fs {
	case pathvars.ClientFaultSource:
		return http.StatusUnprocessableEntity
	case pathvars.ServerFaultSource:
		fallthrough
	default:
		return http.StatusServiceUnavailable
	}
}
