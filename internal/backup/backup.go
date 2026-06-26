package backup

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateDBZip creates a ZIP archive containing the SQLite database.
// Returns the ZIP bytes and the suggested filename.
func CreateDBZip(dbPath string) ([]byte, string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("zedproxy-db-%s.zip", timestamp)

	dbData, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("read database: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Add database file
	dbEntry, err := zw.Create(filepath.Base(dbPath))
	if err != nil {
		return nil, "", fmt.Errorf("zip create db entry: %w", err)
	}
	if _, err := dbEntry.Write(dbData); err != nil {
		return nil, "", fmt.Errorf("zip write db: %w", err)
	}

	// Add info file (no secrets)
	infoEntry, err := zw.Create("backup-info.txt")
	if err == nil {
		fmt.Fprintf(infoEntry, "ZedProxy Database Backup\nCreated: %s\nFile: %s\nSize: %d bytes\n",
			time.Now().Format("2006-01-02 15:04:05"),
			filepath.Base(dbPath),
			len(dbData),
		)
	}

	if err := zw.Close(); err != nil {
		return nil, "", fmt.Errorf("zip close: %w", err)
	}

	return buf.Bytes(), filename, nil
}

// CreateFullZip creates a ZIP archive containing the database and uploads directory.
func CreateFullZip(dbPath, uploadsDir string) ([]byte, string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("zedproxy-full-%s.zip", timestamp)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Add database
	if err := addFileToZip(zw, dbPath, "db/"+filepath.Base(dbPath)); err != nil {
		return nil, "", fmt.Errorf("add db to zip: %w", err)
	}

	// Add uploads directory
	if _, err := os.Stat(uploadsDir); err == nil {
		if err := addDirToZip(zw, uploadsDir, "uploads"); err != nil {
			return nil, "", fmt.Errorf("add uploads to zip: %w", err)
		}
	}

	// Add info file
	infoEntry, _ := zw.Create("backup-info.txt")
	fmt.Fprintf(infoEntry, "ZedProxy Full Backup\nCreated: %s\nIncludes: database, uploads\n",
		time.Now().Format("2006-01-02 15:04:05"))

	if err := zw.Close(); err != nil {
		return nil, "", fmt.Errorf("zip close: %w", err)
	}

	return buf.Bytes(), filename, nil
}

func addFileToZip(zw *zip.Writer, srcPath, entryName string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zw.Create(entryName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func addDirToZip(zw *zip.Writer, dirPath, prefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dirPath, path)
		entryName := prefix + "/" + filepath.ToSlash(rel)
		return addFileToZip(zw, path, entryName)
	})
}

// SaveZipToDir writes ZIP bytes to the backup directory and returns the full file path.
func SaveZipToDir(data []byte, backupDir, filename string) (string, error) {
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	dst := filepath.Join(backupDir, filename)
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return "", fmt.Errorf("write backup file: %w", err)
	}
	return dst, nil
}
