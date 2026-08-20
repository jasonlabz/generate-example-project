package health_check

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	service_mocks "github.com/jasonlabz/generate-example-project/mocks/server/service/health_check"
	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
	"go.uber.org/mock/gomock"
)

func TestController_Register_ReturnsSuccessEnvelope(t *testing.T) {
	router, healthService := newHealthCheckRouter(t)
	healthService.EXPECT().Check(gomock.Any()).Return(service.Result{Status: "success"}, nil)

	request := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertSuccessEnvelope(t, response, "success")
}

func TestController_Register_ReturnsReadinessEnvelope(t *testing.T) {
	router, healthService := newHealthCheckRouter(t)
	healthService.EXPECT().CheckReadiness(gomock.Any()).Return(service.Result{Status: "ready"}, nil)

	request := httptest.NewRequest(http.MethodGet, "/readiness-check", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertSuccessEnvelope(t, response, "ready")
}

func TestController_Register_AdaptsServiceFailure(t *testing.T) {
	router, healthService := newHealthCheckRouter(t)
	serviceErr := errors.New("health dependency unavailable")
	healthService.EXPECT().Check(gomock.Any()).Return(service.Result{}, serviceErr)

	request := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	var payload struct {
		Code     int    `json:"code"`
		Message  string `json:"message"`
		ErrTrace string `json:"err_trace"`
		Version  string `json:"version"`
		Data     []any  `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != 0 || payload.Message != serviceErr.Error() || payload.ErrTrace != serviceErr.Error() {
		t.Fatalf("payload = %#v, want legacy error envelope", payload)
	}
	if payload.Version != "v1" || payload.Data == nil {
		t.Fatalf("payload = %#v, want version v1 and empty data", payload)
	}
}

func TestController_Register_PublishesOpenAPIMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)

	controller := NewController(nil)
	controller.Register(api)

	healthOperation := api.OpenAPI().Paths["/health-check"].Get
	if healthOperation == nil {
		t.Fatal("health-check operation was not registered")
	}
	if healthOperation.OperationID != "health-check" || healthOperation.Summary != "健康检查" {
		t.Fatalf("health-check metadata = %#v, want health-check metadata", healthOperation)
	}

	readinessOperation := api.OpenAPI().Paths["/readiness-check"].Get
	if readinessOperation == nil {
		t.Fatal("readiness-check operation was not registered")
	}
	if readinessOperation.OperationID != "readiness-check" || readinessOperation.Summary != "就绪检查" {
		t.Fatalf("readiness-check metadata = %#v, want readiness-check metadata", readinessOperation)
	}
}

func newHealthCheckRouter(t *testing.T) (*gin.Engine, *service_mocks.MockService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)

	mockController := gomock.NewController(t)
	healthService := service_mocks.NewMockService(mockController)
	NewController(healthService).Register(api)

	return router, healthService
}

func assertSuccessEnvelope(t *testing.T, response *httptest.ResponseRecorder, status string) {
	t.Helper()

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload struct {
		Code    int      `json:"code"`
		Version string   `json:"version"`
		Data    []string `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != 0 || payload.Version != "v1" {
		t.Fatalf("payload = %#v, want success envelope for v1", payload)
	}
	if len(payload.Data) != 1 || payload.Data[0] != status {
		t.Fatalf("data = %#v, want [%s]", payload.Data, status)
	}
	if response.Header().Get("Link") != "" {
		t.Fatalf("Link = %q, want empty", response.Header().Get("Link"))
	}
}
