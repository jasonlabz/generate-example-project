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
	service_mocks "github.com/jasonlabz/generate-example-project/server/mocks/server/service/health_check"
	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
	"go.uber.org/mock/gomock"
)

func TestController_Register_ReturnsSuccessEnvelope(t *testing.T) {
	router, healthCheckService := newHealthCheckRouter(t)
	healthCheckService.EXPECT().Check(gomock.Any()).Return(service.HealthCheckResult{Status: "success"}, nil)

	request := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload struct {
		Code        int      `json:"code"`
		Version     string   `json:"version"`
		CurrentTime string   `json:"current_time"`
		Data        []string `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != 0 || payload.Version != "v1" {
		t.Fatalf("payload = %#v, want success envelope for v1", payload)
	}
	if len(payload.Data) != 1 || payload.Data[0] != "success" {
		t.Fatalf("data = %#v, want [success]", payload.Data)
	}
	if response.Header().Get("Link") != "" {
		t.Fatalf("Link = %q, want empty", response.Header().Get("Link"))
	}
}

func TestController_Register_AdaptsServiceFailure(t *testing.T) {
	router, healthCheckService := newHealthCheckRouter(t)
	serviceErr := errors.New("health dependency unavailable")
	healthCheckService.EXPECT().Check(gomock.Any()).Return(service.HealthCheckResult{}, serviceErr)

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

func newHealthCheckRouter(t *testing.T) (*gin.Engine, *service_mocks.MockHealthCheckService) {
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
	healthCheckService := service_mocks.NewMockHealthCheckService(mockController)
	controller := NewHealthCheckController(healthCheckService)
	controller.Register(api)

	return router, healthCheckService
}
