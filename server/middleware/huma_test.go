package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/potato/consts"

	"github.com/jasonlabz/generate-example-project/common/resource"
)

type humaContextResponse struct {
	Custom   string `json:"custom"`
	Computed string `json:"computed"`
	TraceID  string `json:"trace_id"`
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	ClientIP string `json:"client_ip"`
}

type humaContextOutput struct {
	Body humaContextResponse
}

type humaRequestLogOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func TestSetHumaContext_PropagatesPotatoValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, huma.DefaultConfig("test", "v1"))
	api.UseMiddleware(SetHumaContext(
		WithHumaHeaderField(map[string]string{"X-Custom": "custom"}),
		WithHumaCustomField(map[string]func(huma.Context) string{
			"computed": func(ctx huma.Context) string {
				return ctx.Header("X-Computed")
			},
		}),
	))
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		ginContext := humagin.Unwrap(ctx)
		for _, key := range []string{"custom", "computed", consts.ContextTraceID} {
			if _, exists := ginContext.Get(key); exists {
				t.Errorf("Gin context unexpectedly contains %q", key)
			}
		}
		next(ctx)
	})

	huma.Get(api, "/context", func(ctx context.Context, _ *struct{}) (*humaContextOutput, error) {
		return &humaContextOutput{Body: humaContextResponse{
			Custom:   stringValue(ctx, "custom"),
			Computed: stringValue(ctx, "computed"),
			TraceID:  stringValue(ctx, consts.ContextTraceID),
			UserID:   stringValue(ctx, consts.ContextUserID),
			Token:    stringValue(ctx, consts.ContextToken),
			ClientIP: stringValue(ctx, consts.ContextClientAddr),
		}}, nil
	})

	request := httptest.NewRequest(http.MethodGet, "/context", nil)
	request.Header.Set(consts.HeaderRequestID, "request-123")
	request.Header.Set(consts.HeaderUserID, "user-456")
	request.Header.Set(consts.HeaderAuthorization, "token-789")
	request.Header.Set("X-Custom", "custom-value")
	request.Header.Set("X-Computed", "computed-value")
	request.RemoteAddr = "192.0.2.1:1234"

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response humaContextResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	for key, want := range map[string]string{
		"custom":    "custom-value",
		"computed":  "computed-value",
		"trace_id":  "request-123",
		"user_id":   "user-456",
		"token":     "token-789",
		"client_ip": "192.0.2.1",
	} {
		var got string
		switch key {
		case "custom":
			got = response.Custom
		case "computed":
			got = response.Computed
		case "trace_id":
			got = response.TraceID
		case "user_id":
			got = response.UserID
		case "token":
			got = response.Token
		case "client_ip":
			got = response.ClientIP
		}
		if got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestHumaRecoveryLog_ReturnsInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, huma.DefaultConfig("test", "v1"))
	api.UseMiddleware(HumaRecoveryLog(false))

	huma.Get(api, "/panic", func(context.Context, *struct{}) (*struct{}, error) {
		panic("boom")
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestHumaHandlerError_IsReturnedByHuma(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, huma.DefaultConfig("test", "v1"))
	api.UseMiddleware(SetHumaContext(), HumaRequestMiddleware())

	huma.Get(api, "/error", func(context.Context, *struct{}) (*struct{}, error) {
		return nil, huma.Error400BadRequest("invalid request")
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/error", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestHumaRequestMiddleware_SetsRequestIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := resource.Logger
	resource.Logger = nil
	t.Cleanup(func() { resource.Logger = previousLogger })

	router := gin.New()
	api := humagin.New(router, huma.DefaultConfig("test", "v1"))
	api.UseMiddleware(SetHumaContext(), HumaRequestMiddleware())

	huma.Get(api, "/request-log", func(context.Context, *struct{}) (*humaRequestLogOutput, error) {
		return &humaRequestLogOutput{Body: struct {
			OK bool `json:"ok"`
		}{OK: true}}, nil
	})

	request := httptest.NewRequest(http.MethodGet, "/request-log", nil)
	request.Header.Set(consts.HeaderRequestID, "trace-123")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(consts.HeaderRequestID); got != "trace-123" {
		t.Fatalf("%s = %q, want %q", consts.HeaderRequestID, got, "trace-123")
	}
}

func TestHumaRequestMiddleware_SkipsHiddenOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, huma.DefaultConfig("test", "v1"))
	api.UseMiddleware(HumaRequestMiddleware(WithHumaSkipHiddenOperations()))
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		if _, ok := humagin.Unwrap(ctx).Writer.(*humaBodyLog); ok {
			t.Error("hidden operation should not replace the Gin response writer")
		}
		next(ctx)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "hidden",
		Method:        http.MethodGet,
		Path:          "/hidden",
		Hidden:        true,
		DefaultStatus: http.StatusNoContent,
	}, func(context.Context, *struct{}) (*struct{}, error) {
		return nil, nil
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/hidden", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func stringValue(ctx context.Context, key string) string {
	value, _ := ctx.Value(key).(string)
	return value
}
