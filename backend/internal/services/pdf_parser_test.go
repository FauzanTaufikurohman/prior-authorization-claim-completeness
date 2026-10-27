package services

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseTextPDF(t *testing.T) {
	text, err := ParseTextPDF(testTextPDF("Member ID: 12345"))
	if err != nil {
		t.Fatalf("ParseTextPDF() error = %v", err)
	}
	if !strings.Contains(text, "Member ID: 12345") {
		t.Fatalf("ParseTextPDF() = %q, want extracted text", text)
	}
}

func TestParseTextPDFRejectsScannedPDF(t *testing.T) {
	_, err := ParseTextPDF(testTextPDF(""))
	if err != ErrPDFHasNoText {
		t.Fatalf("ParseTextPDF() error = %v, want %v", err, ErrPDFHasNoText)
	}
}

func TestParseTextPDFRejectsMalformedPDF(t *testing.T) {
	_, err := ParseTextPDF([]byte("%PDF-not-a-valid-document"))
	if !errors.Is(err, ErrPDFInvalidText) {
		t.Fatalf("ParseTextPDF() error = %v, want %v", err, ErrPDFInvalidText)
	}
}

func testTextPDF(content string) []byte {
	stream := ""
	if content != "" {
		stream = fmt.Sprintf("BT /F1 12 Tf 10 100 Td (%s) Tj ET", content)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 144] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}

	var document bytes.Buffer
	document.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = document.Len()
		fmt.Fprintf(&document, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}

	xrefOffset := document.Len()
	fmt.Fprintf(&document, "xref\n0 %d\n", len(offsets))
	document.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&document, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(
		&document,
		"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF",
		len(offsets),
		xrefOffset,
	)
	return document.Bytes()
}
