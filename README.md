# goborumi

`goborumi` is a Go command-line tool to inspect, automate, and manipulate Borumi projects.

This repository is intentionally in bootstrap mode. We are defining product direction first and will iterate quickly on command design.

## Why this exists

The current Borumi workflow has useful operations represented in an agent `SKILL.md`. This CLI is intended to become the stable execution layer behind that skill so automations are:

- easier to version and test
- reusable outside an agent runtime
- more reliable for repeatable project operations

## Early scope

The first focus is local Borumi project bundles (`.bmprojbundle`), which currently look like:

- `project.bmproj` (SQLite database)
- `medias/` (media assets and related files)

## Current status

- Go project bootstrapped
- Product requirements tracked in `PRD.md`
- First command set implemented for scene script management

## Development with mise

This project uses [`mise`](https://mise.jdx.dev/) to pin the Go toolchain and provide common dev tasks.

```bash
mise install
mise tasks
```

Common tasks:

- `mise run build` - build `bin/goborumi`
- `mise run test` - run unit tests
- `mise run test-race` - run tests with race detector
- `mise run fmt` - format Go code
- `mise run vet` - run static checks
- `mise run tidy` - sync `go.mod`/`go.sum`
- `mise run check` - run format, vet, and tests

## Current commands

`goborumi` currently focuses on scene scripts (the first end-to-end win).

- `goborumi scenes list --bundle <path>`
- `goborumi scenes get --bundle <path> --index <n>`
- `goborumi scenes get --bundle <path> --scene <id-or-name>`
- `goborumi scenes set-script --bundle <path> --index <n> --script "<text>"`
- `goborumi scenes set-script --bundle <path> --scene <id-or-name> --script-file <file>`

Output defaults to table format and supports JSON:

- `--format table` (default)
- `--format json`

## Example

```bash
goborumi scenes get \
  --bundle "$HOME/Borumi Projects/How to build an AI agent.bmprojbundle" \
  --index 1 \
  --format table
```
