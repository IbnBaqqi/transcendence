package storage

import (
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
