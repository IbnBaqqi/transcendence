package dtos

import (
	"reflect"
	"strings"
	"testing"
)

func TestThePublicProfileDoesNotSayHowSomeoneSignsIn(t *testing.T) {
	leaks := map[string]bool{"has_password": true, "providers": true}

	for field := range reflect.TypeFor[PublicProfileResponse]().Fields() {
		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if leaks[tag] {
			t.Errorf("public profile carries %q - knowing an address is a Google account tells an attacker what to forge", tag)
		}
	}
}
