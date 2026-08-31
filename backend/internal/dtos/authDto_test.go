package dtos

import (
	"reflect"
	"strings"
	"testing"
)

func TestNoOtherResponseSaysHowSomeoneSignsIn(t *testing.T) {
	leaks := map[string]bool{"has_password": true, "providers": true}

	shapes := []struct {
		typ    reflect.Type
		reason string
	}{
		{reflect.TypeFor[PublicProfileResponse](),
			"knowing an address belongs to a Google account tells an attacker what to forge"},
		{reflect.TypeFor[AdminUserResponse](),
			"the admin list is about moderation, not about how people sign in"},
		{reflect.TypeFor[OwnProfileResponse](),
			"this belongs on UserInfo - a second copy is one more thing to keep in step"},
	}

	for _, shape := range shapes {
		for field := range shape.typ.Fields() {
			tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if leaks[tag] {
				t.Errorf("%s carries %q - %s", shape.typ.Name(), tag, shape.reason)
			}
		}
	}
}
