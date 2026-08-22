package humax_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/generate-example-project/common/humax"
)

func TestSuccess_ReturnsTypedEnvelope(t *testing.T) {
	output := humax.Success("v1", []string{"success"})

	if output.Body == nil {
		t.Fatal("Body is nil")
	}
	if len(output.Body.Data) != 1 || output.Body.Data[0] != "success" {
		t.Fatalf("Body.Data = %#v, want [success]", output.Body.Data)
	}
}

func TestInternalServerError_UsesLegacyEnvelope(t *testing.T) {
	output := humax.InternalServerError("v1", errors.New("probe unavailable"))

	if output.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("GetStatus() = %d, want %d", output.GetStatus(), http.StatusInternalServerError)
	}
	if output.Error() != "probe unavailable" {
		t.Fatalf("Error() = %q, want probe unavailable", output.Error())
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal Huma error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal Huma error: %v", err)
	}
	if payload["message"] != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %#v, want %q", payload["message"], http.StatusText(http.StatusInternalServerError))
	}
	if _, ok := payload["data"]; !ok {
		t.Fatal("data is missing from Huma error")
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the internal error")
	}
}

func TestWrap_MapsUnexpectedErrorToSafeEnvelope(t *testing.T) {
	_, api := humatest.New(t)
	humav2.Get(api, "/failure", humax.Wrap("v1", func(context.Context, *struct{}) (*string, error) {
		return nil, errors.New("database password is invalid")
	}))

	response := api.Get("/failure")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["message"] != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %#v, want %q", payload["message"], http.StatusText(http.StatusInternalServerError))
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the internal error")
	}
}

func TestWrap_PreservesSharedBusinessError(t *testing.T) {
	_, api := humatest.New(t)
	humav2.Get(api, "/missing", humax.Wrap("v1", func(context.Context, *struct{}) (*string, error) {
		return nil, humax.BusinessError("v1", 1001, "resource not found")
	}))

	response := api.Get("/missing")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["code"] != float64(1001) || payload["message"] != "resource not found" {
		t.Fatalf("payload = %#v, want shared not-found envelope", payload)
	}
}

func TestConfigureHumaErrorFactory_UsesEnvelopeForInvalidRequest(t *testing.T) {
	originalFactory := humav2.NewErrorWithContext
	originalError := humav2.NewError
	t.Cleanup(func() {
		humav2.NewErrorWithContext = originalFactory
		humav2.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	router := gin.New()
	config := humav2.DefaultConfig("test", "v1")
	config.DocsPath = ""
	config.OpenAPIPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humagin.New(router, config)
	humav2.Post(api, "/validation", func(context.Context, *struct {
		Body struct {
			Name string `json:"name" minLength:"1"`
		}
	}) (*struct{}, error) {
		return &struct{}{}, nil
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/validation", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["code"] != float64(1) {
		t.Fatalf("code = %#v, want 1", payload["code"])
	}
	if payload["message"] == "" {
		t.Fatal("message is empty")
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not be present for validation errors")
	}
}

func TestConfigureHumaErrorFactory_PreservesInternalServerError(t *testing.T) {
	originalFactory := humav2.NewErrorWithContext
	originalError := humav2.NewError
	t.Cleanup(func() {
		humav2.NewErrorWithContext = originalFactory
		humav2.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	errorResponse := humav2.NewErrorWithContext(nil, http.StatusInternalServerError, "database password is invalid")
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
	if payload["message"] != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %#v, want %q", payload["message"], http.StatusText(http.StatusInternalServerError))
	}
	if _, ok := payload["err_trace"]; ok {
		t.Fatal("err_trace must not expose the internal error")
	}
}

func TestConfigureHumaErrorFactory_UsesEnvelopeForOpenAPIErrorSchema(t *testing.T) {
	originalFactory := humav2.NewErrorWithContext
	originalError := humav2.NewError
	t.Cleanup(func() {
		humav2.NewErrorWithContext = originalFactory
		humav2.NewError = originalError
	})
	humax.ConfigureHumaErrorFactory("v1")

	errorResponse := humav2.NewError(0, "")
	encoded, err := json.Marshal(errorResponse)
	if err != nil {
		t.Fatalf("marshal Huma error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal Huma error: %v", err)
	}
	if payload["code"] != float64(1) || payload["version"] != "v1" {
		t.Fatalf("payload = %#v, want shared error envelope", payload)
	}
}

func TestPaginationSuccessAndOffset(t *testing.T) {
	pagination := &humax.Pagination{Page: 2, PageSize: 20, Total: 41}
	pagination.GetPageCount()

	if pagination.PageCount != 3 {
		t.Fatalf("PageCount = %d, want 3", pagination.PageCount)
	}
	if pagination.GetOffset() != 20 {
		t.Fatalf("offset = %d, want 20", pagination.GetOffset())
	}

	output := humax.PaginationSuccess("v1", []string{"success"}, pagination)
	if output.Body.Pagination != pagination {
		t.Fatal("PaginationSuccess did not preserve pagination metadata")
	}
}

func TestFileStreamsContentWithHeaders(t *testing.T) {
	_, api := humatest.New(t)
	humav2.Get(api, "/download", func(context.Context, *struct{}) (*humav2.StreamResponse, error) {
		return humax.File("v1", &humax.FileDownloadConfig{
			Filename:    "report.txt",
			ContentType: "text/plain",
			Content:     []byte("hello"),
		})
	})

	response := api.Get("/download")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	if got := response.Header().Get("Content-Disposition"); got != `attachment; filename=report.txt` {
		t.Fatalf("Content-Disposition = %q, want attachment; filename=report.txt", got)
	}
}

func TestSimpleFileStreamsAndDeletesPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(filePath, []byte("from disk"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, api := humatest.New(t)
	humav2.Get(api, "/download", func(context.Context, *struct{}) (*humav2.StreamResponse, error) {
		return humax.File("v1", &humax.FileDownloadConfig{
			Filepath:    filePath,
			DeleteAfter: true,
		})
	})

	response := api.Get("/download")
	if response.Code != http.StatusOK || response.Body.String() != "from disk" {
		t.Fatalf("response = (%d, %q), want (200, from disk)", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still exists or stat failed: %v", err)
	}
}

func TestFileStreamsReader(t *testing.T) {
	_, api := humatest.New(t)
	humav2.Get(api, "/download", func(context.Context, *struct{}) (*humav2.StreamResponse, error) {
		return humax.File("v1", &humax.FileDownloadConfig{Reader: strings.NewReader("from reader")})
	})

	response := api.Get("/download")
	if response.Code != http.StatusOK || response.Body.String() != "from reader" {
		t.Fatalf("response = (%d, %q), want (200, from reader)", response.Code, response.Body.String())
	}
}
