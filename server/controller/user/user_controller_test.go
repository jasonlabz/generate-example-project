package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"

	"github.com/jasonlabz/generate-example-project/common/apperr"
	"github.com/jasonlabz/generate-example-project/common/humax"
	service_mocks "github.com/jasonlabz/generate-example-project/mocks/server/service/user"
	"github.com/jasonlabz/generate-example-project/server/service/user"
)

// newJSONRequest 构造带 JSON Content-Type 的请求，huma 据此选择请求体的解析方式。
func newJSONRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

// newUserRouter 用 gin + huma 组装 user 模块的测试路由，service 由 mock 提供。
func newUserRouter(t *testing.T) (*gin.Engine, *service_mocks.MockService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)

	userService := service_mocks.NewMockService(gomock.NewController(t))
	NewController(userService).Register(api)

	return router, userService
}

func TestController_Get_ReturnsSuccessEnvelope(t *testing.T) {
	router, userService := newUserRouter(t)
	userService.EXPECT().Get(gomock.Any(), int64(1)).
		Return(user.User{ID: 1, Name: "alice", Email: "alice@example.com"}, nil)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Code int    `json:"code"`
		Data userVO `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != 0 || payload.Data.Name != "alice" {
		t.Fatalf("payload = %#v, want success envelope with alice", payload)
	}
}

// TestController_Get_MapsNotFoundToHTTPStatus 验证错误码不会被降级成 500：
// service 抛出 apperr.NotFound，接口必须返回 404 与目录登记的 code。
func TestController_Get_MapsNotFoundToHTTPStatus(t *testing.T) {
	router, userService := newUserRouter(t)
	userService.EXPECT().Get(gomock.Any(), int64(7)).
		Return(user.User{}, apperr.NotFound.WithMessage("用户 7 不存在"))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/7", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != apperr.NotFound.Code() {
		t.Fatalf("code = %d, want %d", payload.Code, apperr.NotFound.Code())
	}
	if payload.Message != "用户 7 不存在" {
		t.Fatalf("message = %q, want service message", payload.Message)
	}
}

// TestController_Get_ExposesInnerCauseInDebugMode 验证 4xx 业务错误也能带出内部原因：
// service 用 apperr.NotFound.WithErr 注入真实故障时，调试模式下 err_trace 可见，
// 便于排查"业务上查不到"到底是没数据还是下游挂了。
func TestController_Get_ExposesInnerCauseInDebugMode(t *testing.T) {
	humax.ConfigureErrorDetails(true)
	t.Cleanup(func() { humax.ConfigureErrorDetails(false) })

	router, userService := newUserRouter(t)
	userService.EXPECT().Get(gomock.Any(), int64(7)).Return(user.User{},
		apperr.NotFound.WithErr(errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/7", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Code     int    `json:"code"`
		ErrTrace string `json:"err_trace"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != apperr.NotFound.Code() {
		t.Fatalf("code = %d, want %d", payload.Code, apperr.NotFound.Code())
	}
	// message 是概括性文案，err_trace 是完整报错链（含目录 code 与内部原因）。
	if !strings.Contains(payload.ErrTrace, "dial tcp 10.0.0.5:5432: connect: connection refused") {
		t.Fatalf("err_trace = %q, want it to carry the inner cause", payload.ErrTrace)
	}
	if !strings.Contains(payload.ErrTrace, "100004001") {
		t.Fatalf("err_trace = %q, want it to carry the catalog code", payload.ErrTrace)
	}
}

func TestController_List_ReturnsPagination(t *testing.T) {
	router, userService := newUserRouter(t)
	userService.EXPECT().List(gomock.Any(), "", int64(0), int64(200)).
		Return([]user.User{{ID: 1, Name: "alice"}}, int64(1), nil)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Code       int      `json:"code"`
		Data       []userVO `json:"data"`
		Pagination struct {
			Page      int64 `json:"page"`
			PageSize  int64 `json:"page_size"`
			PageCount int64 `json:"page_count"`
			Total     int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// 分页默认值：第一页、每页 200 条。
	if payload.Pagination.Page != 1 || payload.Pagination.PageSize != 200 {
		t.Fatalf("pagination = %#v, want page 1 size 200", payload.Pagination)
	}
	if payload.Pagination.Total != 1 || payload.Pagination.PageCount != 1 {
		t.Fatalf("pagination = %#v, want total 1 pageCount 1", payload.Pagination)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data = %#v, want one item", payload.Data)
	}
}

func TestController_List_RejectsOversizedPageSize(t *testing.T) {
	router, _ := newUserRouter(t)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users?page_size=201", nil))

	if response.Code != http.StatusUnprocessableEntity && response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 or 422 (page_size exceeds maximum)", response.Code)
	}
}

func TestController_Create_MapsConflictToHTTPStatus(t *testing.T) {
	router, userService := newUserRouter(t)
	userService.EXPECT().Create(gomock.Any(), "alice", "alice@example.com").
		Return(user.User{}, apperr.Conflict.WithMessage("用户名 alice 已存在"))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, newJSONRequest(t, http.MethodPost, "/users",
		`{"name":"alice","email":"alice@example.com"}`))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != apperr.Conflict.Code() {
		t.Fatalf("code = %d, want %d", payload.Code, apperr.Conflict.Code())
	}
}

// TestController_Export_RejectsUnsupportedFormat 覆盖 BusinessError 的典型场景：
// 参数本身通过了 huma 的 enum 校验，但业务上不支持，因此不走 RFC7807，
// 而是返回统一信封 + 422 + 业务 code。
func TestController_Export_RejectsUnsupportedFormat(t *testing.T) {
	router, _ := newUserRouter(t)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/export?format=xlsx", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != apperr.BusinessRule.Code() {
		t.Fatalf("code = %d, want %d", payload.Code, apperr.BusinessRule.Code())
	}
}

func TestController_Export_StreamsCSVFile(t *testing.T) {
	router, userService := newUserRouter(t)
	userService.EXPECT().Export(gomock.Any()).
		Return([]user.User{{ID: 1, Name: "alice", Email: "alice@example.com"}}, nil)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/export?format=csv", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition == "" {
		t.Fatal("Content-Disposition must be set for file download")
	}
	if body := response.Body.String(); body == "" || body[:3] != "id," {
		t.Fatalf("body = %q, want CSV content starting with header", body)
	}
}

func TestController_Register_PublishesDocumentedStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)

	NewController(nil).Register(api)

	// 文档必须声明真实的失败状态，否则前端按 OpenAPI 生成的 client 会漏处理分支。
	wantStatuses := map[string][]string{
		"/users":        {"200", "500"},
		"/users/{id}":   {"200", "404", "500"},
		"/users/export": {"200", "422", "500"},
	}
	for path, want := range wantStatuses {
		pathItem := api.OpenAPI().Paths[path]
		if pathItem == nil {
			t.Fatalf("path %s was not registered", path)
		}
		documented := map[string]bool{}
		if pathItem.Get != nil {
			for status := range pathItem.Get.Responses {
				documented[status] = true
			}
		}
		if pathItem.Post != nil {
			for status := range pathItem.Post.Responses {
				documented[status] = true
			}
		}
		for _, status := range want {
			if !documented[status] {
				t.Errorf("path %s missing documented status %s (got %v)", path, status, documented)
			}
		}
	}
}
