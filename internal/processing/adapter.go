package processing

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileSource implements ImageSource for local filesystem
type FileSource struct {
	Path string
}

func (f *FileSource) Open() (io.ReadCloser, error) {
	return os.Open(f.Path)
}

func (f *FileSource) Name() string {
	return filepath.Base(f.Path)
}

// MultipartFileSource implements ImageSource for uploaded multipart files
type MultipartFileSource struct {
	Header *multipart.FileHeader
}

func (m *MultipartFileSource) Open() (io.ReadCloser, error) {
	return m.Header.Open()
}

func (m *MultipartFileSource) Name() string {
	return m.Header.Filename
}

// FileDestination implements ImageDestination for local filesystem
type FileDestination struct {
	Dir string
}

func (f *FileDestination) Create(filename string) (io.WriteCloser, error) {
	fullPath := filepath.Join(f.Dir, filename)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return os.Create(fullPath)
}

// Exists checks whether a file with the given filename exists in the destination dir
func (f *FileDestination) Exists(filename string) bool {
	path := filepath.Join(f.Dir, filename)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// MultiFileDestination writes normal variants into the provided Dir (project directory)
// and writes thumbnails into a .thumbs directory under SourceDir (the source photos directory).
type MultiFileDestination struct {
	Dir       string
	Root      string
	SourceDir string
}

func (m *MultiFileDestination) Create(filename string) (io.WriteCloser, error) {
	var fullPath string
	if len(filename) >= 6 && filename[:6] == "thumb-" {
		// thumbs go into SourceDir/.thumbs (alongside source photos)
		fullPath = filepath.Join(m.SourceDir, ".thumbs", filename)
	} else {
		fullPath = filepath.Join(m.Dir, filename)
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return os.Create(fullPath)
}

func (m *MultiFileDestination) Exists(filename string) bool {
	var path string
	if len(filename) >= 6 && filename[:6] == "thumb-" {
		path = filepath.Join(m.SourceDir, ".thumbs", filename)
	} else {
		path = filepath.Join(m.Dir, filename)
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
