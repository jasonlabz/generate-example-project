package response_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jasonlabz/generate-example-project/common/response"
)

func TestNew_SetsSuccessEnvelope(t *testing.T) {
	envelope := response.New("v1", []string{"success"})

	if envelope.Code != 0 {
		t.Fatalf("Code = %d, want 0", envelope.Code)
	}
	if envelope.Version != "v1" {
		t.Fatalf("Version = %q, want v1", envelope.Version)
	}
	if len(envelope.Data) != 1 || envelope.Data[0] != "success" {
		t.Fatalf("Data = %#v, want [success]", envelope.Data)
	}
	if _, err := time.Parse(time.DateTime, envelope.CurrentTime); err != nil {
		t.Fatalf("CurrentTime = %q, want Go DateTime format: %v", envelope.CurrentTime, err)
	}
}

func TestNewError_PreservesErrorPayload(t *testing.T) {
	envelope := response.NewError("v1", []any{}, 42, "unavailable", "probe unavailable")

	if envelope.Code != 42 {
		t.Fatalf("Code = %d, want 42", envelope.Code)
	}
	if envelope.Message != "unavailable" {
		t.Fatalf("Message = %q, want unavailable", envelope.Message)
	}
	if envelope.ErrTrace != "probe unavailable" {
		t.Fatalf("ErrTrace = %q, want probe unavailable", envelope.ErrTrace)
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatal("data is missing from error envelope")
	}
}
