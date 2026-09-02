package service

import (
	"reflect"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// fillNonZero sets every field it can reach to something distinguishable from
// the zero value, so a field the mapper forgets shows up as zero afterwards.
// It fails on a type it does not know rather than leaving a hole, since a
// silently unfilled field would make the guard below pass for the wrong reason.
func fillNonZero(t *testing.T, v reflect.Value) {
	t.Helper()

	switch v.Kind() {
	case reflect.String:
		v.SetString("filled")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int32, reflect.Int64:
		v.SetInt(7)
	case reflect.Array:
		for i := range v.Len() {
			v.Index(i).SetUint(1)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			v.Set(reflect.ValueOf(time.Now()))
			return
		}
		for i := range v.NumField() {
			fillNonZero(t, v.Field(i))
		}
	default:
		t.Fatalf("no filler for %v - add one, or this guard silently stops guarding", v.Type())
	}
}

// The view-column test next to this one compares the two structs and so catches
// a column missing from migration 016. It cannot catch a field present in both
// and left out of orderFromAdminRow's struct literal: Go zero-fills the omission
// without complaint, and the admin list publishes a zero value. That is the same
// drift, one layer up, so it gets its own guard.
func TestTheMapperCarriesEveryColumnItIsGiven(t *testing.T) {
	var row database.AdminOrder
	fillNonZero(t, reflect.ValueOf(&row).Elem())

	got := reflect.ValueOf(orderFromAdminRow(row))

	for i := range got.NumField() {
		if got.Field(i).IsZero() {
			t.Errorf("orderFromAdminRow drops %s: it is set on the view row and zero on the order",
				got.Type().Field(i).Name)
		}
	}
}
