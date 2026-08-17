// Package pdf renders PDF pages to images using MuPDF via go-fitz.
package pdf

import (
	"github.com/gen2brain/go-fitz"
)

// Document wraps a MuPDF document.
type Document struct {
	doc *fitz.Document
}

// Open opens a PDF file by path.
func Open(filename string) (*Document, error) {
	doc, err := fitz.New(filename)
	if err != nil {
		return nil, err
	}
	return &Document{doc: doc}, nil
}

// OpenFromBytes opens a PDF from in-memory data.
func OpenFromBytes(data []byte) (*Document, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, err
	}
	return &Document{doc: doc}, nil
}

// NumPages returns the total page count.
func (d *Document) NumPages() int {
	return d.doc.NumPage()
}

// RenderPage renders page n (0-indexed) to PNG bytes at the given DPI.
// A DPI of 0 uses the document's native resolution.
func (d *Document) RenderPage(n int, dpi float64) ([]byte, error) {
	if dpi <= 0 {
		dpi = 150
	}
	return d.doc.ImagePNG(n, dpi)
}

// Close releases resources held by the document. It is safe to call
// multiple times: go-fitz would double-free the underlying MuPDF
// document on a second call.
func (d *Document) Close() error {
	if d.doc == nil {
		return nil
	}
	err := d.doc.Close()
	d.doc = nil
	return err
}
