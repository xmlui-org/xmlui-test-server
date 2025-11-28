package apipkg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mikeschinkel/go-pathvars"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"

	. "github.com/mikeschinkel/go-doterr"
)

var (
	ErrInvalidParameter                = errors.New("invalid parameter")
	ErrInvalidParameterDataType        = errors.New("invalid parameter data type")
	ErrUnspecifiedParameterDataType    = errors.New("unspecified parameter data type")
	ErrParameterHasNoLeadingIdentifier = errors.New("parameter has no leading identifier")

	ErrInvalidEndpointParam           = errors.New("invalid endpoint param")
	ErrFailedToNormalizeEndpointParam = errors.New("failed to normalize endpoint param")

	// ErrMismatchedParameterDataType occurs when parameter type is defined in path and
	// also in params but they are not the same.
	ErrMismatchedParameterDataType = errors.New("mismatched parameter data type")

	// ErrMismatchedParameterConstraints occurs when constraints are defined in path and
	// also in params but they are not the same.
	ErrMismatchedParameterConstraints = errors.New("mismatched parameter constraint ")
)

// ParseEndpointParams converts configuration parameters into EndpointParam structs.
// It handles type conversion and validation for each parameter definition.
func ParseEndpointParams(cfgParams cfgldr.APIParamsMapper, epPath pathvars.Template) (epParams []EndpointParam, err error) {
	var errs []error
	var apiParams cfgldr.APIParamsV1
	var pathVars []pathvars.ParamVar
	var pvLookup map[pathvars.Identifier]pathvars.ParamVar
	var ok bool

	// Add any parameters that are defined ih the URL path but not lists in the array
	// of params.
	pathVars, err = pathvars.ParseParamsInTemplate(epPath)
	if err != nil {
		errs = append(errs, NewErr(
			"url_template", epPath,
			err,
		))
	}

	apiParams, ok = cfgParams.(cfgldr.APIParamsV1)
	if !ok {
		errs = append(errs, NewErr(
			ErrInvalidEndpointParam,
			ErrCannotTypeAssert,
			"from_type", fmt.Sprintf("%T", cfgParams),
			"to_type", fmt.Sprintf("%T", (cfgldr.APIParamsV1)(nil)),
			"parameter_value", cfgParams,
		))
	}
	if len(errs) != 0 {
		// Convert those two errors causes into a single cause, and bail
		err = CombineErrs(errs)
		goto end
	}
	epParams = make([]EndpointParam, 0, len(pathVars)+len(apiParams))

	// First loop through all the path vars to see if we need to add any from the path
	// vars to the slice of APIParamV1.
	//paramsMap = apiParams.Map()
	pvLookup = make(map[pathvars.Identifier]pathvars.ParamVar, len(pathVars))
	for _, pv := range pathVars {
		epParams = append(epParams, EndpointParam{
			Props:       pv.NameSpecProps,
			Type:        pv.Type,
			Location:    pv.Location,
			Constraints: pv.Constraints,
		})
		pvLookup[pv.Name] = pv
	}

	// Now loop through all the APIParamsV1 and see if there are any vars not in the
	// path string, update their Location and/or Constraints, then add them to the
	// epParams. Also check to make sure that there are not conflicting types nor
	// conflicting constraints
	for _, p := range apiParams {
		var pv pathvars.ParamVar

		// Path var use-type is authoritative so assign the use-type from the path var to
		var epp EndpointParam
		epp, err = ParseEndpointParam(p.NameSpec, p)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		pv, ok = pvLookup[epp.Name]
		if ok {
			// "We already got that one"
			// (said with a French accent)
			continue
		}

		err = epp.normalize(pv)
		if err != nil {
			errs = append(errs, NewErr(
				ErrInvalidEndpointParam,
				ErrFailedToNormalizeEndpointParam,
				"type_in_param_list", p.Type,
			))
			continue
		}
		epParams = append(epParams, epp)
	}

	err = CombineErrs(errs)

end:
	return epParams, err
}

func (epp *EndpointParam) normalize(pv pathvars.ParamVar) (err error) {
	var errs []error

	// Path var use-type is authoritative so assign the use-type from the path var to
	// the APIParamV1.
	switch {
	case pv.Location != "":
		// Parameter is in path template - use its location
		epp.Location = pv.Location
	default:
		// Parameter is NOT in path template (Params-defined only) - default to QueryLocation
		epp.Location = pathvars.QueryLocation
	}
	errs = AppendErr(errs, epp.normalizeDataType(pv))
	errs = AppendErr(errs, epp.normalizeConstraints(pv))
	err = CombineErrs(errs)
	if err != nil {
		err = WithErr(err,
			ErrFailedToNormalizeEndpointParam,
			"parameter_name", pv.Name,
		)
	}
	return err
}

func (epp *EndpointParam) normalizeConstraints(pv pathvars.ParamVar) (err error) {
	eppConstraints := pathvars.Constraints(epp.Constraints).String()
	switch {
	case len(pv.Constraints) != 0 && eppConstraints == "":
		// If path var has constraints and APIParamV1 had no, transfer to APIParamV1
		epp.Constraints = pv.Constraints
	case len(pv.Constraints) != 0 && eppConstraints != "":
		// If both path var has constraints and APIParamV1 has constraints, make sure
		// they are the same, otherwise error.
		pvConstraints := pathvars.Constraints(pv.Constraints).String()
		if pvConstraints != eppConstraints {
			err = NewErr(
				ErrMismatchedParameterConstraints,
				"path_constraints", pvConstraints,
				"param_constraints", eppConstraints,
			)
			goto end
		}
		epp.Constraints = pv.Constraints
	}
end:
	return err
}

func (epp *EndpointParam) normalizeDataType(pv pathvars.ParamVar) (err error) {
	switch {
	case epp.Type == pathvars.UnspecifiedDataType:
		if pv.Type == epp.Type {
			// Type not available anywhere
			err = NewErr(ErrUnspecifiedParameterDataType)
			goto end
		}
		epp.Type = pv.Type

	case pv.Type == pathvars.UnspecifiedDataType:
		// epp.Type already has a type so do nothing

	case pv.Type != epp.Type:
		// Both pv.Type and epp.Type specified, but they are mismatched
		// User provided inconsistent types
		err = NewErr(
			ErrMismatchedParameterDataType,
			"type_in_path", pv.Type,
		)
	}
end:
	return err
}

// ParseEndpointParam converts a configuration parameter into a validated
// EndpointParam. It handles both APIParamV1 and APIParamsMapValue formats,
// parsing the parameter specification and validating all constraints.
func ParseEndpointParam(nameSpec string, cfg cfgldr.APIParam) (p EndpointParam, err error) {
	var props *pathvars.NameSpecProps
	var cc []pathvars.Constraint
	var dt pathvars.PVDataType
	param, ok := cfg.(cfgldr.APIParamV1)
	if !ok {
		paramSpec, ok := cfg.(cfgldr.APIParamsMapValue)
		if !ok {
			err = NewErr(ErrInvalidAPIEndpointParameter, ErrCannotTypeAssert,
				"from_type=", fmt.Sprintf("%T", cfg),
				"to_type", fmt.Sprintf("%T", (*cfgldr.APIParamV1)(nil)),
				"parameter_value", cfg,
			)
			goto end
		}
		param, err = cfgldr.ParseAPIParamV1(nameSpec, string(paramSpec))
	}
	if err != nil {
		goto end
	}
	props, err = pathvars.ParseNameSpecProps(param.NameSpec)
	if err != nil {
		goto end
	}
	if props == nil {
		// Added this here because Goland flags props.DataType as possibly being null. I
		// don't see how it could be possible, but maybe Goland knows something I don't?
		panic(fmt.Sprintf("NameSpecProps are nil when err is also nil; spec=%s", param.NameSpec))
	}
	if props.DataType != nil {
		dt = *props.DataType
	}
	if param.Type != "" {
		dt, err = pathvars.ParsePVDataType(param.Type)
		if err != nil {
			goto end
		}
		cc, err = pathvars.ParseConstraints(param.Constraints, dt)
		if err != nil {
			goto end
		}
		p = NewEndpointParam(EndpointParamArgs{
			Props:       *props,
			Type:        dt,
			Constraints: cc,
			RawValue:    param.String(),
		})
	}
end:
	return p, err
}

type Props = pathvars.NameSpecProps

type EndpointParams []EndpointParam

func (eps EndpointParams) FilterByNames(names []string) (out []EndpointParam) {
	var namesRegexp *regexp.Regexp
	if len(names) == 0 {
		goto end
	}
	out = make([]EndpointParam, len(eps))
	namesRegexp = regexp.MustCompile(fmt.Sprintf("^(%s)$", strings.Join(names, "|")))
	for i, ep := range eps {
		if !namesRegexp.MatchString(string(ep.Name)) {
			continue
		}
		out[i] = ep
	}
end:
	return out
}

// EndpointParam represents a parameter that can be extracted from HTTP requests
// and used in SQL query execution. Parameters can come from URL path segments,
// query strings, or JSON request bodies.
type EndpointParam struct {
	Props
	Type        pathvars.PVDataType   // Data type for validation and conversion
	Location    pathvars.LocationType // How the parameter is used (path, query, body, header)
	Constraints []pathvars.Constraint // Validation constraints (min/max, regex, etc.)
	nameSpec    pathvars.PVNameSpec
}

func (epp *EndpointParam) NameSpec() (ns pathvars.PVNameSpec) {
	if epp.nameSpec == "" {
		epp.nameSpec = pathvars.PVNameSpec(epp.Props.String())
	}
	return epp.nameSpec
}

func (epp *EndpointParam) RawValue() (s string) {
	return epp.Props.RawValue
}
func (epp *EndpointParam) HasProps() bool {
	props := epp.Props
	return props.Name != "" && props.RawValue != ""
}

func (epp *EndpointParam) DebugString() string {
	return string(epp.Props.Name)
}
func (epp *EndpointParam) String() (s string) {
	var sb strings.Builder
	var cs string
	if len(epp.Constraints) != 0 {
		sb.WriteByte(':')
		for _, c := range epp.Constraints {
			sb.WriteString(c.String())
			sb.WriteByte(',')
		}
		cs = sb.String()
		cs = cs[:len(cs)-1]
	}
	name := string(epp.Props.Name)
	typ := string(epp.Type.Slug())
	if cs == "" && name == typ {
		s = fmt.Sprintf("{%s}", name)
		goto end
	}
	s = fmt.Sprintf("{%s:%s%s}", name, typ, cs)
end:
	return s
}

// EndpointParamArgs contains the configuration for creating a new EndpointParam.
type EndpointParamArgs struct {
	Props
	Type        pathvars.PVDataType   // Parameter data type
	Location    pathvars.LocationType // Parameter usage type
	Constraints []pathvars.Constraint // Validation constraints
	RawValue    string
}

// NewEndpointParam creates a new EndpointParam with the specified name and configuration.
// If no constraints are provided, an empty slice is initialized.
func NewEndpointParam(args EndpointParamArgs) EndpointParam {
	if args.Constraints == nil {
		args.Constraints = make([]pathvars.Constraint, 0)
	}
	return EndpointParam{
		Props:       args.Props,
		Type:        args.Type,
		Location:    args.Location,
		Constraints: args.Constraints,
	}
}
