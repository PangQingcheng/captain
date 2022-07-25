

package reflectutils

import (
	"reflect"
)

func In(value interface{}, container interface{}) bool {
	containerValue := reflect.ValueOf(container)
	switch reflect.TypeOf(container).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < containerValue.Len(); i++ {
			if containerValue.Index(i).Interface() == value {
				return true
			}
		}
	case reflect.Map:
		if containerValue.MapIndex(reflect.ValueOf(value)).IsValid() {
			return true
		}
	default:
		return false
	}
	return false
}

func Override(left interface{}, right interface{}) {
	if reflect.ValueOf(left).IsNil() || reflect.ValueOf(right).IsNil() {
		return
	}

	if reflect.ValueOf(left).Type().Kind() != reflect.Ptr ||
		reflect.ValueOf(right).Type().Kind() != reflect.Ptr ||
		reflect.ValueOf(left).Kind() != reflect.ValueOf(right).Kind() {
		return
	}

	oldVal := reflect.ValueOf(left).Elem()
	newVal := reflect.ValueOf(right).Elem()

	for i := 0; i < oldVal.NumField(); i++ {
		val := newVal.Field(i).Interface()
		if !reflect.DeepEqual(val, reflect.Zero(reflect.TypeOf(val)).Interface()) {
			oldVal.Field(i).Set(reflect.ValueOf(val))
		}
	}
}
