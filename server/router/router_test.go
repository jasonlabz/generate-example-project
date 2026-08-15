package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	controller_mocks "github.com/jasonlabz/generate-example-project/mocks/server/controller/health_check"
	route_module "github.com/jasonlabz/generate-example-project/server/module"
	"go.uber.org/mock/gomock"
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
	if link := recorder.Header().Get("Link"); link != "" {
		t.Errorf("GET /health-check Link header = %q, want empty", link)
	}

	var rawResponse map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &rawResponse); err != nil {
		t.Fatalf("GET /health-check response is not JSON: %v", err)
	}
	if _, ok := rawResponse["$schema"]; ok {
		t.Error("GET /health-check response contains unexpected $schema field")
	}
	wantKeys := []string{"code", "version", "current_time", "data"}
	if len(rawResponse) != len(wantKeys) {
		t.Errorf("GET /health-check top-level key count = %d, want %d", len(rawResponse), len(wantKeys))
	}
	for _, key := range wantKeys {
		if _, ok := rawResponse[key]; !ok {
			t.Errorf("GET /health-check response is missing top-level key %q", key)
		}
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

func TestNewAPIRouter_DoesNotRegisterLegacyStaticFiles(t *testing.T) {
	router, err := newAPIRouter("example", false)
	if err != nil {
		t.Fatalf("newAPIRouter() error = %v", err)
	}

	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/server/") {
			t.Errorf("legacy static route = %q, want none", route.Path)
		}
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/server/", nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("GET /server/ status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRegisterRootAPI_DelegatesToHealthCheckController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.CreateHooks = nil
	api := humagin.New(router, config)

	mockController := gomock.NewController(t)
	healthCheckController := controller_mocks.NewMockHealthCheckController(mockController)
	healthCheckController.EXPECT().Register(gomock.Any())

	registerRootAPI(api, route_module.Module{RegisterRoot: healthCheckController.Register})
}
