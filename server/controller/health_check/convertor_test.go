package health_check

import (
	"testing"

	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

func TestToHealthCheckData(t *testing.T) {
	data := toHealthCheckData(service.Result{Status: "ready"})

	if len(data) != 1 || data[0] != "ready" {
		t.Fatalf("data = %#v, want [ready]", data)
	}
}
