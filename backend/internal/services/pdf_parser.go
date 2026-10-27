package services

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

const (
	maxPDFPages          = 100
	maxPDFTextCharacters = 100_000
)

var (
	ErrPDFHasNoText    = errors.New("PDF tidak memiliki teks yang dapat dibaca")
	ErrPDFInvalidText  = errors.New("struktur PDF tidak valid atau teks tidak dapat dibaca")
	ErrPDFTextTooLarge = errors.New("teks PDF melebihi batas pemrosesan")
	ErrPDFPageLimit    = errors.New("jumlah halaman PDF melebihi batas pemrosesan")
)

// ParseTextPDF extracts selectable text from a PDF. Scanned PDFs are not OCRed.
func ParseTextPDF(content []byte) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text = ""
			err = fmt.Errorf("%w: %v", ErrPDFInvalidText, recovered)
		}
	}()

	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPDFInvalidText, err)
	}

	pageCount := reader.NumPage()
	if pageCount > maxPDFPages {
		return "", ErrPDFPageLimit
	}

	var extracted strings.Builder
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		rows, err := reader.Page(pageNumber).GetTextByRow()
		if err != nil {
			return "", fmt.Errorf(
				"%w: halaman %d: %v",
				ErrPDFInvalidText,
				pageNumber,
				err,
			)
		}

		for _, row := range rows {
			for _, item := range row.Content {
				extracted.WriteString(item.S)
				extracted.WriteByte(' ')
			}
			extracted.WriteByte('\n')

			if extracted.Len() > maxPDFTextCharacters {
				return "", ErrPDFTextTooLarge
			}
		}
	}

	text = strings.TrimSpace(extracted.String())
	if text == "" {
		return "", ErrPDFHasNoText
	}

	return text, nil
}
