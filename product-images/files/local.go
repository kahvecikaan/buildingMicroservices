package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Local struct {
	maxFileSize int // Maximum number of bytes for files
	basePath    string
}

// maxBytesWriter is a writer that errors when more than N bytes are written
type maxBytesWriter struct {
	w io.Writer // underlying writer
	n int       // max bytes remaining
}

func (l *maxBytesWriter) Write(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	if len(p) > l.n {
		p = p[:l.n]
	}
	n, err := l.w.Write(p)
	l.n -= n
	if err != nil {
		return n, err
	}
	if l.n <= 0 {
		return n, io.EOF
	}
	return n, nil
}

// NewLocal creates a new Local filesystem with the given base path
// basePath is the base directory to save the files to
// maxSize is the max number of bytes that a file can be
func NewLocal(basePath string, maxSize int) (*Local, error) {
	p, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	return &Local{basePath: p, maxFileSize: maxSize}, nil
}

func (l *Local) Save(path string, contents io.Reader) error {
	// Get the full path for the file
	fp := l.fullPath(path)
	// Get the directory of the file
	dir := filepath.Dir(fp)

	// Create all directories in the path if they don't exist
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}

	// Create a temporary file in the same directory
	tempFile, err := os.CreateTemp(dir, "temp-*")
	if err != nil {
		return fmt.Errorf("unable to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	// Ensure the temporary file is deleted if the function returns early
	defer os.Remove(tempPath)

	// Create a maxBytesWriter to limit the file size
	writer := &maxBytesWriter{w: tempFile, n: l.maxFileSize}
	// Copy the contents to the temporary file, limited by maxBytesWriter
	written, err := io.Copy(writer, contents)
	if err != nil && err != io.EOF {
		tempFile.Close()
		return fmt.Errorf("unable to write to file: %w", err)
	}

	// Close the temporary file
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("unable to close temporary file: %w", err)
	}

	// Check if the file size exceeds the limit
	if written > int64(l.maxFileSize) {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", l.maxFileSize)
	}

	// Move the temporary file to the final location
	if err := os.Rename(tempPath, fp); err != nil {
		return fmt.Errorf("unable to move temporary file to final location: %w", err)
	}

	return nil
}

func (l *Local) Get(path string) (*os.File, error) {
	// get the full path for the file
	fp := l.fullPath(path)

	// open the file
	f, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf("unable to open the file: %w", err)
	}

	return f, nil
}

// returns the absolute full path
func (l *Local) fullPath(path string) string {
	// append the given path to the base path
	return filepath.Join(l.basePath, path)
}
