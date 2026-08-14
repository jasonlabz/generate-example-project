package humax_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

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
