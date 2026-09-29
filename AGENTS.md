# AGENTS.md — plugin-loader

Standalone plugin repo for the unified-config loader (`loader:loader`) — the
config front-end every command reaches before any project is read. The plugin is
a Go module at `candy/plugin-loader/` (module path
`github.com/opencharly/plugin-loader/candy/plugin-loader`); the root `charly.yml`
only declares `discover: candy` so the repo is a project and its candy is
scanned.

Canonical files:

- `candy/plugin-loader/charly.yml` — the `plugin-loader:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-loader/` — the Go source: `plugin.go`, `refs_seams.go`,
  `schema/loader.cue`, `cmd/serve/main.go` (plus the plugin's own Go tests).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:go` — the unified YAML loader, the SDD pipeline, and the
  file-by-file source map. Load before changing the parse/walk mechanism.
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, the per-plugin CUE-schema contract, the
  `loader` provider class.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-loader/` — compile the plugin module.
- `go test ./...` in `candy/plugin-loader/` — the plugin's parse/walk tests.
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- The changed path is exercised by every command + check bed (all reach the
  config front-end before any project is read).

## Modify this repo

- Edit the `plugin-loader:` candy entity, the Go source, and
  `schema/loader.cue` **together**.
- Both typed seams (`spec.DocParser` / `spec.ProjectWalker`) MUST delegate to the
  ONE copy in `sdk/loaderkit` (R3); the parse+walk consults only spec vocabulary +
  yaml + the host-threaded `spec.Threaded` + the host-supplied `spec.WalkSeams`,
  never charly core directly.
- Keep the loader COMPILED-IN: it must always resolve, and the bootstrap seed
  stays core with no bootstrap cycle.

## Landing

- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Load
  `/charly-internals:git-workflow` before any git/PR action; history lives in
  `CHANGELOG/`. Do not restate its rules here.
