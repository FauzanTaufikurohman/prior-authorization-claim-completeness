package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
)

func TestWhatsAppIntakeConsentAndConfirmationAreExplicit(t *testing.T) {
	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "SETUJU", want: true},
		{message: "Saya setuju", want: true},
		{message: "setuju dengan diagnosis", want: false},
	} {
		if got := isWhatsAppIntakeConsent(strings.ToUpper(test.message)); got != test.want {
			t.Errorf("isWhatsAppIntakeConsent(%q) = %t, want %t", test.message, got, test.want)
		}
	}

	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "YA", want: true},
		{message: "Iya.", want: true},
		{message: "Setuju", want: true},
		{message: "KONFIRMASI", want: true},
		{message: "KONFIRMASI BUAT PA", want: true},
		{message: "TIDAK", want: false},
		{message: "Tolong konfirmasi payer", want: false},
	} {
		if got := isWhatsAppIntakeConfirmation(test.message); got != test.want {
			t.Errorf("isWhatsAppIntakeConfirmation(%q) = %t, want %t", test.message, got, test.want)
		}
	}

	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "TIDAK", want: true},
		{message: "Tidak setuju.", want: true},
		{message: "Jangan buat PA", want: true},
		{message: "Tolong perbaiki diagnosis", want: false},
	} {
		if got := isWhatsAppIntakeRejection(test.message); got != test.want {
			t.Errorf("isWhatsAppIntakeRejection(%q) = %t, want %t", test.message, got, test.want)
		}
	}
}

func TestWhatsAppIntakeNewRequestCommand(t *testing.T) {
	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "MULAI", want: true},
		{message: "mulai.", want: true},
		{message: "Buat PA baru", want: true},
		{message: "PA BARU", want: true},
		{message: "buat pa", want: false},
		{message: "tolong buat PA baru", want: false},
	} {
		if got := isWhatsAppIntakeNewRequest(test.message); got != test.want {
			t.Errorf("isWhatsAppIntakeNewRequest(%q) = %t, want %t", test.message, got, test.want)
		}
	}
}

func TestWhatsAppIntakeHelpCommandsAndInstructions(t *testing.T) {
	for _, command := range []string{"BANTUAN", "menu", "HELP", "?"} {
		if !isWhatsAppIntakeHelp(command) {
			t.Errorf("isWhatsAppIntakeHelp(%q) = false, want true", command)
		}
	}
	message := whatsAppIntakeHelpMessage()
	for _, instruction := range []string{"SETUJU", "MULAI", "BUAT PA BARU", "BANTUAN", "ID pasien sintetis"} {
		if !strings.Contains(message, instruction) {
			t.Errorf("help message %q does not contain %q", message, instruction)
		}
	}
}

func TestWhatsAppIntakePromptsForFirstMissingField(t *testing.T) {
	data := whatsAppIntakeData{}
	if got := missingWhatsAppIntakeQuestion(data); !strings.Contains(got, "payer") {
		t.Fatalf("first missing question = %q, want payer", got)
	}
	data.PayerName = "Payer sintetis"
	if got := missingWhatsAppIntakeQuestion(data); !strings.Contains(got, "Ceritakan keluhan") {
		t.Fatalf("second missing question = %q, want symptoms", got)
	}
	data.ServiceType = "MRI"
	data.PatientReference = "SYN-WA-TEST"
	data.Symptoms = "Keluhan sintetis"
	if got := missingWhatsAppIntakeQuestion(data); got != "" {
		t.Fatalf("missing question after required fields = %q, want empty", got)
	}
	confirmation := formatWhatsAppIntakeConfirmation(data)
	if !strings.Contains(confirmation, "Balas YA") || !strings.Contains(confirmation, "TIDAK") {
		t.Fatalf("confirmation prompt = %q, want explicit YA/TIDAK choices", confirmation)
	}
	if !strings.Contains(confirmation, "NPI ordering provider: Belum diisi") ||
		!strings.Contains(confirmation, "ID sintetis") ||
		!strings.Contains(confirmation, "Staff tetap perlu memeriksa") {
		t.Fatalf("confirmation prompt = %q, want provider NPI, synthetic ID, and review notice", confirmation)
	}
}

func TestWhatsAppIntakeUsesSymptomsWhenDiagnosisIsUnknown(t *testing.T) {
	data := whatsAppIntakeData{Symptoms: "Keluhan sintetis"}
	if got := formatWhatsAppPADiagnosis(data); got != "Keluhan yang dilaporkan: Keluhan sintetis" {
		t.Fatalf("formatWhatsAppPADiagnosis() = %q, want labeled symptom", got)
	}
	data.Diagnosis = "Diagnosis yang disebutkan"
	if got := formatWhatsAppPADiagnosis(data); got != "Diagnosis yang disebutkan\nKeluhan yang dilaporkan: Keluhan sintetis" {
		t.Fatalf("formatWhatsAppPADiagnosis() = %q, want diagnosis and symptom", got)
	}
}

func TestGenerateWhatsAppSyntheticPatientReference(t *testing.T) {
	first, err := generateWhatsAppSyntheticPatientReference()
	if err != nil {
		t.Fatalf("generateWhatsAppSyntheticPatientReference() error = %v", err)
	}
	second, err := generateWhatsAppSyntheticPatientReference()
	if err != nil {
		t.Fatalf("generateWhatsAppSyntheticPatientReference() error = %v", err)
	}
	if !strings.HasPrefix(first, "SYN-WA-") || first == second {
		t.Fatalf("generated references = %q and %q, want distinct SYN-WA references", first, second)
	}
}

func TestWhatsAppAIErrorCodeDoesNotExposeProviderDetails(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: services.ErrOpenAIKeyMissing, want: "missing_api_key"},
		{err: &services.OpenAIAPIError{StatusCode: 401}, want: "authentication"},
		{err: &services.OpenAIAPIError{StatusCode: 404}, want: "model_not_found"},
		{err: &services.OpenAIAPIError{StatusCode: 429}, want: "rate_limited"},
		{err: &services.OpenAIAPIError{StatusCode: 503}, want: "provider_error"},
		{err: context.DeadlineExceeded, want: "timeout"},
		{err: errors.New("invalid OpenAI response"), want: "invalid_response"},
	} {
		if got := whatsAppAIErrorCode(test.err); got != test.want {
			t.Errorf("whatsAppAIErrorCode(%T) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestMergeWhatsAppIntakeFieldsAcceptsCorrectionsAndRejectsOversizedValues(t *testing.T) {
	data := whatsAppIntakeData{PayerName: "Payer lama"}
	correction := "Payer baru"
	orderingNPI := "1234567893"
	suggestedService := "MRI"
	symptoms := "Nyeri sintetis"
	if err := mergeWhatsAppIntakeFields(&data, services.WhatsAppIntakeFields{
		PayerName: &correction, OrderingProviderNPI: &orderingNPI,
		SuggestedServiceType: &suggestedService, Symptoms: &symptoms,
	}); err != nil {
		t.Fatalf("mergeWhatsAppIntakeFields() error = %v", err)
	}
	if data.PayerName != correction || data.OrderingProviderNPI != orderingNPI ||
		data.ServiceType != suggestedService || !data.ServiceTypeSuggested || data.Symptoms != symptoms {
		t.Fatalf("data = %+v, want corrected payer, provider NPI, symptoms, and suggested service", data)
	}
	explicitService := "Layanan pilihan dokter"
	if err := mergeWhatsAppIntakeFields(&data, services.WhatsAppIntakeFields{ServiceType: &explicitService}); err != nil {
		t.Fatalf("mergeWhatsAppIntakeFields() explicit service error = %v", err)
	}
	if data.ServiceType != explicitService || data.ServiceTypeSuggested {
		t.Fatalf("data = %+v, want explicit service to replace AI suggestion", data)
	}

	oversized := strings.Repeat("x", 151)
	if err := mergeWhatsAppIntakeFields(&data, services.WhatsAppIntakeFields{PayerName: &oversized}); err == nil {
		t.Fatal("mergeWhatsAppIntakeFields() accepted an oversized payer")
	}
	invalidNPI := "12345"
	if err := mergeWhatsAppIntakeFields(&data, services.WhatsAppIntakeFields{OrderingProviderNPI: &invalidNPI}); err == nil {
		t.Fatal("mergeWhatsAppIntakeFields() accepted an invalid NPI")
	}
}

func TestBuildWhatsAppUploadURLUsesFrontendURL(t *testing.T) {
	frontendURL := "https://frontend.example/app"
	got, err := buildWhatsAppUploadURL(frontendURL, "upload-token")
	if err != nil {
		t.Fatalf("buildWhatsAppUploadURL() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "frontend.example" || parsed.Path != "/app/upload-user" || parsed.Fragment != "upload-token" {
		t.Fatalf("upload URL = %q, unexpected components", got)
	}
	if parsed.RawQuery != "" {
		t.Fatal("public upload URL unexpectedly contains query parameters")
	}
}

func TestBuildWhatsAppUploadURLRejectsInvalidFrontendURL(t *testing.T) {
	for _, frontendURL := range []string{
		"",
		"//frontend.example",
		"ftp://frontend.example",
		"https://user:password@frontend.example",
		"https://frontend.example?token=unexpected",
	} {
		if _, err := buildWhatsAppUploadURL(frontendURL, "upload-token"); err == nil {
			t.Errorf("buildWhatsAppUploadURL(%q) succeeded, want invalid frontend URL error", frontendURL)
		}
	}
}

func TestConfiguredWhatsAppWebhookSecretUsesURLParameter(t *testing.T) {
	secret := strings.Repeat("s", 32)
	t.Setenv("FONNTE_WEBHOOK_URL", "https://example.test/api/webhooks/whatsapp?secret="+url.QueryEscape(secret))
	if got := configuredWhatsAppWebhookSecret(); got != secret {
		t.Fatal("configuredWhatsAppWebhookSecret() did not use URL secret")
	}
}

func TestMaskWhatsAppSender(t *testing.T) {
	if got := maskWhatsAppSender("6281234567890"); got != "****7890" {
		t.Fatalf("maskWhatsAppSender() = %q, want masked suffix", got)
	}
	if got := maskWhatsAppSender("1234"); got != "****" {
		t.Fatalf("maskWhatsAppSender() for short number = %q, want ****", got)
	}
}

func TestWhatsAppWebhookPayloadSupportsButtonText(t *testing.T) {
	var input whatsAppWebhookMessage
	if err := json.Unmarshal([]byte(`{
		"device": "6281234567890",
		"sender": "6281234567891",
		"message": "",
		"text": "SETUJU"
	}`), &input); err != nil {
		t.Fatalf("decode Fonnte webhook payload: %v", err)
	}
	if input.Message == "" {
		input.Message = input.Text
	}
	if input.Message != "SETUJU" {
		t.Fatalf("message = %q, want button text", input.Message)
	}
}

func TestWhatsAppWebhookPayloadAcceptsNumericInboxID(t *testing.T) {
	var input whatsAppWebhookMessage
	if err := json.Unmarshal([]byte(`{
		"device": "6281234567890",
		"sender": "6281234567891",
		"message": "SETUJU",
		"inboxid": 123456789012345
	}`), &input); err != nil {
		t.Fatalf("decode Fonnte webhook payload: %v", err)
	}
	if got := string(input.InboxID); got != "123456789012345" {
		t.Fatalf("inbox ID = %q, want exact numeric ID as text", got)
	}
}

func TestWhatsAppWebhookDecodeDiagnosticDoesNotIncludePayloadValues(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		wantReason string
		wantField  string
		wantType   string
		wantOffset int64
	}{
		{
			name:       "field type mismatch",
			payload:    `{"device":123,"sender":"6281234567890","message":"private text"}`,
			wantReason: "type_mismatch",
			wantField:  "device",
			wantType:   "string",
			wantOffset: 13,
		},
		{
			name:       "invalid syntax",
			payload:    `{"device":}`,
			wantReason: "syntax_error",
			wantOffset: 11,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var input whatsAppWebhookMessage
			err := json.Unmarshal([]byte(test.payload), &input)
			if err == nil {
				t.Fatal("json.Unmarshal() succeeded, want error")
			}
			reason, field, expectedType, offset := whatsAppWebhookDecodeDiagnostic(err)
			if reason != test.wantReason || field != test.wantField ||
				expectedType != test.wantType || offset != test.wantOffset {
				t.Fatalf(
					"diagnostic = (%q, %q, %q, %d), want (%q, %q, %q, %d)",
					reason, field, expectedType, offset,
					test.wantReason, test.wantField, test.wantType, test.wantOffset,
				)
			}
		})
	}
}
