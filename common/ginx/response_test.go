package ginx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestResponseOK_WritesCommonSuccessContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	ResponseOK(ctx, "v1", "success")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var payload struct {
		Code        int      `json:"code"`
		Version     string   `json:"version"`
		CurrentTime string   `json:"current_time"`
		Data        []string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != 0 || payload.Version != "v1" {
		t.Fatalf("payload = %#v, want success envelope for v1", payload)
	}
	if len(payload.Data) != 1 || payload.Data[0] != "success" {
		t.Fatalf("data = %#v, want [success]", payload.Data)
	}
	if _, err := time.Parse(time.DateTime, payload.CurrentTime); err != nil {
		t.Fatalf("current_time = %q, want Go DateTime format: %v", payload.CurrentTime, err)
	}
}
