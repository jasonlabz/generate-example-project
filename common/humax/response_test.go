package humax_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/generate-example-project/common/apperr"
	"github.com/jasonlabz/generate-example-project/common/humax"
)

func TestWrap_PassesThroughSingleItem(t *testing.T) {
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) (map[string]any, error) {
		return map[string]any{"id": 1}, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	assertDataJSON(t, out.Body, `{"id":1}`)
}

func TestWrap_PassesThroughList(t *testing.T) {
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return []string{"a", "b"}, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	assertDataJSON(t, out.Body, `["a","b"]`)
}

func TestWrap_MapsError(t *testing.T) {
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) (map[string]any, error) {
		return nil, errors.New("boom")
	})(context.Background(), &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != nil {
		t.Fatal("output must be nil on error")
	}
}

func TestWrap_NilSliceReturnsEmptyArray(t *testing.T) {
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return nil, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	assertDataJSON(t, out.Body, "[]")
}

func TestWrap_NilPointerReturnsZeroStruct(t *testing.T) {
	type Resp struct {
		Name string `json:"name"`
	}
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) (*Resp, error) {
		return nil, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	assertDataJSON(t, out.Body, `{"name":""}`)
}

func TestWrap_NilMapReturnsEmptyMap(t *testing.T) {
	out, err := humax.Wrap("v1", func(context.Context, *struct{}) (map[string]any, error) {
		return nil, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	assertDataJSON(t, out.Body, "{}")
}

func TestWrapList_WrapsSingleItem(t *testing.T) {
	out, err := humax.WrapList("v1", func(context.Context, *struct{}) (map[string]any, error) {
		return map[string]any{"id": 1}, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrapList: %v", err)
	}
	assertDataJSON(t, out.Body, `[{"id":1}]`)
}

func TestWrapList_NilReturnsZeroItemInArray(t *testing.T) {
	type Resp struct {
		Name string `json:"name"`
	}
	out, err := humax.WrapList("v1", func(context.Context, *struct{}) (*Resp, error) {
		return nil, nil
	})(context.Background(), &struct{}{})
	if err != nil {
		t.Fatalf("wrapList: %v", err)
	}
	assertDataJSON(t, out.Body, `[{"name":""}]`)
}

func assertDataJSON(t *testing.T, body any, want string) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var payload struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := string(payload.Data); got != want {
		t.Fatalf("data = %s, want %s", got, want)
	}
}

func TestWrap_RegistersWithHuma(t *testing.T) {
	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)

	type Resp struct {
		Name string `json:"name"`
	}
	huma.Register(api, huma.Operation{
		OperationID:   "test-wrap",
		Method:        http.MethodGet,
		Path:          "/wrap",
		DefaultStatus: http.StatusOK,
	}, humax.Wrap("v1", func(context.Context, *struct{}) (*Resp, error) {
		return &Resp{Name: "x"}, nil
	}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wrap", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := string(payload.Data); got != `{"name":"x"}` {
		t.Fatalf("data = %s, want {\"name\":\"x\"}", got)
	}
}

func TestFromError_MapsUnexpectedErrorToSafe500Envelope(t *testing.T) {
	output := humax.FromError("v1", errors.New("database password is invalid"))
	if output.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", output.GetStatus(), http.StatusInternalServerError)
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload["code"] != float64(100008001) || payload["message"] != "服务内部错误" {
		t.Fatalf("payload = %#v, want code 100008001 and safe internal message", payload)
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the internal error")
	}
}

func TestFromErrorExposesInternalCauseWhenDebugDetailsAreEnabled(t *testing.T) {
	humax.ConfigureErrorDetails(true)
	t.Cleanup(func() { humax.ConfigureErrorDetails(false) })

	output := humax.FromError("v1", errors.New("database password is invalid"))
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload["err_trace"] != "database password is invalid" {
		t.Fatalf("err_trace = %#v, want internal cause", payload["err_trace"])
	}
}

func TestFromErrorMapsRegisteredErrorToPublicContract(t *testing.T) {
	output := humax.FromError("v1", apperr.Forbidden.WithErr(nil))
	if output.GetStatus() != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", output.GetStatus(), http.StatusForbidden)
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal business error: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal business error: %v", err)
	}
	if payload["code"] != float64(100003001) || payload["message"] != "无权执行此操作" {
		t.Fatalf("payload = %#v, want forbidden catalog error", payload)
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the catalog error")
	}
}

func TestBusinessError_UsesRegisteredHTTPStatusAndCode(t *testing.T) {
	output := humax.BusinessError("v1", apperr.NotFound.Code(), "用户不存在")
	if output.GetStatus() != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", output.GetStatus(), http.StatusNotFound)
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal business error: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal business error: %v", err)
	}
	if payload["code"] != float64(100004001) || payload["message"] != "用户不存在" {
		t.Fatalf("payload = %#v, want code 100004001 and custom message", payload)
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not be present for business errors")
	}
}

func TestBusinessError_FallsBackToInvalidRequestForUnregisteredCode(t *testing.T) {
	output := humax.BusinessError("v1", 999999999, "自定义业务校验失败")
	if output.GetStatus() != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", output.GetStatus(), http.StatusBadRequest)
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal business error: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal business error: %v", err)
	}
	// 未登记的 code 回退 HTTP 语义，但保留调用方的 code 与 message，便于前端定位。
	if payload["code"] != float64(999999999) || payload["message"] != "自定义业务校验失败" {
		t.Fatalf("payload = %#v, want original code and message", payload)
	}
}

func TestBusinessError_UsesCatalogMessageWhenMessageIsEmpty(t *testing.T) {
	output := humax.BusinessError("v1", apperr.Conflict.Code(), "")
	if got := output.Error(); got != "资源状态已发生变化，请刷新后重试" {
		t.Fatalf("Error() = %q, want catalog message", got)
	}
}

func TestWrap_PreservesBusinessError(t *testing.T) {
	_, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return nil, humax.BusinessError("v1", apperr.Forbidden.Code(), "该账号无权查看此用户")
	})(context.Background(), &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}

	// Wrap 内的 MapError 必须原样保留业务错误，不能降级为 500。
	mapped, ok := err.(*humax.Error)
	if !ok {
		t.Fatalf("err type = %T, want *humax.Error", err)
	}
	if mapped.GetStatus() != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", mapped.GetStatus(), http.StatusForbidden)
	}
}

func TestFromError_ExposesFullChainForBusinessErrorInDebugMode(t *testing.T) {
	humax.ConfigureErrorDetails(true)
	t.Cleanup(func() { humax.ConfigureErrorDetails(false) })

	cause := errors.New("connection refused: 127.0.0.1:5432")
	output := humax.FromError("v1", apperr.NotFound.WithErr(cause))

	payload := marshalPayload(t, output)
	if output.GetStatus() != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", output.GetStatus(), http.StatusNotFound)
	}
	// message 保持概括性文案，详细链放进 err_trace。
	if payload["message"] != "请求的资源不存在" {
		t.Fatalf("message = %#v, want catalog message", payload["message"])
	}
	trace, _ := payload["err_trace"].(string)
	if !strings.Contains(trace, "connection refused: 127.0.0.1:5432") {
		t.Fatalf("err_trace = %q, want it to carry the inner cause", trace)
	}
	if !strings.Contains(trace, "100004001") {
		t.Fatalf("err_trace = %q, want it to carry the catalog code", trace)
	}
}

func TestFromError_FallsBackToCatalogChainWhenCauseIsAbsent(t *testing.T) {
	humax.ConfigureErrorDetails(true)
	t.Cleanup(func() { humax.ConfigureErrorDetails(false) })

	// 只有业务文案、没有内部原因时不能留空——调试时要明确看到报错来源。
	output := humax.FromError("v1", apperr.NotFound.WithMessage("用户 9999 不存在"))

	payload := marshalPayload(t, output)
	if trace, _ := payload["err_trace"].(string); !strings.Contains(trace, "用户 9999 不存在") {
		t.Fatalf("err_trace = %q, want catalog chain fallback", trace)
	}
}

func TestFromError_OmitsCauseForBusinessErrorWhenDetailsDisabled(t *testing.T) {
	humax.ConfigureErrorDetails(false)

	output := humax.FromError("v1", apperr.NotFound.WithErr(errors.New("connection refused")))

	payload := marshalPayload(t, output)
	if _, ok := payload["err_trace"]; ok {
		t.Fatalf("err_trace = %#v, want omitted when details are disabled", payload["err_trace"])
	}
}

func TestBusinessError_CarriesCodeAndMessageInDebugMode(t *testing.T) {
	humax.ConfigureErrorDetails(true)
	t.Cleanup(func() { humax.ConfigureErrorDetails(false) })

	output := humax.BusinessError("v1", apperr.BusinessRule.Code(), "暂不支持 xlsx 导出")

	payload := marshalPayload(t, output)
	if payload["err_trace"] != "100006001 -> 暂不支持 xlsx 导出" {
		t.Fatalf("err_trace = %#v, want code -> message", payload["err_trace"])
	}
}

func TestBusinessError_OmitsTraceWhenDetailsDisabled(t *testing.T) {
	humax.ConfigureErrorDetails(false)

	output := humax.BusinessError("v1", apperr.BusinessRule.Code(), "暂不支持 xlsx 导出")

	payload := marshalPayload(t, output)
	if _, ok := payload["err_trace"]; ok {
		t.Fatalf("err_trace = %#v, want omitted when details are disabled", payload["err_trace"])
	}
}

func TestConfigureHumaErrorFactory_CarriesTraceForValidationErrorInDebugMode(t *testing.T) {
	originalFactory := huma.NewErrorWithContext
	originalError := huma.NewError
	t.Cleanup(func() {
		huma.NewErrorWithContext = originalFactory
		huma.NewError = originalError
		humax.ConfigureErrorDetails(false)
	})
	humax.ConfigureErrorDetails(true)
	humax.ConfigureHumaErrorFactory("v1")

	// 参数校验的 message 仍保留明细（生产环境也要让调用方知道哪个参数错了），
	// err_trace 在调试模式下额外给出 "code -> message" 链。
	errorResponse := huma.NewErrorWithContext(nil, http.StatusBadRequest, "expected number >= 1")
	payload := marshalPayload(t, errorResponse)

	if payload["code"] != float64(100001001) {
		t.Fatalf("code = %#v, want 100001001", payload["code"])
	}
	if trace, _ := payload["err_trace"].(string); !strings.Contains(trace, "100001001") {
		t.Fatalf("err_trace = %q, want catalog code", trace)
	}
}

// reportedError 记录一次 ErrorReporter 回调。
type reportedError struct {
	message string
	status  int
}

func TestWrap_NotifiesErrorReporterWithFullChain(t *testing.T) {
	t.Cleanup(func() { humax.SetErrorReporter(nil) })

	var reported []reportedError
	humax.SetErrorReporter(func(_ context.Context, err error, status int) {
		reported = append(reported, reportedError{message: err.Error(), status: status})
	})

	// 5xx：技术故障，回调必须拿到未脱敏的完整错误链。
	if _, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return nil, fmt.Errorf("query user: %w", errors.New("connection refused: 127.0.0.1:5432"))
	})(context.Background(), &struct{}{}); err == nil {
		t.Fatal("expected error")
	}
	// 4xx：业务失败。
	if _, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return nil, apperr.NotFound.WithMessage("用户 9 不存在")
	})(context.Background(), &struct{}{}); err == nil {
		t.Fatal("expected error")
	}

	if len(reported) != 2 {
		t.Fatalf("reported %d errors, want 2", len(reported))
	}
	if reported[0].status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", reported[0].status, http.StatusInternalServerError)
	}
	if !strings.Contains(reported[0].message, "connection refused: 127.0.0.1:5432") {
		t.Fatalf("message = %q, want it to keep the full chain", reported[0].message)
	}
	if reported[1].status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", reported[1].status, http.StatusNotFound)
	}
}

func TestReportError_NotifiesReporterForUnwrappedHandlers(t *testing.T) {
	t.Cleanup(func() { humax.SetErrorReporter(nil) })

	var reported []reportedError
	humax.SetErrorReporter(func(_ context.Context, err error, status int) {
		reported = append(reported, reportedError{message: err.Error(), status: status})
	})

	// 文件流接口不走 Wrap，需显式上报；ReportError 必须原样返回错误。
	err := humax.ReportError(context.Background(), apperr.BusinessRule.WithMessage("暂不支持 xlsx 导出"))
	if err == nil {
		t.Fatal("ReportError must return the error unchanged")
	}
	if len(reported) != 1 {
		t.Fatalf("reported %d errors, want 1", len(reported))
	}
	if reported[0].status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", reported[0].status, http.StatusUnprocessableEntity)
	}
}

func TestErrorReporter_IsOptional(t *testing.T) {
	humax.SetErrorReporter(nil)

	// 未注册回调时不应 panic，行为与改造前保持一致。
	if _, err := humax.Wrap("v1", func(context.Context, *struct{}) ([]string, error) {
		return nil, errors.New("boom")
	})(context.Background(), &struct{}{}); err == nil {
		t.Fatal("expected error")
	}
	if err := humax.ReportError(context.Background(), errors.New("boom")); err == nil {
		t.Fatal("ReportError must still return the error")
	}
}

// marshalPayload 把错误响应序列化后还原为 map，便于断言字段是否存在。
func marshalPayload(t *testing.T, output any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	return payload
}

func TestConfigureHumaErrorFactory_MapsValidationErrorToBusinessEnvelope(t *testing.T) {
	originalFactory := huma.NewErrorWithContext
	originalError := huma.NewError
	t.Cleanup(func() {
		huma.NewErrorWithContext = originalFactory
		huma.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	router := gin.New()
	config := huma.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)
	huma.Post(api, "/validation", func(context.Context, *struct {
		Body struct {
			Name string `json:"name" minLength:"1"`
		}
	}) (*struct{}, error) {
		return &struct{}{}, nil
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/validation", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["code"] != float64(100001001) {
		t.Fatalf("code = %#v, want 100001001", payload["code"])
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not be present for validation errors")
	}
}

// TestConfigureHumaErrorFactory_MapsFramework422ToInvalidRequest 覆盖 huma 真实的校验失败路径：
// huma v2 用 422（不是 400）表示参数不合法，响应需保留 422，业务码归 InvalidRequest。
func TestConfigureHumaErrorFactory_MapsFramework422ToInvalidRequest(t *testing.T) {
	originalFactory := huma.NewErrorWithContext
	originalError := huma.NewError
	t.Cleanup(func() {
		huma.NewErrorWithContext = originalFactory
		huma.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	errorResponse := huma.NewErrorWithContext(nil, http.StatusUnprocessableEntity, "expected number <= 200")
	payload := marshalPayload(t, errorResponse)

	if errorResponse.GetStatus() != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", errorResponse.GetStatus(), http.StatusUnprocessableEntity)
	}
	if payload["code"] != float64(100001001) {
		t.Fatalf("code = %#v, want 100001001 (InvalidRequest)", payload["code"])
	}
}

func TestConfigureHumaErrorFactory_PreservesInternalServerError(t *testing.T) {
	originalFactory := huma.NewErrorWithContext
	originalError := huma.NewError
	t.Cleanup(func() {
		huma.NewErrorWithContext = originalFactory
		huma.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	errorResponse := huma.NewErrorWithContext(nil, http.StatusInternalServerError, "database password is invalid")
	if errorResponse.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", errorResponse.GetStatus(), http.StatusInternalServerError)
	}
	encoded, err := json.Marshal(errorResponse)
	if err != nil {
		t.Fatalf("marshal Huma error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal Huma error: %v", err)
	}
	if payload["code"] != float64(100008001) || payload["message"] != "服务内部错误" {
		t.Fatalf("payload = %#v, want safe internal error", payload)
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the internal error")
	}
}

func TestConfigureHumaErrorFactory_UsesEnvelopeForOpenAPIErrorSchema(t *testing.T) {
	originalFactory := huma.NewErrorWithContext
	originalError := huma.NewError
	t.Cleanup(func() {
		huma.NewErrorWithContext = originalFactory
		huma.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	errorResponse := huma.NewError(0, "")
	encoded, err := json.Marshal(errorResponse)
	if err != nil {
		t.Fatalf("marshal Huma error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal Huma error: %v", err)
	}
	if payload["code"] != float64(100001001) || payload["version"] != "v1" {
		t.Fatalf("payload = %#v, want shared error envelope", payload)
	}
}
