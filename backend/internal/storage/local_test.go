package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveWritesContent(t *testing.T) {
	store := newTestStore(t)

	name, err := store.Save(strings.NewReader("mushroom bytes"), ".jpg")
	if err != nil {
		t.Fatalf("Save failed %v", err)
	}
	if !strings.HasSuffix(name, ".jpg") {
		t.Errorf("name%q does not end in .jpg", name)
	}

	got, err := os.ReadFile(filepath.Join(store.Dir(), name))
	if err != nil {
		t.Fatalf("stored file unreadable: %v", err)
	}
	if string(got) != "mushroom bytes" {
		t.Errorf("content = %q, want %q", got, "mushroom bytes")
	}
}

func TestSavegeneratesUniqueNames(t *testing.T) {
	store := newTestStore(t)

	first, err := store.Save(strings.NewReader("a"), ".png")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	second, err := store.Save(strings.NewReader("b"), ".png")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if first == second {
		t.Fatalf("two uploads got the same name")
	}
}

func TestDeleteRemovesFileAndIsIdempotent(t *testing.T) {
	store := newTestStore(t)

	name, err := store.Save(strings.NewReader("bytes"), ".png")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.Delete(name); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir(), name)); !os.IsNotExist(err) {
		t.Error("file still on disk after Delete")
	}
	if err := store.Delete(name); err != nil {
		t.Errorf("second Delete should be a no-op, got %v", err)
	}
}

func TestDeleteRejectsPaths(t *testing.T) {
	store := newTestStore(t)

	outside := filepath.Join(filepath.Dir(store.Dir()), "victim.txt")
	if err := os.WriteFile(outside, []byte("do not delete me"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	for _, name := range []string{"", "../victim.txt", "sub/file.png"} {
		if err := store.Delete(name); err == nil {
			t.Errorf("Delete(%q) was allowed", name)
		}
	}

	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file outside the upload dir was touched: %v", err)
	}
}

func newTestStore(t *testing.T) *Local {
	t.Helper()

	store, err := NewLocal(filepath.Join(t.TempDir(), "uploads"))
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}
	return store
}

func TestQuarantineHidesAFileAndReleaseBringsItBack(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	name, err := s.Save(strings.NewReader("a photo"), ".jpg")
	if err != nil {
		t.Fatalf("saving: %v", err)
	}

	served := func() bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	if !served() {
		t.Fatal("the file was not saved where it is served from")
	}

	if err := s.Quarantine(name); err != nil {
		t.Fatalf("quarantining: %v", err)
	}
	if served() {
		t.Error("the file is still in the served directory")
	}

	// Moved, not destroyed - a restore has to be able to undo it.
	if _, err := os.Stat(filepath.Join(dir, QuarantineDir, name)); err != nil {
		t.Errorf("the file was not kept in quarantine: %v", err)
	}

	if err := s.Release(name); err != nil {
		t.Fatalf("releasing: %v", err)
	}
	if !served() {
		t.Error("the file did not come back")
	}

	body, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil || string(body) != "a photo" {
		t.Errorf("contents changed across the round trip: %q, %v", body, err)
	}
}

func TestQuarantineIsForgivingAndStillGuardsNames(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// A listing whose file is already gone must not fail a moderation.
	if err := s.Quarantine("does-not-exist.jpg"); err != nil {
		t.Errorf("quarantining a missing file: %v", err)
	}
	if err := s.Release("does-not-exist.jpg"); err != nil {
		t.Errorf("releasing a missing file: %v", err)
	}

	for _, bad := range []string{"", "../escape.jpg", "sub/dir.jpg"} {
		if err := s.Quarantine(bad); !errors.Is(err, ErrInvalidName) {
			t.Errorf("Quarantine(%q) = %v, want ErrInvalidName", bad, err)
		}
		if err := s.Release(bad); !errors.Is(err, ErrInvalidName) {
			t.Errorf("Release(%q) = %v, want ErrInvalidName", bad, err)
		}
	}
}
