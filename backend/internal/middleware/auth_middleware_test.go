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

func TestRequireStaffRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, test := range []struct {
		role       string
		wantStatus int
	}{
		{role: "staff", wantStatus: http.StatusOK},
		{role: "admin", wantStatus: http.StatusForbidden},
		{role: "reviewer", wantStatus: http.StatusForbidden},
		{role: "", wantStatus: http.StatusForbidden},
	} {
		t.Run(test.role, func(t *testing.T) {
			router := gin.New()
			router.GET("/whatsapp", func(c *gin.Context) {
				c.Set("role", test.role)
				c.Next()
			}, RequireRoles("staff"), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/whatsapp", nil))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}
