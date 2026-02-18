package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bamorim/goborumi/internal/borumi"
	"github.com/bamorim/goborumi/internal/script"
	urfavecli "github.com/urfave/cli/v3"
)

const version = "0.1.0-bootstrap"

type outputFormat string

const (
	formatTable outputFormat = "table"
	formatJSON  outputFormat = "json"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	cmd := NewRootCommand(stdout, stderr)
	if err := cmd.Run(context.Background(), append([]string{cmd.Name}, args...)); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func NewRootCommand(stdout io.Writer, stderr io.Writer) *urfavecli.Command {
	root := &urfavecli.Command{
		Name:                   "goborumi",
		Usage:                  "CLI utilities for Borumi projects.",
		Version:                version,
		Writer:                 stdout,
		ErrWriter:              stderr,
		UseShortOptionHandling: true,
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			if cmd.Args().Present() {
				return fmt.Errorf("No help topic for '%s'", cmd.Args().First())
			}
			return urfavecli.ShowRootCommandHelp(cmd)
		},
		Commands: []*urfavecli.Command{
			newScenesCommand(),
			{
				Name:  "version",
				Usage: "Print version",
				Action: func(_ context.Context, cmd *urfavecli.Command) error {
					fmt.Fprintf(commandWriter(cmd), "%s %s\n", cmd.Root().Name, cmd.Root().Version)
					return nil
				},
			},
		},
	}
	setCommandWriters(root, stdout, stderr)
	return root
}

func newScenesCommand() *urfavecli.Command {
	return &urfavecli.Command{
		Name:    "scenes",
		Usage:   "Scene script operations",
		Aliases: []string{"scene"},
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			if cmd.Args().Present() {
				return fmt.Errorf("No help topic for '%s %s'", cmd.FullName(), cmd.Args().First())
			}
			return urfavecli.ShowSubcommandHelp(cmd)
		},
		Commands: []*urfavecli.Command{
			newScenesListCommand(),
			newScenesGetCommand(),
			newScenesSetScriptCommand(),
		},
	}
}

func newScenesListCommand() *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "list",
		Usage: "List scenes for a project",
		Flags: sceneOutputFlags(),
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			format, err := parseFormat(cmd.String("format"))
			if err != nil {
				return err
			}
			renderScript := resolveRenderScript(cmd.Bool("render-script"), cmd.IsSet("render-script"), format)

			bundle, err := borumi.ResolveBundle(cmd.String("bundle"))
			if err != nil {
				return err
			}

			scenes, err := borumi.ListScenes(bundle.DBPath)
			if err != nil {
				return err
			}

			switch format {
			case formatJSON:
				out, err := scenesForJSONOutput(scenes, renderScript)
				if err != nil {
					return err
				}
				return writeJSON(commandWriter(cmd), out)
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
				fmt.Fprintln(commandWriter(cmd), Table(
					[]string{"INDEX", "ID", "NAME", "SCRIPT_LEN"},
					rows,
				))
				return nil
			}
		},
	}
}

func newScenesGetCommand() *urfavecli.Command {
	return &urfavecli.Command{
		Name:    "get",
		Usage:   "Read one scene script/details",
		Aliases: []string{"show"},
		Flags: append(sceneOutputFlags(),
			&urfavecli.StringFlag{Name: "scene", Usage: "Scene id, name, or seq"},
			&urfavecli.IntFlag{Name: "index", Usage: "Scene index from scenes list (1-based)"},
		),
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			format, err := parseFormat(cmd.String("format"))
			if err != nil {
				return err
			}
			renderScript := resolveRenderScript(cmd.Bool("render-script"), cmd.IsSet("render-script"), format)

			bundle, err := borumi.ResolveBundle(cmd.String("bundle"))
			if err != nil {
				return err
			}

			scene, err := resolveScene(bundle.DBPath, cmd.String("scene"), cmd.Int("index"))
			if err != nil {
				return err
			}

			if format == formatJSON {
				jsonScene, err := sceneForJSONOutput(scene, renderScript)
				if err != nil {
					return err
				}
				return writeJSON(commandWriter(cmd), jsonScene)
			}

			fmt.Fprintln(commandWriter(cmd), renderSceneGetOutput(scene, renderScript))
			return nil
		},
	}
}

func newScenesSetScriptCommand() *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "set-script",
		Usage: "Update one scene script",
		Flags: append(sceneOutputFlags(),
			&urfavecli.StringFlag{Name: "scene", Usage: "Scene id, name, or seq"},
			&urfavecli.IntFlag{Name: "index", Usage: "Scene index from scenes list (1-based)"},
			&urfavecli.StringFlag{Name: "script", Usage: "New script text"},
			&urfavecli.StringFlag{Name: "script-file", Usage: "Path to file containing new script text"},
		),
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			newScript, err := readScriptInput(cmd.String("script"), cmd.String("script-file"))
			if err != nil {
				return err
			}

			scriptJSON, err := prepareScriptForStorage(newScript)
			if err != nil {
				return err
			}

			format, err := parseFormat(cmd.String("format"))
			if err != nil {
				return err
			}
			renderScript := resolveRenderScript(cmd.Bool("render-script"), cmd.IsSet("render-script"), format)

			bundle, err := borumi.ResolveBundle(cmd.String("bundle"))
			if err != nil {
				return err
			}

			targetScene, err := resolveScene(bundle.DBPath, cmd.String("scene"), cmd.Int("index"))
			if err != nil {
				return err
			}

			updated, err := borumi.SetSceneScript(bundle.DBPath, targetScene.ID, scriptJSON)
			if err != nil {
				return err
			}

			if format == formatJSON {
				jsonScene, err := sceneForJSONOutput(updated, renderScript)
				if err != nil {
					return err
				}
				return writeJSON(commandWriter(cmd), jsonScene)
			}

			displayScript := renderScriptForDisplay(updated.Script)
			if !renderScript {
				displayScript = updated.Script
			}
			fmt.Fprintln(commandWriter(cmd), Properties([][2]string{
				{"status", "updated"},
				{"id", updated.ID},
				{"index", fmt.Sprintf("%d", updated.Index)},
				{"seq", updated.Seq},
				{"name", updated.Name},
				{"script", displayScript},
			}))
			return nil
		},
	}
}

func sceneOutputFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "bundle", Value: ".", Usage: "Bundle path"},
		&urfavecli.StringFlag{Name: "format", Value: string(formatTable), Usage: "Output format: table|json"},
		&urfavecli.BoolFlag{Name: "render-script", Usage: "Render script as markdown"},
	}
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

func writeJSON(stdout io.Writer, value any) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to write json output: %w", err)
	}
	return nil
}

func commandWriter(cmd *urfavecli.Command) io.Writer {
	if cmd.Writer != nil {
		return cmd.Writer
	}
	root := cmd.Root()
	if root != nil && root.Writer != nil {
		return root.Writer
	}
	return os.Stdout
}

func setCommandWriters(cmd *urfavecli.Command, stdout io.Writer, stderr io.Writer) {
	if cmd.Writer == nil {
		cmd.Writer = stdout
	}
	if cmd.ErrWriter == nil {
		cmd.ErrWriter = stderr
	}
	for _, child := range cmd.Commands {
		setCommandWriters(child, stdout, stderr)
	}
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
	table := Properties([][2]string{
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
