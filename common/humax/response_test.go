package humax_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
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
	if payload["message"] != "probe unavailable" {
		t.Fatalf("message = %#v, want probe unavailable", payload["message"])
	}
	if _, ok := payload["data"]; !ok {
		t.Fatal("data is missing from Huma error")
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
