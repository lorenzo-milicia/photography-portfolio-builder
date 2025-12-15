package processing

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"testing"
)

// MockImageSource
type MockImageSource struct {
	data []byte
	name string
}

func (m *MockImageSource) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.data)), nil
}

func (m *MockImageSource) Name() string {
	return m.name
}

// MockImageDestination
type MockImageDestination struct {
	Files map[string][]byte
}

func NewMockImageDestination() *MockImageDestination {
	return &MockImageDestination{Files: make(map[string][]byte)}
}

func (m *MockImageDestination) Create(filename string) (io.WriteCloser, error) {
	return &MockWriter{filename: filename, dest: m}, nil
}

// Exists reports whether a file with the given filename was created
func (m *MockImageDestination) Exists(filename string) bool {
	_, ok := m.Files[filename]
	return ok
}

type MockWriter struct {
	filename string
	dest     *MockImageDestination
	buf      bytes.Buffer
}

func (m *MockWriter) Write(p []byte) (int, error) {
	return m.buf.Write(p)
}

func (m *MockWriter) Close() error {
	m.dest.Files[m.filename] = m.buf.Bytes()
	return nil
}

func createTestImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestProcessor_ProcessImage(t *testing.T) {
	// Setup
	imageData := createTestImage()
	src := &MockImageSource{data: imageData, name: "test.jpg"}
	dst := NewMockImageDestination()
	processor := NewProcessor(ProcessConfig{Widths: []int{50}, Quality: 75})

	// Execute
	err := processor.ProcessImage(src, dst)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Verify
	if len(dst.Files) == 0 {
		t.Error("No files generated")
	}

	for filename, content := range dst.Files {
		t.Logf("Generated file: %s (%d bytes)", filename, len(content))
		if len(content) == 0 {
			t.Errorf("File %s is empty", filename)
		}
		// Basic check if it looks like WebP (RIFF header)
		if !bytes.HasPrefix(content, []byte("RIFF")) {
			t.Errorf("File %s does not look like a WebP file", filename)
		}
	}
}
