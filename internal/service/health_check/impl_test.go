package health_check

import (
	"context"
	"testing"
)

func TestDoCheck(t *testing.T) {
	if got := GetService().DoCheck(context.Background()); got != "success" {
		t.Fatalf("expect success, got %s", got)
	}
}
