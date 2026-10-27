package services

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateDocument(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
		wantMIME string
		wantErr  error
	}{
		{
			name:     "valid text PDF signature",
			filename: "clinical-note.pdf",
			content:  []byte("%PDF-1.7\nsynthetic PDF content"),
			wantMIME: "application/pdf",
		},
		{
			name:     "valid CSV",
			filename: "clinical-note.csv",
			content:  []byte("field,value\nservice,MRI\n"),
			wantMIME: "text/csv",
		},
		{
			name:     "unsupported extension",
			filename: "clinical-note.txt",
			content:  []byte("some text"),
			wantErr:  ErrUnsupportedType,
		},
		{
			name:     "invalid PDF signature",
			filename: "clinical-note.pdf",
			content:  []byte("not a PDF"),
			wantErr:  ErrInvalidPDF,
		},
		{
			name:     "invalid CSV quoting",
			filename: "clinical-note.csv",
			content:  []byte("field,value\n\"unterminated,value\n"),
			wantErr:  ErrInvalidCSV,
		},
		{
			name:     "empty file",
			filename: "empty.csv",
			content:  []byte{},
			wantErr:  ErrEmptyFile,
		},
		{
			name:     "file exceeds maximum size",
			filename: "large.csv",
			content:  []byte(strings.Repeat("a", int(MaxDocumentSize)+1)),
			wantErr:  ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMIME, err := ValidateDocument(tt.filename, tt.content)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotMIME != tt.wantMIME {
				t.Errorf("expected MIME %q, got %q", tt.wantMIME, gotMIME)
			}
		})
	}
}
