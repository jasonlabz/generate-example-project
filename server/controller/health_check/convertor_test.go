package health_check

import (
	"testing"

	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

func TestToHealthCheckOutput(t *testing.T) {
	output := toHealthCheckOutput(service.HealthCheckResult{Status: "ready"})

	if output == nil || output.Body == nil {
		t.Fatal("toHealthCheckOutput() returned nil output")
	}
	if output.Body.Version != "v1" {
		t.Fatalf("version = %q, want %q", output.Body.Version, "v1")
	}
	if len(output.Body.Data) != 1 || output.Body.Data[0] != "ready" {
		t.Fatalf("data = %#v, want [ready]", output.Body.Data)
	}
}
