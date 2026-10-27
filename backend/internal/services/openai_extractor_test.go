package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestOpenAIExtractorRequiresAPIKey(t *testing.T) {
	extractor := OpenAIExtractor{}

	_, err := extractor.Extract(
		context.Background(),
		"synthetic text",
		[]AIFieldDefinition{{Key: "diagnosis"}},
	)
	if !errors.Is(err, ErrOpenAIKeyMissing) {
		t.Fatalf("Extract() error = %v, want %v", err, ErrOpenAIKeyMissing)
	}
}

func TestOpenAIExtractorValidatesAndCompletesFields(t *testing.T) {
	extractor := OpenAIExtractor{
		APIKey: "test-key",
		Model:  "test-model",
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != openAIChatCompletionsURL {
					t.Fatalf("request URL = %q, want %q", request.URL, openAIChatCompletionsURL)
				}
				if request.Header.Get("Authorization") != "Bearer test-key" {
					t.Fatal("OpenAI authorization header was not set")
				}

				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), `"store":false`) ||
					!strings.Contains(string(body), `"model":"test-model"`) {
					t.Fatalf("request body missing store/model settings: %s", body)
				}

				response := `{"choices":[{"message":{"content":"{\"fields\":[{\"key\":\"patient_reference\",\"value\":\"12345\",\"evidence_text\":\"Member ID: 12345\",\"confidence\":0.91}]}"}}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(response)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	results, err := extractor.Extract(
		context.Background(),
		"Member ID: 12345",
		[]AIFieldDefinition{
			{Key: "patient_reference", Description: "Member identifier"},
			{Key: "diagnosis", Description: "Diagnosis"},
		},
	)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Value == nil || *results[0].Value != "12345" {
		t.Fatalf("patient reference = %v, want 12345", results[0].Value)
	}
	if results[0].EvidenceText == nil || *results[0].EvidenceText != "Member ID: 12345" {
		t.Fatalf("evidence = %v, want exact source quote", results[0].EvidenceText)
	}
	if results[0].VerificationStatus != "pending" {
		t.Fatalf("verification status = %q, want pending", results[0].VerificationStatus)
	}
	if results[1].Value != nil || results[1].Key != "diagnosis" {
		t.Fatalf("missing field = %+v, want diagnosis with nil value", results[1])
	}
}

func TestOpenAIExtractorDropsEvidenceNotFoundInSource(t *testing.T) {
	extractor := OpenAIExtractor{
		APIKey: "test-key",
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				response := `{"choices":[{"message":{"content":"{\"fields\":[{\"key\":\"diagnosis\",\"value\":\"A09\",\"evidence_text\":\"not in the source\",\"confidence\":0.9}]}"}}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(response)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	results, err := extractor.Extract(
		context.Background(),
		"Diagnosis: B20",
		[]AIFieldDefinition{{Key: "diagnosis"}},
	)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if results[0].EvidenceText != nil {
		t.Fatalf("evidence = %q, want nil for quote absent from source", *results[0].EvidenceText)
	}
}

func TestOpenAIExtractorDoesNotExposeAPIErrorBody(t *testing.T) {
	extractor := OpenAIExtractor{
		APIKey: "test-key",
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("sensitive vendor response")),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	_, err := extractor.Extract(
		context.Background(),
		"synthetic text",
		[]AIFieldDefinition{{Key: "diagnosis"}},
	)
	if err == nil || strings.Contains(err.Error(), "sensitive vendor response") {
		t.Fatalf("Extract() error = %v, should not include response body", err)
	}
}
