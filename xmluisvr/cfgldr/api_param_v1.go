package cfgldr

import (
	"strings"

	"github.com/mikeschinkel/go-pathvars"
	_ "github.com/mikeschinkel/go-pathvars/dtclassifiers"
)

var _ APIParam = (*APIParamV1)(nil)

type APIParamV1 struct {
	NameSpec     string `json:"name"`
	Type         string `json:"type"`
	Constraints  string `json:"constraints"`
	MultiSegment bool   `json:"multi_segment"`
	Optional     bool   `json:"optional"`
	DefaultValue string `json:"default"`
	Location     string `json:"-"`
}

type APIParamV1Args struct {
	NameSpec     string
	Type         string
	Constraints  string
	MultiSegment bool
	Optional     bool
	DefaultValue string
}

func (APIParamV1) APIParam() {}

func NewAPIParamV1(args APIParamV1Args) APIParamV1 {
	return APIParamV1{
		NameSpec:     args.NameSpec,
		Type:         args.Type,
		Constraints:  args.Constraints,
		MultiSegment: args.MultiSegment,
		Optional:     args.Optional,
		DefaultValue: args.DefaultValue,
	}
}

func NewAPIParamV1WithConstraints(nameSpec, typ, constraints string) APIParamV1 {
	return APIParamV1{
		NameSpec:    nameSpec,
		Type:        typ,
		Constraints: constraints,
	}
}

func (p APIParamV1) String() string {
	sb := strings.Builder{}
	sb.WriteByte('{')
	sb.WriteString(p.NameSpec)

	switch {
	case p.Type != "":
		sb.WriteByte(':')
		sb.WriteString(strings.ToLower(p.Type))
	case len(p.Constraints) != 0:
		sb.WriteByte(':')
	}

	if len(p.Constraints) != 0 {
		sb.WriteByte(':')
		sb.WriteString(p.Constraints)
	}
	sb.WriteByte('}')
	return sb.String()
}

// ParseAPIParamV1 parses the name and type/constraint spec and validates the
// type, returning an instance of APIParamV1 is a valid type, or an error
// otherwise.
func ParseAPIParamV1(name, spec string) (param APIParamV1, err error) {
	typ, cs, _ := strings.Cut(spec, ":")
	_, err = pathvars.ParsePVDataType(typ)
	if err != nil {
		// Type is explicitly specified, and invalid
		goto end
	}
	param = NewAPIParamV1WithConstraints(name, typ, cs)
end:
	return param, err
}
