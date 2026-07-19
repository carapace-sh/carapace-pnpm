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

- **`parser.go`** — Main parser. `Parse()` → `*Filter` AST with spans. Strict: rejects partial/invalid input (e.g. trailing commas, lone `!`, bare `...`, unterminated `{`/`[`). The parser strips relational modifiers from both ends of each selector body, then parses the remaining base as `name?{brace}?[diff]?` with a location fallback.
- **`completion_parser.go`** — Completion parser. `ParseForCompletion(input)` → `*CompletionContext` describing what tokens are valid at end of input. Tolerant: recovers from incomplete input to report expectations. Uses the same end-stripping logic as the main parser but on the last selector only.
- **`scanner.go`** — Shared character-class helpers (`isWhitespace`). The old per-token scanner methods were removed when the parser switched to whole-selector-body parsing.
- **`ast.go`** — AST types: `Filter`, `Selector`, `SelectorKind`.
- **`completion.go`** — Completion context types: `CompletionContext`, `ExpectedToken`, `ValidRelation`, `SelectorContext`.

Both parsers implement the same grammar. The completion parser mirrors the main parser's relation-stripping logic but operates on the last (in-progress) selector and records expectations instead of building a full AST.

### The pnpm filter grammar

```
filter    = selector ( "," selector )*
selector  = "!"? prefix? base suffix?
prefix    = "..." "^"?            (dependents; optional excludeSelf)
suffix    = "^"? "..."            (dependencies; optional excludeSelf)
base      = name? brace? diff?
          | location
name      = [^.] [^{}[\]]*        (package-name glob; may start with '@')
brace     = "{" [^}]+ "}"         (directory selector; inner resolved vs prefix)
diff      = "[" [^\]]+ "]"        (changed-packages selector; git ref)
location  = "." | ".." | "./"… | "../"…
```

The relational modifiers are **orthogonal booleans**, not a single enum — pnpm allows any combination:

| Form | IncludeDependents | IncludeDependencies | ExcludeSelf |
|------|-------------------|---------------------|-------------|
| `foo` | false | false | false |
| `foo...` | false | true | false |
| `foo^...` | false | true | true |
| `...foo` | true | false | false |
| `...^foo` | true | false | true |
| `...foo...` | true | true | false |
| `...^foo^...` | true | true | true |

`!` negates a selector (subtracts matches from the selection). Selectors are combined with `,` (union). Whitespace is allowed around commas but not within a selector.

### Selector kinds

| `Kind` | Matches |
|--------|---------|
| `PackageName` | A package name or name glob like `foo`, `@scope/bar`, `@pnpm.e2e/*` |
| `Path` | A filesystem path like `./packages/*`, `../shared` |
| `Self` | The `.` selector (current directory) |
| `Parent` | The `..` selector (parent directory) |
| `Brace` | A `{...}` directory selector — inner text resolved against the workspace prefix |
| `Diff` | A `[ref]` changed-packages selector with no name/brace base |

A selector can combine a name with a brace and a diff: `pattern{foo}[master]` has `Name=pattern`, `BraceInner=foo`, `Diff=master`.

### Relational modifiers

The three orthogonal booleans on `Selector`:

| Field | Set by | Meaning |
|-------|--------|---------|
| `IncludeDependents` | leading `...` | Also select packages that depend on the match |
| `IncludeDependencies` | trailing `...` | Also select packages the match depends on |
| `ExcludeSelf` | `^` adjacent to a `...` | Exclude the matched package itself, keeping only its relations |

### File responsibilities

| File | Purpose |
|------|---------|
| `pkg/pnpm/span.go` | `Span` (Start/End byte offsets) and `Pos` types |
| `pkg/pnpm/ast.go` | pnpm AST node types: `Filter`, `Selector`, `SelectorKind` |
| `pkg/pnpm/scanner.go` | Shared character-class helpers (`isWhitespace`) |
| `pkg/pnpm/parser.go` | Main parser + public API: `Parse()`, `parseSelectorBody`, `parseBase` |
| `pkg/pnpm/completion.go` | Completion context types: `CompletionContext`, `ExpectedToken`, `ValidRelation`, `SelectorContext` |
| `pkg/pnpm/completion_parser.go` | Completion parser: `ParseForCompletion()` |
| `pkg/pnpm/parser_test.go` | Parser tests (derived from pnpm's parse_project_selector fixtures) |
| `pkg/pnpm/completion_test.go` | Completion parser tests covering each grammar position |
| `cmd/carapace-pnpm/main.go` | CLI entrypoint |
| `cmd/carapace-pnpm/cmd/root.go` | Root cobra command |
| `cmd/carapace-pnpm/cmd/filter.go` | Filter subcommands |
| `pkg/actions/tools/pnpm/uid.go` | UID generator for carapace action deduplication |
| `pkg/actions/tools/pnpm/completion.go` | `ActionFilters()` — carapace action wiring the parser to completion |
| `pkg/actions/tools/pnpm/sandbox_test.go` | Sandbox tests for the action |

## Key Patterns & Gotchas

### Two parsers must stay in sync

When modifying the grammar in `parser.go`, the same relation-stripping and base-parsing logic must be mirrored in `completion_parser.go`. They share the `isLocation`, `kindForLocation`, and `classifyPartialBase` helpers but have independent entry points (`Parse` vs `ParseForCompletion`).

### Relational modifiers are stripped from both ends

The parser does not scan relation tokens incrementally. Instead, `parseSelectorBody` takes the whole selector text, strips a trailing `...` (with optional preceding `^`), then strips a leading `...` (with optional following `^`), and what remains is the base. This handles all combinations including `...foo...` and `...^foo^...` uniformly. A bare `...` with no base is a syntax error.

### `^` is a modifier, not a relation kind

`^` does not select a different relation — it sets `ExcludeSelf=true` on whichever relation is active. It can appear adjacent to a leading `...` (`...^foo`) or a trailing `...` (`foo^...`) or both (`...^foo^...`). The parser records it as a boolean, not as a separate `RelKind`.

### `[diff]` and `{brace}` are orthogonal to names and relations

A selector base is `name?{brace}?[diff]?` — all three are optional and independent. `[master]` alone is a diff selector; `{foo}` alone is a brace selector; `pattern{foo}[master]` combines all three. Relations apply to the whole base: `[master]...` selects changed packages and their dependencies; `...{./foo}` selects dependents of the brace-matched directory.

### Commas inside `{}` are part of the glob

`{./apps/*,./packages/*}` is a single selector (one brace group), not two selectors. The `selectorRest` function tracks brace depth so top-level commas separate selectors while commas inside `{...}` stay part of the brace inner text. `!` is similarly allowed inside `{...}` for glob exclusion (`{./apps/*,!./apps/legacy}`).

### No whitespace within a selector

`foo bar` is a syntax error. Whitespace is only allowed around `,`. Relational modifiers attach directly to the base with no space (`foo...`, not `foo ...`).

### The action layer shells out to pnpm for dynamic values

`pkg/pnpm/` is pure stdlib (no I/O, no carapace dependency). The action layer (`pkg/actions/tools/pnpm/`) shells out to `pnpm list --json -r --depth -1` to discover workspace packages at completion time:

- `ActionWorkspacePackages()` — completes package names with version descriptions.
- `ActionWorkspacePaths()` — completes `./`-prefixed relative paths with package-name descriptions.

Both use `carapace.ActionExecCommandE` and fall back to an empty action (not an error) when pnpm is unavailable or not in a workspace, so the static selector bases (`.`, `{./`, `[`) still work. The exec output is a JSON array of `{name, version, path, private}`.

## Testing

- Tests use the standard `testing` package and `github.com/carapace-sh/carapace/pkg/sandbox` (for action tests).
- `parser_test.go` — parser tests over realistic pnpm filter examples.
- `completion_test.go` — completion parser tests covering each grammar position.
- `sandbox_test.go` — end-to-end action tests using carapace's sandbox.

## Skills

The `skills/` directory contains the `pnpm` compound skill — reference documentation for pnpm concepts (CLI, layout, lockfile, workspaces, etc.). See `skills/pnpm/SKILL.md`.
