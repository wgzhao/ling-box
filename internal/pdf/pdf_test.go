package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

// minPDF generates a minimal single-page blank PDF.
// Returns the raw PDF bytes, suitable for OpenFromBytes.
func minPDF() []byte {
	// Objects
	objs := []string{
		"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj",
		"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj",
		"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>endobj",
	}

	var buf bytes.Buffer

	// Header
	buf.WriteString("%PDF-1.4\n")

	// Record byte offsets of each object relative to start of file
	offsets := make([]int64, len(objs)+1) // +1 for the zero entry
	for i, obj := range objs {
		offsets[i+1] = int64(buf.Len())
		buf.WriteString(obj + "\n")
	}

	// Cross-reference table
	xrefStart := int64(buf.Len())
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", len(offsets))
	fmt.Fprintf(&buf, "%010d 65535 f \n", 0)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}

	// Trailer
	buf.WriteString("trailer<</Size 4/Root 1 0 R>>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n", xrefStart)
	buf.WriteString("%%EOF\n")

	return buf.Bytes()
}

func TestOpenFromBytes(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}
	defer doc.Close()

	if doc.NumPages() != 1 {
		t.Errorf("NumPages = %d, want 1", doc.NumPages())
	}
}

func TestOpenFromBytesEmpty(t *testing.T) {
	_, err := OpenFromBytes(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
	_, err = OpenFromBytes([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestNumPages(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}
	defer doc.Close()

	n := doc.NumPages()
	if n <= 0 {
		t.Errorf("NumPages should be positive, got %d", n)
	}
}

func TestRenderPage(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}
	defer doc.Close()

	// Render page 0 at 72 DPI — small but valid
	png, err := doc.RenderPage(0, 72)
	if err != nil {
		t.Fatalf("RenderPage: %v", err)
	}

	if len(png) == 0 {
		t.Error("rendered PNG is empty")
	}

	// Verify PNG magic bytes
	if len(png) < 8 || png[0] != 0x89 || png[1] != 'P' || png[2] != 'N' || png[3] != 'G' {
		t.Error("output is not a valid PNG")
	}
}

func TestRenderPageOutOfRange(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}
	defer doc.Close()

	_, err = doc.RenderPage(99, 72)
	if err == nil {
		t.Error("expected error for out-of-range page")
	}
}

func TestRenderPageZeroDPI(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}
	defer doc.Close()

	// Zero DPI should fall back to default (150)
	png, err := doc.RenderPage(0, 0)
	if err != nil {
		t.Fatalf("RenderPage with zero DPI: %v", err)
	}
	if len(png) == 0 {
		t.Error("rendered PNG is empty")
	}
}

func TestClose(t *testing.T) {
	doc, err := OpenFromBytes(minPDF())
	if err != nil {
		t.Fatalf("OpenFromBytes: %v", err)
	}

	if err := doc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Second close should be safe (no-op or error)
	_ = doc.Close()
}
