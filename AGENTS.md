# AGENTS.md

## Project Overview

Go library for parsing [pnpm](https://pnpm.io/) filter selectors (the argument to `pnpm --filter`) into an AST, with completion support. Part of the [carapace-sh](https://github.com/carapace-sh) ecosystem (shell completion framework). The module path is `github.com/carapace-sh/carapace-pnpm`.

## Commands

```sh
go test ./...                              # run all tests
go test ./pkg/pnpm/                        # run pnpm parser package tests only
go test -run TestParse ./pkg/pnpm/         # run specific test
go build ./...                             # build all packages
go run . filter "<selector>"               # parse a pnpm filter selector, output AST as JSON
go run . filter-complete "<selector>"      # filter completion context as JSON
```

No Makefile, no linter config, no CI config present.

## Architecture

Cobra-based CLI (`cmd/`) wrapping a recursive-descent parser (`pkg/pnpm/`) with completion actions that wire the parser to carapace (`pkg/actions/tools/pnpm/`).

### CLI (`cmd/`)

- **`root.go`** — Root cobra command with `carapace.Gen(rootCmd).Standalone()`
- **`filter.go`** — `filter` and `filter-complete` subcommands

Entry point is `main.go` at `cmd/carapace-pnpm/`, which calls `cmd.Execute()`.

### Parser (`pkg/pnpm/`)

- **`parser.go`** — Main parser. `Parse()` → `*Filter` AST with spans. Strict: rejects partial/invalid input (e.g. trailing commas, lone `!`, bare `...`).
- **`completion_parser.go`** — Completion parser. `ParseForCompletion(input)` → `*CompletionContext` describing what tokens are valid at end of input. Tolerant: recovers from incomplete input to report expectations.

Both parsers implement the same grammar but independently. The completion parser mirrors the main parser's structure but stops at the cursor and records expectations instead of building a full AST.

### The pnpm filter grammar

```
filter    = selector ( "," selector )*
selector  = [ "!" ] [ "..." ] base [ suffix ]
base      = packageName | pathGlob | "." | ".."
suffix    = "..."       (the package and its dependents)
          | "^..."      (the package and its direct dependencies)
```

The `...` prefix on a base (e.g. `...foo`) selects the package **and its dependencies** (transitive). The `...` suffix (e.g. `foo...`) selects the package **and its dependents**. The `^...` suffix (e.g. `foo^...`) selects the package **and its direct dependencies only**.

`!` negates a selector. Selectors are combined with `,` (union). Whitespace is allowed around commas but not within a selector base.

### Selector kinds

| `Kind` | Matches |
|--------|---------|
| `PackageName` | A bare package name like `foo` or `@scope/bar` |
| `PathGlob` | A filesystem path/glob like `./packages/*`, `../shared`, `{./apps/*}` |
| `Self` | The `.` selector (the package in the current directory) |
| `All` | A selector that effectively matches the whole workspace graph |

### Relational modifiers

| `Relation` | Form | Meaning |
|------------|------|---------|
| `RelNone` | (none) | Just the base package |
| `RelDependents` | `pkg...` | The package and all packages that depend on it |
| `RelDependencies` | `...pkg` | The package and all of its dependencies |
| `RelDirectDependencies` | `pkg^...` | The package and its direct dependencies |

### File responsibilities

| File | Purpose |
|------|---------|
| `pkg/pnpm/span.go` | `Span` (Start/End byte offsets) and `Pos` types |
| `pkg/pnpm/ast.go` | pnpm AST node types: `Filter`, `Selector`, `SelectorKind`, `RelKind` |
| `pkg/pnpm/scanner.go` | Scanner methods (selector bases, relational prefixes/suffixes) for both parsers |
| `pkg/pnpm/parser.go` | Main parser + public API: `Parse()` |
| `pkg/pnpm/completion.go` | Completion context types: `CompletionContext`, `ExpectedToken`, `ValidRelation`, `SelectorContext` |
| `pkg/pnpm/completion_parser.go` | Completion parser: `ParseForCompletion()` |
| `pkg/pnpm/parser_test.go` | Parser tests |
| `pkg/pnpm/completion_test.go` | Completion parser tests |
| `cmd/carapace-pnpm/main.go` | CLI entrypoint |
| `cmd/carapace-pnpm/cmd/root.go` | Root cobra command |
| `cmd/carapace-pnpm/cmd/filter.go` | Filter subcommands |
| `pkg/actions/tools/pnpm/uid.go` | UID generator for carapace action deduplication |
| `pkg/actions/tools/pnpm/completion.go` | `ActionFilters()` — carapace action wiring the parser to completion |
| `pkg/actions/tools/pnpm/sandbox_test.go` | Sandbox tests for the action |

## Key Patterns & Gotchas

### Two parsers must stay in sync

When modifying the grammar in `parser.go` / `scanner.go`, the same changes must be mirrored in `completion_parser.go` (and the cursor-bounded scanner methods in `scanner.go`). They share the character-class helpers but have independent parser types.

### Relational `...` is ambiguous without context

`...` can be a prefix (`...pkg` = dependencies) or a suffix (`pkg...` = dependents). The scanner distinguishes by position: a `...` at the start of a selector (before any base character) and followed by a base-start character is a prefix; a `...` immediately after a base and followed by `,`/whitespace/EOF is a suffix. A bare `...` with no base is a syntax error.

### Path globs contain `.` and `*`

`./packages/*` and `{./apps/*}` are valid bases. The scanner must not mistake the leading `./` of a path for the `.` self-selector or a `...` suffix. The `scanSelectorBase` function handles this by checking the full `...` sequence and its surroundings before stopping.

### No whitespace within a selector

`foo bar` is a syntax error. Whitespace is only allowed around `,`. Relational suffixes attach directly to the base with no space (`foo...`, not `foo ...`).

### The action layer is the I/O boundary

`pkg/pnpm/` is pure stdlib (no I/O, no carapace dependency). All carapace integration and any workspace introspection (reading `pnpm-workspace.yaml` to list package names) belongs in `pkg/actions/tools/pnpm/`. `ActionWorkspacePackages()` is currently a stub that callers fill in.

## Testing

- Tests use the standard `testing` package and `github.com/carapace-sh/carapace/pkg/sandbox` (for action tests).
- `parser_test.go` — parser tests over realistic pnpm filter examples.
- `completion_test.go` — completion parser tests covering each grammar position.
- `sandbox_test.go` — end-to-end action tests using carapace's sandbox.

## Skills

The `skills/` directory contains the `pnpm` compound skill — reference documentation for pnpm concepts (CLI, layout, lockfile, workspaces, etc.). See `skills/pnpm/SKILL.md`.
