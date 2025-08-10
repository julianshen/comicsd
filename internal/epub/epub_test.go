package epub

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// Test that EPUBWriter records filenames and MIME types correctly in the manifest
func TestEPUBWriterManifestRecordsMimeTypes(t *testing.T) {
	var buf bytes.Buffer
	writer := NewEPUBWriter(&buf, "Test Title")

	if err := writer.AddPage("img1.png", []byte("data1")); err != nil {
		t.Fatalf("AddPage img1 failed: %v", err)
	}
	if err := writer.AddPage("img2.jpg", []byte("data2")); err != nil {
		t.Fatalf("AddPage img2 failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read zip: %v", err)
	}

	var contentOpf string
	for _, f := range zr.File {
		if f.Name == "OEBPS/content.opf" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open content.opf: %v", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read content.opf: %v", err)
			}
			contentOpf = string(data)
			break
		}
	}

	if contentOpf == "" {
		t.Fatalf("content.opf not found in EPUB")
	}

	if !strings.Contains(contentOpf, "href=\"images/img1.png\" media-type=\"image/png\"") {
		t.Errorf("manifest missing img1.png with image/png: %s", contentOpf)
	}
	if !strings.Contains(contentOpf, "href=\"images/img2.jpg\" media-type=\"image/jpeg\"") {
		t.Errorf("manifest missing img2.jpg with image/jpeg: %s", contentOpf)
	}
}
