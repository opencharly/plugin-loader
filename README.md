# plugin-loader

The unified-config loader plugin for [opencharly/charly](https://github.com/opencharly/charly) —
the config front-end every command reaches before any project is read.

The loader is split PARSE / MATERIALIZE: this plugin PARSES the project (file read
→ import resolution → YAML multi-doc → the `#NodeDoc` CUE gate → reserved-word
node decomposition → discover walk) into a generic, SDK-expressible
`ParsedProject` (kind-keyed opaque nodes + the resolved import/discover
structure); the HOST MATERIALIZES the typed `*UnifiedFile` from it.

## What it provides

| Capability | Surface |
|---|---|
| `loader:loader` | the swappable config front-end — `spec.DocParser` (per-document parse) + `spec.ProjectWalker` (whole-project walk) |

Both typed seams delegate to the shared `sdk/loaderkit` (`loaderkit.ParseDoc` /
`loaderkit.Walk`) — the ONE copy of the parse+walk mechanism. An alternative
loader plugin serves a different config front-end / walk mechanism by
implementing the same two interfaces.

## Placement

The loader is **compiled into charly** (`compiled_plugins:`): it must ALWAYS
resolve — it IS the config front-end every command reaches. It is registered at
`init()` before the first load; the host calls its typed `ParseDoc` /
`WalkProject` directly (no wire envelope). The bootstrap seed stays core and
never calls the loader, so there is no bootstrap cycle.

## How to use it

The plugin is compiled into charly — no candy composition is needed.

## Layout

- `candy/plugin-loader/` — the plugin module: `plugin.go`, `refs_seams.go`,
  `schema/loader.cue`, `cmd/serve/main.go` (plus the plugin's own Go tests).
- `candy/plugin-loader/charly.yml` — the `plugin-loader:` candy entity.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-internals:go` — the unified YAML loader, the SDD
  pipeline, and the source map. This candy carries no `skill:` entity of its
  own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin/provider model, including the `loader`
  provider class.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
