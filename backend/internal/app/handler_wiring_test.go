package app

import (
	"reflect"
	"testing"
)

// The Deps struct traded a compile error for a runtime nil: adding a field to
// handler.Deps compiles everywhere, and a forgotten line in newHandler leaves
// production passing nil until the first request to that endpoint panics.
//
// This test puts the guarantee back. It fills every pointer on api, builds the
// handler the way NewRouter does, and fails if anything arrived nil.
func TestEveryHandlerDependencyIsWired(t *testing.T) {
	appService := &api{}

	// A non-nil value for every pointer field, so a nil on the other side can
	// only mean newHandler dropped it.
	populated := reflect.ValueOf(appService).Elem()
	for i := range populated.NumField() {
		field := populated.Field(i)
		if field.Kind() == reflect.Pointer && field.CanSet() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	}

	h := reflect.ValueOf(newHandler(appService)).Elem()

	for i := range h.NumField() {
		name := h.Type().Field(i).Name
		field := h.Field(i)

		switch field.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		default:
			// Scalars like maxUploadBytes and cookieSecure cannot be nil, so
			// there is nothing here to forget.
			continue
		}

		if field.IsNil() {
			t.Errorf("Handler.%s is nil - newHandler in router.go does not pass it", name)
		}
	}
}
