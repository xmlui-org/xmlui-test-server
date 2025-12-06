package localsvr

import (
	"reflect"
)

func IsNil(value any) (isNil bool) {
	var v reflect.Value

	isNil = true
	if value == nil {
		goto end
	}
	v = reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		isNil = v.IsNil()
	default:
		isNil = false
	}
end:
	return isNil
}
