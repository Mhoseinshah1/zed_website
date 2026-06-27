package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// BaseUploadDir is the base directory for all uploads.
var BaseUploadDir = "/opt/zedproxy/static/uploads"

// AllowedExts lists allowed file extensions.
var AllowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
	".mp4":  true,
	".webm": true,
	".pdf":  true,
}

// MaxFileSize is 20MB.
const MaxFileSize = 20 * 1024 * 1024

// Save saves an uploaded file to a subdirectory under BaseUploadDir.
// It returns the public URL path (e.g. /uploads/images/abc123.jpg).
func Save(fh *multipart.FileHeader, subdir string) (string, error) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !AllowedExts[ext] {
		return "", fmt.Errorf("file type not allowed: %s", ext)
	}
	if fh.Size > MaxFileSize {
		return "", fmt.Errorf("file too large: %d bytes", fh.Size)
	}

	// sanitize subdir to prevent path traversal
	subdir = filepath.Clean(subdir)
	if strings.Contains(subdir, "..") {
		return "", fmt.Errorf("invalid subdir")
	}

	dir := filepath.Join(BaseUploadDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	name := hex.EncodeToString(b) + ext
	dst := filepath.Join(dir, name)

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		return "", err
	}
	return "/uploads/" + subdir + "/" + name, nil
}
