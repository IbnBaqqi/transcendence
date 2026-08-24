package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const sniffLen = 512

type imageUpload struct {
	Body io.Reader
	Ext  string
	file multipart.File
}

func (u imageUpload) Close() error {
	return u.file.Close()
}

func (h *Handler) readImageUpload(w http.ResponseWriter, r *http.Request, field string) (imageUpload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)

	file, _, err := r.FormFile(field)
	if err != nil {
		if isTooLarge(err) {
			respondWithError(w, http.StatusRequestEntityTooLarge, "image is too large")
			return imageUpload{}, false
		}
		respondWithError(w, http.StatusBadRequest,
			fmt.Sprintf("expected a multipart form with a %q file field", field))
		return imageUpload{}, false
	}

	head := make([]byte, sniffLen)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		_ = file.Close()
		if isTooLarge(err) {
			respondWithError(w, http.StatusRequestEntityTooLarge, "image is too large")
			return imageUpload{}, false
		}
		respondWithError(w, http.StatusBadRequest, "could not read uploaded file")
		return imageUpload{}, false
	}
	head = head[:n]

	ext, ok := detectImageExt(head)
	if !ok {
		_ = file.Close()
		respondWithError(w, http.StatusUnsupportedMediaType, "only JPEG, PNG and WebP images are allowed")
		return imageUpload{}, false
	}

	return imageUpload{
		Body: io.MultiReader(bytes.NewReader(head), file),
		Ext:  ext,
		file: file,
	}, true
}

func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

func detectImageExt(head []byte) (string, bool) {
	ext, ok := allowedImageTypes[http.DetectContentType(head)]
	return ext, ok
}
