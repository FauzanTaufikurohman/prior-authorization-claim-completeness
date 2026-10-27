package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
