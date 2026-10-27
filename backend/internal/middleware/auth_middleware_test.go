package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{name: "admin allowed", role: "admin", wantStatus: http.StatusOK},
		{name: "staff denied", role: "staff", wantStatus: http.StatusForbidden},
		{name: "reviewer denied", role: "reviewer", wantStatus: http.StatusForbidden},
		{name: "missing role denied", role: "", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin", func(c *gin.Context) {
				c.Set("role", tt.role)
				c.Next()
			}, RequireRoles("admin"), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/admin", nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d",
					tt.wantStatus, recorder.Code)
			}
		})
	}
}
