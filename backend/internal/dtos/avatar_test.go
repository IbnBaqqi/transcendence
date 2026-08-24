package dtos

import (
	"database/sql"
	"testing"
)

func TestAvatarURLIsNilWhenUnset(t *testing.T) {
	if got := avatarURL(sql.NullString{}); got != nil {
		t.Errorf("got %q, want nil - an unset avatar must not become a URL", *got)
	}
}

func TestAvatarURLIsServedFromTheUploadPrefix(t *testing.T) {
	got := avatarURL(sql.NullString{String: "abc.png", Valid: true})
	if got == nil {
		t.Fatal("got nil for a set avatar")
	}
	if want := UploadURLPrefix + "abc.png"; *got != want {
		t.Errorf("got %q, want %q", *got, want)
	}
}
