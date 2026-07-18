# pnpm CLI Overview

The full command surface of the `pnpm` executable: commands, aliases, global flags, and exit codes. For the install pipeline specifically, see [install.md](install.md); for run/exec/dlx/create, see [run-exec-dlx.md](run-exec-dlx.md); for publish/pack, see [publish-pack.md](publish-pack.md).

> **Source of truth**: <https://pnpm.io/cli>. Command names and flags verified against pnpm v10–v11 docs.

## Command Reference

### Lifecycle and dependency management

| Command | Aliases | Purpose |
|---------|---------|---------|
| `pnpm install` | `i` | Install all deps from lockfile/manifest. See [install.md](install.md). |
| `pnpm add <pkg...>` | | Add a package (saves to `dependencies` by default). |
| `pnpm remove <pkg...>` | `rm`, `uninstall`, `un` | Remove a package. |
| `pnpm update [pkg...]` | `up`, `upgrade` | Update packages within their semver range (or `--latest` to jump ranges). |
| `pnpm outdated [pkg...]` | | Check for newer versions than the locked/declared range. |
| `pnpm prune` | | Remove unneeded dev/optional packages from `node_modules`. |
| `pnpm fetch` | | Resolve + fetch only (no link). Designed for Docker layers. |
| `pnpm dedupe` | | Remove older deps from lockfile if a newer resolved version already satisfies. |

### Run and execute

| Command | Aliases | Purpose |
|---------|---------|---------|
| `pnpm run <script>` | `run-script` | Run a `package.json` script. Bare `pnpm <script>` works if no command conflicts. |
| `pnpm exec <cmd>` | | Run a binary from `node_modules/.bin` in project scope. |
| `pnpm dlx <pkg>` | `pnpx` | Fetch a package, hotload, run its default bin. |
| `pnpm create <starter>` | | Create a project from a `create-*` starter. |

See [run-exec-dlx.md](run-exec-dlx.md) for the full model.

### Publish and pack

| Command | Purpose |
|---------|---------|
| `pnpm publish` | Publish to the registry (native implementation since v11). |
| `pnpm pack` | Create a tarball without publishing. |

See [publish-pack.md](publish-pack.md).

### Inspection

| Command | Aliases | Purpose |
|---------|---------|---------|
| `pnpm list` | `ls` | Print the dependency tree. |
| `pnpm why <pkg>` | | Show who depends on a package (reverse tree). |
| `pnpm audit` | | Check for security vulnerabilities. Subcommand: `audit signatures`. |
| `pnpm doctor` | | Diagnose common configuration/environment issues. |

### Store

| Command | Purpose |
|---------|---------|
| `pnpm store status` | Check for modified/edited packages in the store. |
| `pnpm store add <pkg...>` | Add packages to the store without modifying a project. |
| `pnpm store prune` | Remove unreferenced packages from the store. |
| `pnpm store path` | Print the active store directory. |

See [store.md](store.md).

### Patching

| Command | Purpose |
|---------|---------|
| `pnpm patch <pkg>` | Extract a dependency into a temp dir for editing. |
| `pnpm patch-commit <dir>` | Generate a `.patch` file and register it. |

See [overrides-patch.md](overrides-patch.md).

### Linking

| Command | Purpose |
|---------|---------|
| `pnpm link <dir>` | Symlink a local package into the current project's `node_modules`. |
| `pnpm unlink <pkg...>` | Remove a previously linked package. |

### Environment and config

| Command | Aliases | Purpose |
|---------|---------|---------|
| `pnpm config <subcmd>` | `c` | `set` / `get` / `delete` / `list` config keys. |
| `pnpm runtime set <runtime> <ver>` | `rt` | Manage runtimes (node, deno, bun). Replaces the deprecated `pnpm env`. |
| `pnpm env <subcmd>` | | (Deprecated) Legacy Node.js version management. |
| `pnpm bin` | | Print the bin directory (`-g` for global). |
| `pnpm setup` | | Create pnpm home, add to PATH, copy executable. Run once after install. |
| `pnpm rebuild [pkg...]` | `rb` | Rebuild a package (re-run its install scripts). |

## Global Flags

These flags apply to most commands. Command-specific flags live in each command's reference.

### Selection

| Flag | Purpose |
|------|---------|
| `--recursive`, `-r` | Run the command across all workspace packages. See [workspace.md](workspace.md). |
| `--filter <selector>` | Select packages by name, path, or dependency relation. |
| `--workspace-root` | Run against the workspace root (alias `-w`). |
| `--include-workspace-root` | Include the root in recursive operations. |
| `--global`, `-g` | Operate on the global install location. |

### Save target (`pnpm add` / `pnpm update`)

| Flag | Saves to |
|------|----------|
| `--save-dev`, `-D` | `devDependencies` |
| `--save-optional`, `-O` | `optionalDependencies` |
| `--save-peer` | `peerDependencies` (and `--save-prod` adds it to prod too) |
| `--save-exact`, `-E` | Pin exact version (no `^`/`~`) |
| `--save-catalog` | Add using `catalog:` protocol. See [catalogs.md](catalogs.md). |
| `--workspace` | Require the package to be a workspace member. |
| `--allow-build <pkgs>` | Allow the listed package(s) to run install scripts and add them to `allowBuilds` (v10+; see [config.md](config.md)). |

### Dependency class filters

| Flag | Effect |
|------|--------|
| `--prod`, `-P` | Only production dependencies. |
| `--dev`, `-D` | Only dev dependencies (note: same short flag as `--save-dev` — meaning depends on command). |
| `--no-optional` | Skip optional dependencies. |

### Output

| Flag | Effect |
|------|--------|
| `--silent`, `-s` | Suppress all output. |
| `--reporter <ndjson|default|silent>` | Output format for install. |
| `--long` | Show more detail (`outdated`, `list`, `why`). |
| `--json` | Emit JSON (`list`, `outdated`, `why`, `audit`, `pack`, `publish`). |
| `--no-color` | Disable ANSI colors. |
| `--report-summary` | Write last command's exit code to a file (recursive runs). |

### Other

| Flag | Effect |
|------|--------|
| `--offline` | Skip network; use the store only. |
| `--prefer-offline` | Use the store; hit the network only if missing. |
| `--ignore-scripts` | Skip `preinstall`/`install`/`postinstall` scripts. |
| `--ignore-pnpmfile` | Skip `.pnpmfile.mjs` hooks. See [hooks.md](hooks.md). |
| `--node-linker <isolated\|hoisted\|pnp>` | Override the linker for this run. |
| `--shell-mode`, `-c` | (`exec`/`dlx`) Run in a shell with glob/expansion. |
| `--help`, `-h` | Show help for a command. |
| `--version`, `-v` | Print pnpm version. |

## Filtering Syntax (`--filter`)

`--filter` selects workspace packages for recursive operations. A filter argument is a comma-separated list of selectors (union). Each selector has the form:

```
selector  = "!"? prefix? base suffix?
prefix    = "..." "^"?            (dependents; optional excludeSelf)
suffix    = "^"? "..."            (dependencies; optional excludeSelf)
base      = name? "{...}"? "[ref]"?
          | location
```

### Base selectors

| Selector | Kind | Matches |
|----------|------|---------|
| `foo` | name | A package by its `name` (supports globs: `@pnpm.e2e/*`) |
| `@scope/bar` | name | A scoped package |
| `./packages/*` | path | Packages whose directory matches the glob |
| `../shared` | path | A relative path |
| `.` | self | The package in the current directory |
| `..` | parent | The package in the parent directory |
| `{foo}` | brace | A directory selector resolved against the workspace prefix |
| `{./apps/*,./packages/*}` | brace | A brace group (comma is part of the glob, not a separator) |
| `[master]` | diff | Packages changed since the `master` ref |
| `pattern{foo}` | name+brace | Name `pattern` in directory `foo` |
| `pattern{foo}[master]` | name+brace+diff | Name `pattern` in dir `foo`, changed since `master` |

### Relational modifiers

The prefix `...`, suffix `...`, and `^` are **orthogonal** — any combination is valid:

| Form | Dependents | Dependencies | ExcludeSelf |
|------|------------|--------------|-------------|
| `foo` | | | |
| `foo...` | | ✓ | |
| `foo^...` | | ✓ | ✓ |
| `...foo` | ✓ | | |
| `...^foo` | ✓ | | ✓ |
| `...foo...` | ✓ | ✓ | |
| `...^foo^...` | ✓ | ✓ | ✓ |

- **`...` suffix** (`foo...`) — the package **and its dependencies** (packages `foo` depends on).
- **`...` prefix** (`...foo`) — the package **and its dependents** (packages that depend on `foo`).
- **`^` adjacent to `...`** — exclude the matched package itself, keeping only its relations.

### Negation and combination

| Form | Meaning |
|------|---------|
| `!foo` | Exclude `foo` from the selection |
| `foo,bar` | Union of `foo` and `bar` |
| `foo..., !bar` | Dependents of `foo`, excluding `bar` |
| `!{./apps/legacy}` | Exclude the legacy app directory |

`--filter` can be repeated (`--filter foo --filter bar`) or use comma-separated unions within one argument. `--filter` implies recursive execution.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success. |
| `1` | General failure (generic error). |
| `2` | No packages matched the filter (recursive runs). |
| other | propagated from a failed child script or the command-specific error path. |

A failed script in `pnpm -r run` exits with that script's code unless `--no-bail` is given (then it continues and reports the worst exit code at the end via `--report-summary`).

## References

- Command index: <https://pnpm.io/cli>
- Filtering: <https://pnpm.io/filtering>

## Related

- Install pipeline → [install.md](install.md)
- Run/exec/dlx → [run-exec-dlx.md](run-exec-dlx.md)
- Workspace selection → [workspace.md](workspace.md)
