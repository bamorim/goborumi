package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bamorim/goborumi/internal/borumi"
	"github.com/bamorim/goborumi/internal/render"
	"github.com/bamorim/goborumi/internal/script"
)

const version = "0.1.0-bootstrap"

type outputFormat string

const (
	formatTable outputFormat = "table"
	formatJSON  outputFormat = "json"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "goborumi %s\n", version)
		return 0
	case "scenes":
		return runScenes(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "goborumi")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "CLI utilities for Borumi projects.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  goborumi <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  scenes list        List scenes for a project")
	fmt.Fprintln(w, "  scenes get         Read one scene script/details")
	fmt.Fprintln(w, "  scenes set-script  Update one scene script")
	fmt.Fprintln(w, "  version            Print version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global conventions:")
	fmt.Fprintln(w, "  - Use --bundle with a .bmprojbundle path, a project directory, or project.bmproj")
	fmt.Fprintln(w, "  - Use --format table|json (default: table)")
}

func runScenes(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printScenesHelp(stderr)
		return 1
	}

	switch args[0] {
	case "list":
		return runScenesList(args[1:], stdout, stderr)
	case "get", "show":
		return runScenesGet(args[1:], stdout, stderr)
	case "set-script":
		return runScenesSetScript(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown scenes command: %s\n\n", args[0])
		printScenesHelp(stderr)
		return 1
	}
}

func printScenesHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  goborumi scenes list [--bundle <path>] [--format table|json] [--render-script <bool>]")
	fmt.Fprintln(w, "  goborumi scenes get (--scene <id-or-name-or-seq> | --index <n>) [--bundle <path>] [--format table|json] [--render-script <bool>]")
	fmt.Fprintln(w, "  goborumi scenes set-script (--scene <id-or-name-or-seq> | --index <n>) (--script <markdown-or-json> | --script-file <path>) [--bundle <path>] [--format table|json] [--render-script <bool>]")
}

func runScenesList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("scenes list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	bundlePath := fs.String("bundle", ".", "Bundle path")
	formatArg := fs.String("format", string(formatTable), "Output format: table|json")
	renderScriptArg := fs.Bool("render-script", false, "Render script as markdown")
	args = normalizeBoolFlagArgs(args, "render-script")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		printScenesHelp(stderr)
		return 1
	}

	format, err := parseFormat(*formatArg)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	renderScript := resolveRenderScript(*renderScriptArg, wasFlagProvided(fs, "render-script"), format)

	bundle, err := borumi.ResolveBundle(*bundlePath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	scenes, err := borumi.ListScenes(bundle.DBPath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	switch format {
	case formatJSON:
		out, err := scenesForJSONOutput(scenes, renderScript)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return writeJSON(stdout, stderr, out)
	default:
		rows := make([][]string, 0, len(scenes))
		for _, scene := range scenes {
			rows = append(rows, []string{
				fmt.Sprintf("%d", scene.Index),
				scene.ID,
				scene.Name,
				fmt.Sprintf("%d", len(scene.Script)),
			})
		}
		fmt.Fprintln(stdout, render.Table(
			[]string{"INDEX", "ID", "NAME", "SCRIPT_LEN"},
			rows,
		))
		return 0
	}
}

func runScenesGet(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("scenes get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	bundlePath := fs.String("bundle", ".", "Bundle path")
	sceneSelector := fs.String("scene", "", "Scene id, name, or seq")
	sceneIndex := fs.Int("index", 0, "Scene index from scenes list (1-based)")
	formatArg := fs.String("format", string(formatTable), "Output format: table|json")
	renderScriptArg := fs.Bool("render-script", false, "Render script as markdown")
	args = normalizeBoolFlagArgs(args, "render-script")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		printScenesHelp(stderr)
		return 1
	}

	format, err := parseFormat(*formatArg)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	renderScript := resolveRenderScript(*renderScriptArg, wasFlagProvided(fs, "render-script"), format)

	bundle, err := borumi.ResolveBundle(*bundlePath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	scene, err := resolveScene(bundle.DBPath, *sceneSelector, *sceneIndex)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if format == formatJSON {
		jsonScene, err := sceneForJSONOutput(scene, renderScript)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return writeJSON(stdout, stderr, jsonScene)
	}

	fmt.Fprintln(stdout, renderSceneGetOutput(scene, renderScript))
	return 0
}

func runScenesSetScript(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("scenes set-script", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	bundlePath := fs.String("bundle", ".", "Bundle path")
	sceneSelector := fs.String("scene", "", "Scene id, name, or seq")
	sceneIndex := fs.Int("index", 0, "Scene index from scenes list (1-based)")
	script := fs.String("script", "", "New script text")
	scriptFile := fs.String("script-file", "", "Path to file containing new script text")
	formatArg := fs.String("format", string(formatTable), "Output format: table|json")
	renderScriptArg := fs.Bool("render-script", false, "Render script as markdown")
	args = normalizeBoolFlagArgs(args, "render-script")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		printScenesHelp(stderr)
		return 1
	}

	newScript, err := readScriptInput(*script, *scriptFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	scriptJSON, err := prepareScriptForStorage(newScript)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	format, err := parseFormat(*formatArg)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	renderScript := resolveRenderScript(*renderScriptArg, wasFlagProvided(fs, "render-script"), format)

	bundle, err := borumi.ResolveBundle(*bundlePath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	targetScene, err := resolveScene(bundle.DBPath, *sceneSelector, *sceneIndex)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	updated, err := borumi.SetSceneScript(bundle.DBPath, targetScene.ID, scriptJSON)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if format == formatJSON {
		jsonScene, err := sceneForJSONOutput(updated, renderScript)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return writeJSON(stdout, stderr, jsonScene)
	}

	displayScript := renderScriptForDisplay(updated.Script)
	if !renderScript {
		displayScript = updated.Script
	}
	fmt.Fprintln(stdout, render.Properties([][2]string{
		{"status", "updated"},
		{"id", updated.ID},
		{"index", fmt.Sprintf("%d", updated.Index)},
		{"seq", updated.Seq},
		{"name", updated.Name},
		{"script", displayScript},
	}))
	return 0
}

func parseFormat(value string) (outputFormat, error) {
	switch outputFormat(strings.ToLower(strings.TrimSpace(value))) {
	case formatTable:
		return formatTable, nil
	case formatJSON:
		return formatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format value %q (expected table|json)", value)
	}
}

func readScriptInput(script string, scriptFile string) (string, error) {
	scriptFile = strings.TrimSpace(scriptFile)

	if strings.TrimSpace(script) == "" && scriptFile == "" {
		return "", errors.New("one of --script or --script-file is required")
	}

	if strings.TrimSpace(script) != "" && scriptFile != "" {
		return "", errors.New("use only one of --script or --script-file")
	}

	if strings.TrimSpace(script) != "" {
		return script, nil
	}

	content, err := os.ReadFile(scriptFile)
	if err != nil {
		return "", fmt.Errorf("could not read --script-file %q: %w", scriptFile, err)
	}
	return string(content), nil
}

func resolveScene(dbPath string, sceneSelector string, sceneIndex int) (borumi.Scene, error) {
	sceneSelector = strings.TrimSpace(sceneSelector)

	if sceneSelector == "" && sceneIndex == 0 {
		return borumi.Scene{}, errors.New("one of --scene or --index is required")
	}
	if sceneSelector != "" && sceneIndex != 0 {
		return borumi.Scene{}, errors.New("use only one of --scene or --index")
	}

	if sceneIndex > 0 {
		return borumi.GetSceneByIndex(dbPath, sceneIndex)
	}

	return borumi.GetScene(dbPath, sceneSelector)
}

func writeJSON(stdout io.Writer, stderr io.Writer, value any) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintf(stderr, "failed to write json output: %v\n", err)
		return 1
	}
	return 0
}

func renderScriptForDisplay(rawScript string) string {
	markdown, err := script.JSONToMarkdown(rawScript)
	if err != nil {
		return rawScript
	}
	return markdown
}

func prepareScriptForStorage(input string) (string, error) {
	raw, err := script.NormalizeOrConvertToJSON(input)
	if err != nil {
		return "", fmt.Errorf("could not convert script input: %w", err)
	}
	return raw, nil
}

func renderSceneGetOutput(scene borumi.Scene, renderScript bool) string {
	table := render.Properties([][2]string{
		{"id", scene.ID},
		{"index", fmt.Sprintf("%d", scene.Index)},
		{"seq", scene.Seq},
		{"name", scene.Name},
	})

	displayScript := scene.Script
	if renderScript {
		displayScript = renderScriptForDisplay(scene.Script)
	}
	if strings.TrimSpace(displayScript) == "" {
		return table
	}

	return table + "\n\n" + displayScript
}

func wasFlagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}

func resolveRenderScript(value bool, provided bool, format outputFormat) bool {
	if provided {
		return value
	}
	return format == formatTable
}

type sceneJSONOutput struct {
	Index  int    `json:"index,omitempty"`
	ID     string `json:"id"`
	Seq    string `json:"seq"`
	Name   string `json:"name"`
	Script any    `json:"script"`
}

func sceneForJSONOutput(scene borumi.Scene, renderScript bool) (sceneJSONOutput, error) {
	out := sceneJSONOutput{
		Index: scene.Index,
		ID:    scene.ID,
		Seq:   scene.Seq,
		Name:  scene.Name,
	}

	if renderScript {
		out.Script = renderScriptForDisplay(scene.Script)
		return out, nil
	}

	scriptJSON, err := scriptRawJSON(scene.Script)
	if err != nil {
		return sceneJSONOutput{}, err
	}
	out.Script = scriptJSON
	return out, nil
}

func scenesForJSONOutput(scenes []borumi.Scene, renderScript bool) ([]sceneJSONOutput, error) {
	out := make([]sceneJSONOutput, 0, len(scenes))
	for _, scene := range scenes {
		converted, err := sceneForJSONOutput(scene, renderScript)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func scriptRawJSON(rawScript string) (any, error) {
	trimmed := strings.TrimSpace(rawScript)
	if trimmed == "" {
		return nil, nil
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, fmt.Errorf("script is not valid json")
	}
	return json.RawMessage(trimmed), nil
}

func normalizeBoolFlagArgs(args []string, flagName string) []string {
	long := "--" + flagName
	short := "-" + flagName

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		current := args[i]
		if (current == long || current == short) && i+1 < len(args) {
			next := strings.ToLower(strings.TrimSpace(args[i+1]))
			if next == "true" || next == "false" {
				out = append(out, current+"="+next)
				i++
				continue
			}
		}
		out = append(out, current)
	}
	return out
}
