package script

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGoMockgen_GeneratesMirroredMocksAndSkipsOutput(t *testing.T) {
	projectRoot := projectRoot(t)
	outputRoot := filepath.Join(projectRoot, "server", ".test-mocks")
	fixtureRoot := filepath.Join(projectRoot, "cmd", "mockgenfixture")
	fixtureSource := filepath.Join(fixtureRoot, "internal", "sample", "contract.go")
	internalMock := filepath.Join(fixtureRoot, ".mocks", "internal", "sample", "mock_contract.go")

	removeTestPath(t, outputRoot)
	removeTestPath(t, fixtureRoot)
	t.Cleanup(func() {
		removeTestPath(t, outputRoot)
		removeTestPath(t, fixtureRoot)
		assertFileDoesNotExist(t, outputRoot)
		assertFileDoesNotExist(t, fixtureRoot)
	})

	if err := os.MkdirAll(filepath.Dir(fixtureSource), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(fixtureSource, []byte("package sample\n\nimport \"context\"\n\ntype Contract interface {\n\tRun(context.Context) error\n}\n"), 0o600); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}

	bashPath := gitBashPath(t)
	runMockgen(t, bashPath, projectRoot, outputRoot)
	assertFileExists(t, filepath.Join(outputRoot, "server", "controller", "health_check", "mock_register.go"))
	assertFileExists(t, internalMock)
	assertFileDoesNotExist(t, filepath.Join(outputRoot, "bootstrap", "mock_migrate_dialect.go"))

	poisonSource := filepath.Join(outputRoot, "poison.go")
	if err := os.WriteFile(poisonSource, []byte("package mocks\n\ntype MustNotGenerate interface {\n\tStop()\n}\n"), 0o600); err != nil {
		t.Fatalf("write output poison source: %v", err)
	}

	runMockgen(t, bashPath, projectRoot, outputRoot)
	assertFileDoesNotExist(t, filepath.Join(outputRoot, "server", ".test-mocks", "mock_poison.go"))
}

func projectRoot(t *testing.T) string {
	t.Helper()

	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Dir(directory)
}

func gitBashPath(t *testing.T) string {
	t.Helper()

	if runtime.GOOS != "windows" {
		path, err := exec.LookPath("bash")
		if err != nil {
			t.Fatalf("find bash: %v", err)
		}
		return path
	}

	for _, path := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "bin", "bash.exe"),
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Fatal("Git Bash is required to run script/go-mockgen.sh on Windows")
	return ""
}

func runMockgen(t *testing.T, bashPath, projectRoot, outputRoot string) {
	t.Helper()

	command := exec.Command(bashPath, "script/go-mockgen.sh", "-o", outputRoot)
	command.Dir = projectRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run go-mockgen.sh: %v\n%s", err, output)
	}
}

func removeTestPath(t *testing.T, path string) {
	t.Helper()

	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("remove test path %q: %v", path, err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected generated file %q: %v", path, err)
	}
}

func assertFileDoesNotExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected generated file %q: %v", path, err)
	}
}
