package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestAPIRouter builds an isolated router without reading global configuration.
func newTestAPIRouter(t *testing.T, debug bool) http.Handler {
	t.Helper()

	router, err := newAPIRouter("example", debug)
	if err != nil {
		t.Fatalf("newAPIRouter() error = %v", err)
	}

	return router
}

func TestNewAPIRouter_HealthCheck(t *testing.T) {
	router := newTestAPIRouter(t, true)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health-check", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /health-check status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code        int      `json:"code"`
		Version     string   `json:"version"`
		CurrentTime string   `json:"current_time"`
		Data        []string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("GET /health-check response is not JSON: %v", err)
	}
	if response.Code != 0 {
		t.Errorf("GET /health-check code = %d, want 0", response.Code)
	}
	if response.Version != "v1" {
		t.Errorf("GET /health-check version = %q, want %q", response.Version, "v1")
	}
	if _, err := time.Parse(time.DateTime, response.CurrentTime); err != nil {
		t.Errorf("GET /health-check current_time = %q, want time.DateTime: %v", response.CurrentTime, err)
	}
	if len(response.Data) != 1 || response.Data[0] != "success" {
		t.Errorf("GET /health-check data = %#v, want []string{\"success\"}", response.Data)
	}
}

func TestNewAPIRouter_DebugOpenAPIDocument(t *testing.T) {
	router := newTestAPIRouter(t, true)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/example/v3/api-docs", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /example/v3/api-docs status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var document struct {
		OpenAPI string `json:"openapi"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("GET /example/v3/api-docs response is not JSON: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Errorf("GET /example/v3/api-docs openapi = %q, want %q", document.OpenAPI, "3.0.3")
	}
}

func TestNewAPIRouter_DebugDocumentationPage(t *testing.T) {
	router := newTestAPIRouter(t, true)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/example/doc.html", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /example/doc.html status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "knife4j-vue") {
		t.Error("GET /example/doc.html body does not contain knife4j-vue")
	}
}

func TestNewAPIRouter_ProductionHidesOpenAPIDocument(t *testing.T) {
	router := newTestAPIRouter(t, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/example/v3/api-docs", nil))

	if recorder.Code != http.StatusNotFound {
		t.Errorf("GET /example/v3/api-docs status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
