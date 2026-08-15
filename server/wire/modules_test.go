package wire

import "testing"

func TestModules_ExposeHealthCheckModule(t *testing.T) {
	modules := Modules()
	if len(modules) != 1 {
		t.Fatalf("Modules() length = %d, want 1", len(modules))
	}
	if modules[0].Name != "health-check" {
		t.Fatalf("Modules()[0].Name = %q, want health-check", modules[0].Name)
	}
	if modules[0].RegisterRoot == nil {
		t.Fatal("health-check module has no root registration")
	}
}
