package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseKeyValueText extracts labeled values and simple pipe/tabular rows.
// It intentionally keeps the source label in Evidence so a reviewer can
// verify the generated key and value against the document.
func ParseKeyValueText(document string) []ExtractedField {
	fields := make([]ExtractedField, 0)
	lines := strings.Split(document, "\n")
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		rawLine := lines[lineIndex]
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		separator := ""
		if strings.Contains(line, "|") {
			separator = "|"
		} else if strings.Contains(rawLine, "\t") {
			separator = "\t"
		}
		if separator != "" {
			cells := splitTableLine(line, separator)
			if len(cells) >= 3 && lineIndex+1 < len(lines) {
				nextCells := splitTableLine(lines[lineIndex+1], separator)
				if len(nextCells) == len(cells) {
					for lineIndex+1 < len(lines) {
						values := splitTableLine(lines[lineIndex+1], separator)
						if len(values) != len(cells) {
							break
						}
						lineIndex++
						for column, value := range values {
							appendKeyValue(&fields, cells[column], value, fmt.Sprintf("%s: %s", cells[column], value))
						}
					}
					continue
				}
			}
			if len(cells) >= 2 {
				for i := 0; i+1 < len(cells); i += 2 {
					if isColumnHeader(cells[i]) || isColumnHeader(cells[i+1]) {
						continue
					}
					appendKeyValue(&fields, cells[i], cells[i+1], line)
				}
				continue
			}
		}
		// Split only on the first separator; values may themselves contain
		// colons or equals signs (for example URLs and expressions).
		idx := strings.IndexAny(line, ":=")
		if idx > 0 {
			appendKeyValue(&fields, line[:idx], line[idx+1:], line)
		}
	}
	return fields
}

func isColumnHeader(value string) bool {
	switch normalizeFieldKey(value) {
	case "field", "fields", "key", "variable", "label", "name", "value", "values", "response", "answer", "result", "data":
		return true
	default:
		return false
	}
}

// ParseFlexibleCSVFields accepts conventional key/value CSVs with any header
// names, or a table whose first row names the variables and later rows contain
// their values. Each cell retains row-level evidence.
func ParseFlexibleCSVFields(content []byte) ([]ExtractedField, error) {
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("%w: konten harus berupa UTF-8", ErrInvalidCSVStructure)
	}
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = false
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: header tidak dapat dibaca", ErrInvalidCSVStructure)
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}
	if len(header) < 2 {
		return nil, ErrInvalidCSVStructure
	}

	var records [][]string
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCSVStructure, readErr)
		}
		if len(row) != len(header) {
			return nil, fmt.Errorf("%w: jumlah kolom baris tidak cocok dengan header", ErrInvalidCSVStructure)
		}
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		records = append(records, row)
		if len(records) > MaxCSVRecords {
			return nil, ErrCSVRecordLimit
		}
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: tidak ada data", ErrInvalidCSVStructure)
	}

	// A two-column file is treated as key/value pairs regardless of header
	// wording. This covers common exports such as "Question,Response".
	fields := make([]ExtractedField, 0)
	if len(header) == 2 {
		for i, row := range records {
			if row[0] == "" && row[1] == "" {
				continue
			}
			if row[0] == "" {
				return nil, fmt.Errorf("%w: label baris %d kosong", ErrInvalidCSVField, i+2)
			}
			appendKeyValue(&fields, row[0], row[1], fmt.Sprintf("%s: %s", row[0], row[1]))
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("%w: tidak ada data field", ErrInvalidCSVStructure)
		}
		return fields, nil
	}

	// Multi-column tables use their header labels as variable names.
	for _, row := range records {
		for column, value := range row {
			if value == "" || header[column] == "" {
				continue
			}
			evidence := fmt.Sprintf("%s: %s", header[column], value)
			appendKeyValue(&fields, header[column], value, evidence)
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: tidak ada data field", ErrInvalidCSVStructure)
	}
	return fields, nil
}

func appendKeyValue(fields *[]ExtractedField, label, value, evidence string) {
	label = strings.TrimSpace(strings.Trim(label, "|"))
	value = strings.TrimSpace(strings.Trim(value, "|"))
	if label == "" || value == "" || isTableSeparator(label) || isTableSeparator(value) {
		return
	}
	key := normalizeFieldKey(label)
	if key == "" || len(key) > 100 {
		return
	}
	*fields = append(*fields, ExtractedField{Key: key, Value: value, Evidence: evidence})
}

func splitTableLine(line, separator string) []string {
	parts := strings.Split(strings.TrimSpace(line), separator)
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			clean = append(clean, part)
		}
	}
	return clean
}

func isTableSeparator(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && strings.Trim(value, "-:|+") == ""
}

func normalizeFieldKey(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	compact := strings.Join(strings.Fields(label), " ")
	aliases := map[string]string{
		"member id": "patient_reference", "member number": "patient_reference",
		"subscriber id": "patient_reference", "subscriber number": "patient_reference",
		"patient id": "patient_reference", "patient reference": "patient_reference",
		"health plan": "payer_name", "insurance company": "payer_name", "insurance plan": "payer_name",
		"procedure": "service_type", "requested service": "service_type", "service requested": "service_type",
		"diagnosis code": "diagnosis", "icd code": "diagnosis",
		"npi": "provider_npi", "provider npi": "provider_npi",
		"ordering npi": "ordering_provider_npi", "ordering provider npi": "ordering_provider_npi",
		"servicing npi": "servicing_provider_npi", "servicing provider npi": "servicing_provider_npi",
	}
	if canonical, ok := aliases[compact]; ok {
		return canonical
	}
	var out strings.Builder
	underscore := false
	for _, r := range label {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			underscore = false
		} else if out.Len() > 0 && !underscore {
			out.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(out.String(), "_")
}
