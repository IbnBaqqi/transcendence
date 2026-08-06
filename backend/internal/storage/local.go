package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// ErrInvalidName guards Delete against anything that isn't a bare filename.
var ErrInvalidName = errors.New("invalid file name")

// Local stores files in a directory on the local filesystem.
type Local struct {
	dir string
}

// NewLocal creates the upload directory if it doesn't exist yet, so a fresh
// checkout or a fresh container works without a manual mkdir.
func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Local{dir: dir}, nil
}

// Dir exposes the directory so the router can serve it as static files.
func (l *Local) Dir() string {
	return l.dir
}

// Save streams r into a new file and returns the generated name.
func (l *Local) Save(r io.Reader, ext string) (string, error) {
	name := uuid.NewString() + ext
	path := filepath.Join(l.dir, name)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(path)
		return "", fmt.Errorf("write file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close file: %w", err)
	}

	return name, nil
}

// Delete removes a stored file.
func (l *Local) Delete(name string) error {
	if name == "" || name != filepath.Base(name) {
		return ErrInvalidName
	}

	err := os.Remove(filepath.Join(l.dir, name))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
