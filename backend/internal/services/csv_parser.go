package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const MaxCSVRecords = 5000

var (
	ErrInvalidCSVStructure = errors.New(
		"CSV harus memiliki header field,value",
	)
	ErrCSVRecordLimit = errors.New(
		"jumlah baris CSV melebihi batas",
	)
	ErrInvalidCSVField = errors.New(
		"field CSV tidak valid",
	)
)

type ExtractedField struct {
	Key      string
	Value    string
	Evidence string
}

// ParseCSVFields converts a CSV containing field,value columns
// into normalized extracted fields.
func ParseCSVFields(content []byte) ([]ExtractedField, error) {
	if len(content) == 0 {
		return nil, ErrInvalidCSVStructure
	}

	if !utf8.Valid(content) {
		return nil, fmt.Errorf(
			"%w: konten harus berupa UTF-8",
			ErrInvalidCSVStructure,
		)
	}

	// Remove UTF-8 BOM if present.
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = false
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf(
			"%w: header tidak dapat dibaca",
			ErrInvalidCSVStructure,
		)
	}

	if len(header) != 2 ||
		!strings.EqualFold(strings.TrimSpace(header[0]), "field") ||
		!strings.EqualFold(strings.TrimSpace(header[1]), "value") {
		return nil, ErrInvalidCSVStructure
	}

	fields := make([]ExtractedField, 0)
	lineNumber := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"%w: baris %d: %v",
				ErrInvalidCSVStructure,
				lineNumber+1,
				err,
			)
		}

		lineNumber++

		if len(record) != 2 {
			return nil, fmt.Errorf(
				"%w: baris %d harus memiliki dua kolom",
				ErrInvalidCSVStructure,
				lineNumber,
			)
		}

		key := strings.TrimSpace(record[0])
		value := strings.TrimSpace(record[1])

		// Ignore completely blank rows.
		if key == "" && value == "" {
			continue
		}

		if key == "" || len(key) > 100 {
			return nil, fmt.Errorf(
				"%w: nama field pada baris %d kosong atau terlalu panjang",
				ErrInvalidCSVField,
				lineNumber,
			)
		}

		fields = append(fields, ExtractedField{
			Key:      key,
			Value:    value,
			Evidence: fmt.Sprintf("%s: %s", key, value),
		})

		if len(fields) > MaxCSVRecords {
			return nil, ErrCSVRecordLimit
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf(
			"%w: tidak ada data field",
			ErrInvalidCSVStructure,
		)
	}

	return fields, nil
}
