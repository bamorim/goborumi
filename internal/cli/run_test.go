package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bamorim/goborumi/internal/borumi"
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

func TestRenderScriptForDisplay(t *testing.T) {
	t.Parallel()

	rawJSON := `{"selection":null,"content":[{"type":"paragraph","content":[{"type":"text","text":"Body"}]}]}`
	got := renderScriptForDisplay(rawJSON)
	want := "Body"

	if got != want {
		t.Fatalf("unexpected rendered script:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	rawText := "not-json-script"
	if got := renderScriptForDisplay(rawText); got != rawText {
		t.Fatalf("expected raw script fallback, got %q", got)
	}
}

func TestPrepareScriptForStorage(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{
		"## Scene 1",
		"",
		"Hello world",
	}, "\n")

	rawJSON, err := prepareScriptForStorage(markdown)
	if err != nil {
		t.Fatalf("prepareScriptForStorage(markdown) returned error: %v", err)
	}
	if !strings.Contains(rawJSON, `"content"`) {
		t.Fatalf("expected script json output, got %q", rawJSON)
	}

	_, err = prepareScriptForStorage(`{"type":"doc"`)
	if err == nil {
		t.Fatal("expected malformed JSON input to fail")
	}
}

func TestRenderSceneGetOutput(t *testing.T) {
	t.Parallel()

	scene := borumi.Scene{
		Index:  1,
		ID:     "scene-1",
		Seq:    "a",
		Name:   "Intro",
		Script: `{"selection":null,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
	}

	got := renderSceneGetOutput(scene, true)

	if strings.Contains(got, "script |") {
		t.Fatalf("script should not be rendered inside table: %q", got)
	}
	if !strings.Contains(got, "\n\nHello") {
		t.Fatalf("expected blank line between table and script, got: %q", got)
	}
}

func TestResolveRenderScript(t *testing.T) {
	t.Parallel()

	if got := resolveRenderScript(false, false, formatTable); !got {
		t.Fatalf("expected table default to render script")
	}
	if got := resolveRenderScript(false, false, formatJSON); got {
		t.Fatalf("expected json default to not render script")
	}
	if got := resolveRenderScript(false, true, formatTable); got {
		t.Fatalf("expected explicit false to disable rendering")
	}
	if got := resolveRenderScript(true, true, formatJSON); !got {
		t.Fatalf("expected explicit true to enable rendering")
	}
}

func TestSceneForJSONOutputRawScript(t *testing.T) {
	t.Parallel()

	scene := borumi.Scene{
		Index:  1,
		ID:     "scene-1",
		Seq:    "a",
		Name:   "Intro",
		Script: `{"selection":null,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
	}

	out, err := sceneForJSONOutput(scene, false)
	if err != nil {
		t.Fatalf("sceneForJSONOutput returned error: %v", err)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if !strings.Contains(string(raw), `"script":{"selection":null`) {
		t.Fatalf("expected unwrapped script json, got %s", string(raw))
	}
}

func TestSceneForJSONOutputRenderedScript(t *testing.T) {
	t.Parallel()

	scene := borumi.Scene{
		Index:  1,
		ID:     "scene-1",
		Seq:    "a",
		Name:   "Intro",
		Script: `{"selection":null,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
	}

	out, err := sceneForJSONOutput(scene, true)
	if err != nil {
		t.Fatalf("sceneForJSONOutput returned error: %v", err)
	}

	text, ok := out.Script.(string)
	if !ok {
		t.Fatalf("expected rendered script string, got %T", out.Script)
	}
	if text != "Hello" {
		t.Fatalf("unexpected rendered script: %q", text)
	}
}

func TestScenesForJSONOutput(t *testing.T) {
	t.Parallel()

	scenes := []borumi.Scene{
		{
			Index:  1,
			ID:     "scene-1",
			Seq:    "a",
			Name:   "Intro",
			Script: `{"selection":null,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
		},
	}

	out, err := scenesForJSONOutput(scenes, false)
	if err != nil {
		t.Fatalf("scenesForJSONOutput returned error: %v", err)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if !strings.Contains(string(raw), `"script":{"selection":null`) {
		t.Fatalf("expected unwrapped json script, got %s", string(raw))
	}
}

func TestNormalizeBoolFlagArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "space separated true",
			args: []string{"--render-script", "true", "--format", "json"},
			want: []string{"--render-script=true", "--format", "json"},
		},
		{
			name: "space separated false",
			args: []string{"--render-script", "false"},
			want: []string{"--render-script=false"},
		},
		{
			name: "already equals form",
			args: []string{"--render-script=true"},
			want: []string{"--render-script=true"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeBoolFlagArgs(tt.args, "render-script")
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("unexpected normalized args: got=%v want=%v", got, tt.want)
			}
		})
	}
}
