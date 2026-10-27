package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const MaxDocumentSize int64 = 10 * 1024 * 1024 // 10 MiB

var (
	ErrEmptyFile       = errors.New("file kosong")
	ErrFileTooLarge    = errors.New("ukuran file melebihi batas 10 MiB")
	ErrUnsupportedType = errors.New("format file tidak didukung; gunakan PDF atau CSV")
	ErrInvalidPDF      = errors.New("file bukan PDF yang valid")
	ErrInvalidCSV      = errors.New("struktur CSV tidak valid atau tidak dapat dibaca")
)

// ValidateDocument checks the file extension, size, and basic file structure.
// The returned MIME type is determined from the validated content/extension,
// not from the client-supplied Content-Type header.
func ValidateDocument(filename string, content []byte) (string, error) {
	if len(content) == 0 {
		return "", ErrEmptyFile
	}

	if int64(len(content)) > MaxDocumentSize {
		return "", ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".pdf":
		if !bytes.HasPrefix(content, []byte("%PDF-")) {
			return "", ErrInvalidPDF
		}

		return "application/pdf", nil

	case ".csv":
		reader := csv.NewReader(bytes.NewReader(content))
		reader.FieldsPerRecord = -1
		reader.LazyQuotes = false

		records := 0

		for {
			_, err := reader.Read()

			if err == io.EOF {
				break
			}

			if err != nil {
				return "", fmt.Errorf("%w: %v", ErrInvalidCSV, err)
			}

			records++
		}

		if records == 0 {
			return "", ErrInvalidCSV
		}

		return "text/csv", nil

	default:
		return "", ErrUnsupportedType
	}
}
