package services

import (
	"errors"
	"testing"
)

func TestParseCSVFieldsValid(t *testing.T) {
	content := []byte(
		"field,value\n" +
			"service_type,MRI\n" +
			"patient_reference,SYNTH-PAT-001\n" +
			"diagnosis,Synthetic demo diagnosis\n",
	)

	fields, err := ParseCSVFields(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	if fields[0].Key != "service_type" {
		t.Errorf("unexpected key: %q", fields[0].Key)
	}

	if fields[0].Value != "MRI" {
		t.Errorf("unexpected value: %q", fields[0].Value)
	}

	if fields[0].Evidence != "service_type: MRI" {
		t.Errorf("unexpected evidence: %q", fields[0].Evidence)
	}
}

func TestParseCSVFieldsWithBOM(t *testing.T) {
	content := append(
		[]byte{0xEF, 0xBB, 0xBF},
		[]byte("field,value\nservice_type,MRI\n")...,
	)

	fields, err := ParseCSVFields(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 1 || fields[0].Value != "MRI" {
		t.Fatalf("unexpected result: %+v", fields)
	}
}

func TestParseCSVFieldsInvalidHeader(t *testing.T) {
	content := []byte("name,value\nservice_type,MRI\n")

	_, err := ParseCSVFields(content)
	if !errors.Is(err, ErrInvalidCSVStructure) {
		t.Fatalf("expected ErrInvalidCSVStructure, got %v", err)
	}
}

func TestParseCSVFieldsMissingKey(t *testing.T) {
	content := []byte("field,value\n,MRI\n")

	_, err := ParseCSVFields(content)
	if !errors.Is(err, ErrInvalidCSVField) {
		t.Fatalf("expected ErrInvalidCSVField, got %v", err)
	}
}

func TestParseCSVFieldsMalformedCSV(t *testing.T) {
	content := []byte("field,value\nservice_type,\"MRI\n")

	_, err := ParseCSVFields(content)
	if !errors.Is(err, ErrInvalidCSVStructure) {
		t.Fatalf("expected ErrInvalidCSVStructure, got %v", err)
	}
}

func TestParseCSVFieldsEmptyData(t *testing.T) {
	content := []byte("field,value\n")

	_, err := ParseCSVFields(content)
	if !errors.Is(err, ErrInvalidCSVStructure) {
		t.Fatalf("expected ErrInvalidCSVStructure, got %v", err)
	}
}
