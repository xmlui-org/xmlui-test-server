package cfgldr

import (
	"fmt"

	"github.com/mikeschinkel/go-pathvars"
)

var _ APIParamsMapper = (*APIParamsV1)(nil)

type APIParamsV1 []APIParamV1

func (ps APIParamsV1) APIParamsMap() (pm *APIParamsMap) {
	pm = &APIParamsMap{
		OrderedMap: *NewOrderedMap[APIParamsMapKey, APIParamsMapValue](),
	}
	for _, p := range ps {
		var value string
		switch {
		case p.Type == "" && p.Constraints == "":
			// Just use default type
			typ := pathvars.DefaultPVDataTypeName
			dt, err := pathvars.ParsePVDataType(p.NameSpec)
			if err == nil {
				typ = dt.Slug()
			}
			value = string(typ)
		case p.Type != "" && p.Constraints == "":
			// Just type
			value = p.Type
		case p.Type == "" && p.Constraints != "":
			// Default type with constraints
			typ := pathvars.DefaultPVDataTypeName
			dt, err := pathvars.ParsePVDataType(p.NameSpec)
			if err == nil {
				typ = dt.Slug()
			}
			value = fmt.Sprintf("%s:%s", typ, p.Constraints)
		case p.Type != "" && p.Constraints != "":
			// Type with constraints
			value = fmt.Sprintf("%s:%s", p.Type, p.Constraints)
		}
		if value != "" {
			pm.Set(APIParamsMapKey(p.NameSpec), APIParamsMapValue(value))
		}
	}
	return pm
}

func (ps APIParamsV1) Map() (m map[string]APIParamV1) {
	m = make(map[string]APIParamV1)
	for _, p := range ps {
		m[p.NameSpec] = p
	}
	return m
}

func (ps APIParamsV1) APIParamsV1() APIParamsV1 {
	return ps
}
