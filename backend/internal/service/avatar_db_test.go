package service

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/storage"
)

func pngBytes() []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
}

func uploadDir(t *testing.T, svc *ProfileService) string {
	t.Helper()

	local, ok := svc.files.(*storage.Local)
	if !ok {
		t.Fatalf("expected a *storage.Local, got %T", svc.files)
	}
	return local.Dir()
}

func fileExists(t *testing.T, dir, name string) bool {
	t.Helper()

	_, err := os.Stat(filepath.Join(dir, name))
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	t.Fatalf("stat %s: %v", name, err)
	return false
}

func TestSetAvatarStoresTheFileAndPointsTheProfileAtIt(t *testing.T) {
	svc, userID := newProfileService(t)
	dir := uploadDir(t, svc)
	ctx := context.Background()

	name, err := svc.SetAvatar(ctx, userID, bytes.NewReader(pngBytes()), ".png")
	if err != nil {
		t.Fatalf("setting the avatar failed: %v", err)
	}

	if !fileExists(t, dir, name) {
		t.Errorf("%s was not written to disk", name)
	}

	detail, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("reading the profile back: %v", err)
	}
	if detail.Profile.AvatarFilename.String != name {
		t.Errorf("column = %q, want %q", detail.Profile.AvatarFilename.String, name)
	}
}

func TestSetAvatarDeletesTheFileItReplaces(t *testing.T) {
	svc, userID := newProfileService(t)
	dir := uploadDir(t, svc)
	ctx := context.Background()

	first, err := svc.SetAvatar(ctx, userID, bytes.NewReader(pngBytes()), ".png")
	if err != nil {
		t.Fatalf("first upload failed: %v", err)
	}

	second, err := svc.SetAvatar(ctx, userID, bytes.NewReader(pngBytes()), ".png")
	if err != nil {
		t.Fatalf("second upload failed: %v", err)
	}

	if first == second {
		t.Fatal("both uploads produced the same filename - storage is not generating unique names")
	}
	if fileExists(t, dir, first) {
		t.Errorf("the replaced file %s is still on disk", first)
	}
	if !fileExists(t, dir, second) {
		t.Errorf("the new file %s is missing", second)
	}

	detail, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("reading the profile back: %v", err)
	}
	if detail.Profile.AvatarFilename.String != second {
		t.Errorf("column = %q, want the newer %q", detail.Profile.AvatarFilename.String, second)
	}
}

func TestRemoveAvatarClearsTheColumnAndDeletesTheFile(t *testing.T) {
	svc, userID := newProfileService(t)
	dir := uploadDir(t, svc)
	ctx := context.Background()

	name, err := svc.SetAvatar(ctx, userID, bytes.NewReader(pngBytes()), ".png")
	if err != nil {
		t.Fatalf("setting the avatar failed: %v", err)
	}

	if err := svc.RemoveAvatar(ctx, userID); err != nil {
		t.Fatalf("removing the avatar failed: %v", err)
	}

	if fileExists(t, dir, name) {
		t.Errorf("%s is still on disk after removal", name)
	}

	detail, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("reading the profile back: %v", err)
	}
	if detail.Profile.AvatarFilename.Valid {
		t.Errorf("column = %q, want NULL", detail.Profile.AvatarFilename.String)
	}
}

func TestRemoveAvatarIsIdempotent(t *testing.T) {
	svc, userID := newProfileService(t)
	ctx := context.Background()

	if err := svc.RemoveAvatar(ctx, userID); err != nil {
		t.Fatalf("removing an avatar that was never set failed: %v", err)
	}
	if err := svc.RemoveAvatar(ctx, userID); err != nil {
		t.Fatalf("removing twice failed: %v", err)
	}
}
