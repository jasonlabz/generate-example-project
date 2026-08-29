package humax_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
