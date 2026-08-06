package handler

import "testing"

func TestDetectImageExt(t *testing.T) {
	tests := []struct {
		name    string
		head    []byte
		wantExt string
		wantOK  bool
	}{
		{"png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...), ".png", true},
		{"jpeg", append([]byte("\xff\xd8\xff\xe0"), make([]byte, 16)...), ".jpg", true},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp", true},
		{"git is not allowed", []byte("GIF87a\x00\x00"), "", false},
		{"pdf is not allowed", []byte("%PDF-1.7\n"), "", false},
		{"text is not an imgae", []byte("hello, this is not an image"), "", false},
		{"empty upload", []byte{}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, ok := detectImageExt(tt.head)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ext != tt.wantExt {
				t.Errorf("ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestDetectImageExtIgnoresClaimedType(t *testing.T) {
	head := []byte("<script>alert(1)/<script>")
	if _, ok := detectImageExt(head); ok {
		t.Error("HTML accepted as an image")
	}
}
