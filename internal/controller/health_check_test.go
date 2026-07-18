package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/health-check", HealthCheck)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expect 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "success") {
		t.Fatalf("expect body contains success, got %s", body)
	}
}
