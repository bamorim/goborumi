# Product Requirements Document (Draft)

> Temporary document to drive discovery and implementation. This file is expected to evolve heavily and can be removed later.

## 1. Product summary

`borumi-cli` is a Go CLI that provides a reliable interface for operating on Borumi projects, starting with local `.bmprojbundle` files and eventually powering agent skills/workflows.

## 2. Problem statement

Borumi-related operations currently live mostly in ad-hoc workflows (including agent skill instructions). This creates friction:

- operations are hard to test and version as software
- logic is duplicated across prompts/workflows
- behavior can drift based on tool/runtime context

We need a canonical, scriptable, and testable command-line interface.

## 3. Goals

- Provide a stable command interface for Borumi project operations.
- Encapsulate project parsing and mutations in Go code with tests.
- Support local automation and agent integration through deterministic outputs.
- Make commands composable in shell scripts and CI.
- Prioritize a great human CLI UX first (agents can use `--format json` + tools like `jq`).

## 4. Non-goals (initially)

- Re-implementing Borumi desktop UI features.
- Building a public cloud service in this repository.
- Full backward compatibility guarantees before v1 scope is clear.

## 5. Primary users

- Internal/advanced Borumi users automating repetitive workflows.
- Agent/skill maintainers who need dependable primitives.
- Developers building tooling around Borumi project artifacts.

## 6. Initial domain assumptions

Observed local Borumi bundles contain:

- `project.bmproj` SQLite database
- `medias/` directory with media assets (`.mov`, `.wav`, `.cursor`, etc.)

Observed `project.bmproj` tables include:

- `projects`, `scenes`, `takes`, `clips`, `medias`
- `project_sources`, `project_timeline`, `scene_timeline_segments`
- `_borumi_migrations`

These assumptions should be validated against more projects and versions.

## 7. CLI principles

- Human-friendly defaults with machine-readable output modes.
- Safe-by-default: explicit flags for destructive mutations.
- Predictable exit codes and error messages.
- Fast local-first operations.

## 8. Candidate feature areas (to prioritize)

- Project introspection
  - detect/validate bundle structure
  - summarize project metadata and asset counts
- Pre-production
  - view/write scene scripts
- Data export
  - export timelines/scenes/takes/metadata to JSON
- Post-production
  - extract final edit scene timestamps for YouTube chapters/timestamps
- Edit phase
  - extract voice transcripts to feed animation/video tooling (for example Remotion)
- Data quality checks
  - detect missing media references or inconsistent records
- Project utilities
  - rename/move/normalize bundle contents
  - cleanup derived/transient data

## 9. Product decisions captured (Feb 17, 2026)

1. Priority workflows:
   - Pre-prod: view/write scene scripts.
   - Post-prod: extract final edit scene timestamps for YouTube timestamps.
   - Edit phase: extract voice transcripts for downstream animation tooling.
2. Writes are allowed for low-risk operations (for example scene script editing). Destructive operations (delete flows) are out of scope for now.
3. Binary name: `borumi-cli`.
4. Output format: default `table` + opt-in `--format json`.
5. Scope: single project first (no batch mode requirement right now).
6. Schema/version strategy: keep initial implementation simple and pragmatic.
7. Agent strategy: do not over-optimize for machine contracts; human-friendly CLI + JSON mode is sufficient.
8. First end-to-end win: reading scene scripts.

## 10. Remaining open questions

1. Should script writes preserve trailing newlines exactly from `--script-file`, or should we normalize?
2. Should scene selection allow exact `name` matching, or only `id/seq` for safety?
3. How should we resolve and expose "final edit scene timestamps" from current Borumi data model?
4. What transcript granularity is required first (full media transcript vs. per-scene snippets)?

## 11. Success criteria (draft)

- At least one workflow currently done via skill instructions can be executed end-to-end via CLI command(s).
- Output is stable enough to be consumed by an updated Borumi skill.
- Core command paths covered by automated tests.

## 12. Risks

- Borumi schema drift over time without formal schema contracts.
- Ambiguity on what operations are safe vs. destructive.
- Prematurely locking command names before workflow discovery.

## 13. Milestones (draft)

- M0: Bootstrap repo, document direction, establish Go skeleton.
- M1: Scene script read/write commands (`scenes list|get|set-script`) with table/json output.
- M2: Structured JSON output and tests.
- M3: Extract final edit scene timestamps.
- M4: Extract voice transcripts for downstream animation tooling.
- M5: Integrate/replace existing skill workflows with CLI usage.
