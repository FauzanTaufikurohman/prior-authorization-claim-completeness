package routes

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := SetupRouter(nil)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/health",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status ok, got %q", response["status"])
	}
}

func TestDatabaseReadinessWithoutPool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := SetupRouter(nil)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/health/ready",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			recorder.Code,
		)
	}
}

func TestPublicUploadRoutesDoNotRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(nil)

	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/public/pa-upload"},
		{method: http.MethodPost, path: "/api/public/pa-upload/documents"},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s returned %d, want %d without a database", test.method, test.path, recorder.Code, http.StatusServiceUnavailable)
		}
	}
}

func TestWhatsAppRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(nil)
	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"POST /api/webhooks/whatsapp",
		"GET /api/whatsapp/intake-activity",
		"GET /api/whatsapp/device-pairing",
		"POST /api/whatsapp/device-pairing",
		"POST /api/whatsapp/device-pairing/webhook",
		"POST /api/whatsapp/device-pairing/qr",
		"DELETE /api/whatsapp/device-pairing",
		"GET /api/whatsapp/device-pairing/:id/status",
	} {
		if !registered[route] {
			t.Errorf("route %q is not registered", route)
		}
	}
}

func TestWhatsAppWebhookLogsStatusWithoutLoggingQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(nil)
	var output bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previousOutput)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/webhooks/whatsapp?secret=do-not-log-this",
		nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("webhook status = %d, want %d without a database", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(output.String(), "status=503") {
		t.Fatalf("webhook log = %q, want status", output.String())
	}
	if strings.Contains(output.String(), "do-not-log-this") {
		t.Fatalf("webhook log exposed query secret: %q", output.String())
	}
}
