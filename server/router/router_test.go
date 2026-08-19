package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasonlabz/generate-example-project/bootstrap"
)

// newTestAPIRouter 用确定性的服务名与 gin 调试模式构造路由。
//
// InitApiRouter 从 bootstrap 全局配置读取服务名、用 gin.IsDebugging() 决定是否
// 注册文档端点，因此测试需在调用前覆盖这两处全局状态。
func newTestAPIRouter(t *testing.T, debug bool) *gin.Engine {
	t.Helper()

	bootstrap.GetConfig().Application.Name = "example"

	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	t.Cleanup(func() { gin.SetMode(gin.DebugMode) })

	router, err := InitApiRouter()
	if err != nil {
		t.Fatalf("InitApiRouter() error = %v", err)
	}

	return router
}

func TestInitApiRouter_HealthCheck(t *testing.T) {
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

// TestInitApiRouter_ReadinessCheck verifies that the health-check domain also
// exposes its independently assembled readiness use case.
func TestInitApiRouter_ReadinessCheck(t *testing.T) {
	router := newTestAPIRouter(t, true)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readiness-check", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /readiness-check status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code int      `json:"code"`
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("GET /readiness-check response: %v", err)
	}
	if response.Code != 0 || len(response.Data) != 1 || response.Data[0] != "ready" {
		t.Fatalf("GET /readiness-check response = %#v, want code 0 and [ready]", response)
	}
}

func TestInitApiRouter_DebugOpenAPIDocument(t *testing.T) {
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

func TestInitApiRouter_DebugDocumentationPage(t *testing.T) {
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

func TestInitApiRouter_ProductionHidesOpenAPIDocument(t *testing.T) {
	router := newTestAPIRouter(t, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/example/v3/api-docs", nil))

	if recorder.Code != http.StatusNotFound {
		t.Errorf("GET /example/v3/api-docs status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestInitApiRouter_DoesNotRegisterLegacyStaticFiles(t *testing.T) {
	router := newTestAPIRouter(t, false)

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
