package services

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

const MaxCSVRecords = 5000

var (
	ErrInvalidCSVStructure = errors.New(
		"CSV harus memiliki header dan data yang dapat dipetakan ke field",
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

// ParseCSVFields parses key/value CSVs and header-based tables into extracted fields.
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

	return ParseFlexibleCSVFields(content)
}
