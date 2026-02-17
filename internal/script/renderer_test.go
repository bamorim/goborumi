package script

import (
	"strings"
	"testing"
)

func TestJSONToMarkdown(t *testing.T) {
	t.Parallel()

	raw := `{
  "type":"doc",
  "blocks":[
    {"type":"heading","level":2,"text":"Intro"},
    {"type":"paragraph","text":"First line\nSecond line"},
    {"type":"bullet_list","items":["one","two"]},
    {"type":"ordered_list","items":["alpha","beta"]},
    {"type":"quote","text":"quoted\ntext"},
    {"type":"code","language":"go","text":"fmt.Println(\"hi\")"}
  ]
}`

	got, err := JSONToMarkdown(raw)
	if err != nil {
		t.Fatalf("JSONToMarkdown returned error: %v", err)
	}

	want := strings.Join([]string{
		"## Intro",
		"",
		"First line",
		"Second line",
		"",
		"- one",
		"- two",
		"",
		"1. alpha",
		"2. beta",
		"",
		"> quoted",
		"> text",
		"",
		"```go",
		`fmt.Println("hi")`,
		"```",
	}, "\n")

	if got != want {
		t.Fatalf("unexpected markdown output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONToMarkdownBorumiDocument(t *testing.T) {
	t.Parallel()

	raw := `{
  "selection": null,
  "content": [
    {
      "type": "bullet-list",
      "content": [
        {
          "content": [
            {
              "type": "paragraph",
              "content": [
                {"type": "text", "text": "first bullet"}
              ]
            }
          ]
        },
        {
          "content": [
            {
              "type": "paragraph",
              "content": [
                {"type": "text", "text": "second bullet"}
              ]
            }
          ]
        }
      ]
    }
  ]
}`

	got, err := JSONToMarkdown(raw)
	if err != nil {
		t.Fatalf("JSONToMarkdown returned error: %v", err)
	}

	want := strings.Join([]string{
		"- first bullet",
		"- second bullet",
	}, "\n")

	if got != want {
		t.Fatalf("unexpected markdown output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestMarkdownToJSONRoundTrip(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{
		"# Scene",
		"",
		"Hello team",
		"",
		"- item one",
		"- item two",
		"",
		"> quote line",
		"",
		"```txt",
		"code line",
		"```",
	}, "\n")

	raw, err := MarkdownToJSON(markdown)
	if err != nil {
		t.Fatalf("MarkdownToJSON returned error: %v", err)
	}

	got, err := JSONToMarkdown(raw)
	if err != nil {
		t.Fatalf("JSONToMarkdown returned error: %v", err)
	}

	if got != markdown {
		t.Fatalf("markdown did not round-trip:\n--- got ---\n%s\n--- want ---\n%s", got, markdown)
	}
}

func TestNormalizeOrConvertToJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "json",
			input: `{"type":"doc","blocks":[{"type":"paragraph","text":"hello"}]}`,
		},
		{
			name: "markdown",
			input: strings.Join([]string{
				"# Intro",
				"",
				"body",
			}, "\n"),
		},
		{
			name:    "invalid json starts with brace",
			input:   `{"type":"doc"`,
			wantErr: true,
		},
		{
			name:  "empty markdown",
			input: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw, err := NormalizeOrConvertToJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (raw=%q)", raw)
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeOrConvertToJSON returned error: %v", err)
			}

			if _, err := ParseJSON(raw); err != nil {
				t.Fatalf("normalized output is invalid script json: %v (raw=%q)", err, raw)
			}
		})
	}
}

func TestParseJSONRejectsUnknownObjectShape(t *testing.T) {
	t.Parallel()

	_, err := ParseJSON(`{"selection":null}`)
	if err == nil {
		t.Fatal("expected ParseJSON to reject unknown object shape")
	}
}

func TestParseMarkdownUnclosedCodeFence(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("```go\nfmt.Println(\"x\")")
	if err == nil {
		t.Fatal("expected ParseMarkdown to fail on unclosed code fence")
	}
}
