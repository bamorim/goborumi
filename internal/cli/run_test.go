package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	got, err := parseFormat("json")
	if err != nil {
		t.Fatalf("parseFormat(json) returned error: %v", err)
	}
	if got != formatJSON {
		t.Fatalf("expected %q, got %q", formatJSON, got)
	}

	_, err = parseFormat("xml")
	if err == nil {
		t.Fatal("expected parseFormat(xml) to fail")
	}
}

func TestReadScriptInput(t *testing.T) {
	t.Parallel()

	got, err := readScriptInput("hello", "")
	if err != nil {
		t.Fatalf("readScriptInput(script) returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("unexpected script: %q", got)
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "scene.txt")
	if err := os.WriteFile(scriptPath, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("could not write temp script file: %v", err)
	}

	got, err = readScriptInput("", scriptPath)
	if err != nil {
		t.Fatalf("readScriptInput(script-file) returned error: %v", err)
	}
	if got != "from-file" {
		t.Fatalf("unexpected script file content: %q", got)
	}

	_, err = readScriptInput("", "")
	if err == nil {
		t.Fatal("expected missing script input to fail")
	}

	_, err = readScriptInput("inline", scriptPath)
	if err == nil {
		t.Fatal("expected both script inputs to fail")
	}
}

