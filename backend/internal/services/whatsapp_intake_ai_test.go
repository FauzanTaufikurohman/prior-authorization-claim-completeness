package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractWhatsAppIntakeFieldsUsesStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want bearer test-key", got)
		}
		var payload struct {
			Store    bool `json:"store"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			ResponseFormat map[string]any `json:"response_format"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Store {
			t.Error("request store = true, want false")
		}
		if len(payload.Messages) != 2 || payload.Messages[1].Content != "Payer uji, nyeri sintetis, NPI 1234567893" {
			t.Errorf("messages = %+v, want system prompt and current message only", payload.Messages)
		}
		if strings.Contains(payload.Messages[1].Content, "6281234567890") {
			t.Error("request included sender phone number")
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		jsonSchema, ok := payload.ResponseFormat["json_schema"].(map[string]any)
		if !ok {
			t.Fatal("response_format has no json_schema")
		}
		schemaData, err := json.Marshal(jsonSchema["schema"])
		if err != nil {
			t.Fatalf("marshal response schema: %v", err)
		}
		if err := json.Unmarshal(schemaData, &schema); err != nil {
			t.Fatalf("decode response schema: %v", err)
		}
		if _, ok := schema.Properties["suggested_service_type"]; !ok {
			t.Error("response schema does not expose suggested_service_type")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"payer_name\":\"Payer uji\",\"service_type\":null,\"suggested_service_type\":\"MRI\",\"ordering_provider_npi\":\"1234567893\",\"servicing_provider_npi\":null,\"symptoms\":\"nyeri sintetis\",\"diagnosis\":null}"}}]}`))
	}))
	defer server.Close()

	fields, err := extractWhatsAppIntakeFields(
		context.Background(),
		server.URL,
		OpenAIExtractor{APIKey: "test-key", Model: "test-model", HTTPClient: server.Client()},
		"Payer uji, nyeri sintetis, NPI 1234567893",
	)
	if err != nil {
		t.Fatalf("extractWhatsAppIntakeFields() error = %v", err)
	}
	if fields.PayerName == nil || *fields.PayerName != "Payer uji" ||
		fields.ServiceType != nil ||
		fields.SuggestedServiceType == nil || *fields.SuggestedServiceType != "MRI" ||
		fields.OrderingProviderNPI == nil || *fields.OrderingProviderNPI != "1234567893" ||
		fields.ServicingProviderNPI != nil ||
		fields.Symptoms == nil || *fields.Symptoms != "nyeri sintetis" ||
		fields.Diagnosis != nil {
		t.Fatalf("fields = %+v, want all supplied intake fields", fields)
	}
}

func TestExtractWhatsAppIntakeFieldsRejectsMissingKeyAndOversizedMessage(t *testing.T) {
	extractor := OpenAIExtractor{}
	if _, err := extractor.ExtractWhatsAppIntakeFields(context.Background(), "jawaban"); err != ErrOpenAIKeyMissing {
		t.Fatalf("missing key error = %v, want ErrOpenAIKeyMissing", err)
	}
	extractor.APIKey = "test-key"
	if _, err := extractor.ExtractWhatsAppIntakeFields(
		context.Background(),
		strings.Repeat("x", MaxWhatsAppIntakeMessageCharacters+1),
	); err == nil {
		t.Fatal("ExtractWhatsAppIntakeFields() accepted an oversized message")
	}
}
