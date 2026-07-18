# `pnpm run`, `pnpm exec`, `pnpm dlx`, `pnpm create`

The four commands for running code in pnpm's scope: running declared scripts, executing bins, one-off packages, and project scaffolders. For the full command list, see [cli.md](cli.md); for script-related settings, see [config.md](config.md).

> **Source of truth**: <https://pnpm.io/cli/run>, <https://pnpm.io/cli/exec>, <https://pnpm.io/cli/dlx>, <https://pnpm.io/cli/create>. Verified against pnpm v10–v11.

## Quick Comparison

| Command | What it runs | Where the bin comes from | Modifies project? |
|---------|--------------|--------------------------|-------------------|
| `pnpm run <script>` | A script defined in `package.json` `scripts` | — | No (unless the script does) |
| `pnpm <script>` | Same as `pnpm run <script>` if no command conflicts | — | No |
| `pnpm exec <cmd>` | Any binary, in project scope | `node_modules/.bin` (project + workspace) | No |
| `pnpm dlx <pkg>` | A package's default bin, fetched on the fly | Hotloaded from the registry into a temp scope | No (uses a cache) |
| `pnpm create <starter>` | A `create-*` package's bin | Fetched from the registry | Yes (writes new files) |

## `pnpm run`

Runs a script from `package.json`:

```json
{
  "scripts": {
    "build": "tsc",
    "test": "jest",
    "dev": "vite"
  }
}
```

```bash
pnpm run build        # explicit
pnpm build            # shorthand: pnpm <script> works if no command named "build" exists
pnpm run build -- --watch   # args after -- go to the script
```

### Lifecycle scripts

pnpm runs `pre<script>` and `post<script>` automatically (when `enablePrePostScripts: true`, the default):

```json
{
  "scripts": {
    "prebuild": "rimraf dist",
    "build": "tsc",
    "postbuild": "node post-build.js"
  }
}
```

`pnpm run build` runs `prebuild` → `build` → `postbuild`. The standard npm lifecycle names (`preinstall`/`install`/`postinstall`, `prepublish`/`prepare`, etc.) are honored.

### Recursive `pnpm run`

```bash
pnpm -r run build              # run "build" in every workspace package, topologically
pnpm -r run build --parallel   # in parallel (unordered)
pnpm -r run build --stream     # stream output, prefixed with package name
pnpm -r run test --no-bail     # don't stop on first failure
pnpm -r run build --resume-from packages/widget   # start from this package
pnpm -r run build --if-present # skip packages without a "build" script
pnpm -r run build --report-summary  # write exit codes to a file
```

See [workspace.md](workspace.md) for the recursive model.

### `pnpm run` flags

| Flag | Effect |
|------|--------|
| `--recursive`, `-r` | Run across all workspace packages. |
| `--if-present` | Skip packages that don't define the script. |
| `--no-bail` | Continue after a failure; report worst exit code at end. |
| `--parallel` | Run all packages' scripts in parallel. |
| `--stream` | Stream output prefixed with package name. |
| `--aggregate-output` | Buffer each package's output; print whole at end. |
| `--resume-from <pkg>` | Resume a topological run from a package. |
| `--report-summary` | Write exit codes to `pnpm-workspace-summary.json`. |
| `--reporter-hide-prefix` | Don't prefix output with package name. |
| `--silent`, `-s` | Suppress pnpm's own output (script output still shows). |

### Script environment

When pnpm runs a script, it sets:

- `PATH` with `node_modules/.bin` (and workspace `node_modules/.bin`) prepended.
- `npm_lifecycle_event` — the script name.
- `npm_command` — the high-level command (`run-script`, `install`, etc.).
- `npm_config_*` — populated from pnpm's resolved config (for compat with scripts that read them).
- `pnpm_config_*` — pnpm's own config env vars.
- `PNPM_PACKAGE_NAME` — the current package's name (in recursive runs).
- `INIT_CWD` — the directory pnpm was invoked from.
- `NODE_OPTIONS` — preserved.

### `scriptShell` and `shellEmulator`

By default, pnpm runs scripts through the system shell (`/bin/sh` on Unix, `cmd.exe` on Windows). Override with:

```yaml
scriptShell: /bin/bash       # use bash explicitly
shellEmulator: true          # use pnpm's JS bash-like shell (no system shell)
```

`shellEmulator: true` uses a cross-platform JS shell — useful on Windows where `cmd.exe` differs from POSIX shells. It supports a subset of bash syntax; not every script works under it.

## `pnpm exec`

Runs a binary from `node_modules/.bin` in the project's scope, without defining a script. Useful for one-off invocations:

```bash
pnpm exec tsc --noEmit
pnpm exec eslint src/
pnpm exec jest path/to/test.test.ts
```

Equivalent to `npx <cmd>` when the cmd is already installed. Differences from `pnpm dlx`:

- `exec` uses bins **already in `node_modules/.bin`**; it doesn't fetch.
- `dlx` fetches a package on the fly (below).

### `pnpm exec` flags

| Flag | Effect |
|------|--------|
| `--recursive`, `-r` | Run in every workspace package. |
| `--parallel` | Run in parallel across packages. |
| `--shell-mode`, `-c` | Run the command in a shell (enables globs, pipes, redirection). |
| `--resume-from <pkg>` | Resume a recursive run. |
| `--report-summary` | Write exit codes to a file. |
| `--no-reporter-hide-prefix` | Show package-name prefix on output. |

Without `--shell-mode`, `pnpm exec` passes args verbatim (no shell expansion). With it, the command is a shell command string.

## `pnpm dlx`

Fetches a package from the registry, hotloads it, and runs its default bin. Alias: `pnpx`. Nothing is added to `package.json` or `node_modules` permanently.

```bash
pnpm dlx cowsay "hello"               # fetch cowsay, run its bin, discard
pnpm dlx create-vite my-app           # fetch create-vite, run it
pnpm dlx --package=pkg-a --package=pkg-b some-bin   # fetch multiple, run some-bin
pnpm dlx --shell-mode -c 'cowsay "hi" | lolcat'     # run in a shell
pnpm dlx --silent prettier --version                 # suppress pnpm output
pnpm dlx react@17 --package=react-cli react-cli     # specific version
```

### `pnpm dlx` flags

| Flag | Effect |
|------|--------|
| `--package <pkg>` | Additional package(s) to fetch (the bin may live in a different package). Repeatable. |
| `--allow-build <pkgs>` | Comma-separated package names allowed to run install scripts (v10+ trusted deps). The packages executed by `dlx` are allowed by default. |
| `--shell-mode`, `-c` | Run in a shell. |
| `--silent`, `-s` | Suppress pnpm's own output. |

### Caching

`pnpm dlx` caches fetched packages for `dlxCacheMaxAge` minutes (default 1440 = 1 day). Subsequent `dlx` of the same package within the window reuses the cache; after it, pnpm re-fetches. The cache lives in the global virtual store (v11) — see [store.md](store.md).

### `catalog:` with `dlx`

`pnpm dlx` supports the `catalog:` protocol for the `--package` argument:

```bash
pnpm dlx --package=catalog:my-tool my-tool
```

This resolves `my-tool`'s version from the workspace catalog. See [catalogs.md](catalogs.md).

## `pnpm create`

Fetches a `create-*` starter package and runs it to scaffold a new project. `pnpm create <name>` is shorthand for `pnpm dlx create-<name>` (pnpm prepends `create-` if the name doesn't already start with it). No documented aliases — use `pnpm create` in full (`c` is the alias for `pnpm config`, not `create`).

```bash
pnpm create vite my-app            # fetches create-vite, runs it with "my-app"
pnpm create next-app               # fetches create-next-app
pnpm create astro@latest           # specific version
pnpm create --allow-build my-saas  # allow the starter to run install scripts
```

`pnpm create <name>` is shorthand for `pnpm dlx create-<name>` — pnpm prepends `create-` if the name doesn't already start with it.

### `pnpm create` flags

| Flag | Effect |
|------|--------|
| `--allow-build <pkgs>` | Comma-separated package names allowed to run install scripts (v10+). |

Args after the package name are passed to the starter's bin.

## Trusted Dependencies (v10+, `allowBuilds` in v11)

Since v10, pnpm does **not** run install scripts (`preinstall`/`install`/`postinstall`) for dependencies by default. A package must be marked as trusted to run its scripts. You mark trust:

- At add time: `pnpm add --allow-build <pkgs>` (comma-separated). The flag also writes the entries into `allowBuilds` in `pnpm-workspace.yaml` for future installs.
- In config (`pnpm-workspace.yaml`): `allowBuilds` — a map of package matchers to `true`/`false` (v11). Replaces the v10 list settings `onlyBuiltDependencies` / `neverBuiltDependencies` / `ignoredBuiltDependencies`, which were removed in v11.

```yaml
# pnpm-workspace.yaml (v11)
allowBuilds:
  esbuild: true
  core-js: false
  nx@21.6.4 || 21.6.5: true
```

- To allow all packages' install scripts (dangerous): `dangerouslyAllowAllBuilds: true`.
- `strictDepBuilds` (default `true` in v11) makes the install **exit non-zero** when any unlisted dependency has build scripts; set `false` to downgrade to a warning.

This affects `pnpm dlx`/`pnpm create` too: pass `--allow-build` if the fetched package needs its install scripts (native add-ons, etc.). The packages a `dlx` invocation actually executes are allowed by default.

## Edge Cases and Gotchas

- **`pnpm <script>` vs `pnpm run <script>`.** The shorthand works only if no pnpm command has the same name. `pnpm install` is a command, so `pnpm install` runs the command, not a script named "install". `pnpm build` runs the script (no `build` command). When in doubt, use `pnpm run`.
- **Args after `--`.** `pnpm run build -- --watch` passes `--watch` to the script. Without `--`, pnpm would try to parse `--watch` as its own flag.
- **`exec` vs `dlx` for installed tools.** If the tool is in your `devDependencies`, use `pnpm exec` (no fetch). If it's not and you don't want to add it, use `pnpm dlx`.
- **`dlx` and `npx`.** `pnpm dlx` is pnpm's `npx`. It uses pnpm's store and cache; `npx` uses npm's. Don't mix in a pnpm project — `dlx` integrates with the pnpm store.
- **Windows and `shellEmulator`.** Scripts with POSIX shell syntax (`&&`, `&&`, `$VAR`) may fail on Windows under `cmd.exe`. Set `shellEmulator: true` for a consistent cross-platform shell, or `scriptShell: bash` if Git Bash is available.
- **Lifecycle scripts and `--ignore-scripts`.** `pnpm install --ignore-scripts` skips `preinstall`/`install`/`postinstall` for dependencies, but `pnpm run <script>` still runs `pre<script>`/`post<script>` for your own scripts. `--ignore-scripts` is about dependency install scripts, not your `pnpm run`.
- **Recursive `run` and topological order.** `pnpm -r run build` builds in dependency order by default. If package A depends on workspace package B, B builds first. `--parallel` disables this — use only when order doesn't matter.

## References

- <https://pnpm.io/cli/run>
- <https://pnpm.io/cli/exec>
- <https://pnpm.io/cli/dlx>
- <https://pnpm.io/cli/create>

## Related

- Full command list → [cli.md](cli.md)
- Recursive execution → [workspace.md](workspace.md)
- Script shell settings → [config.md](config.md)
- Store used by `dlx` → [store.md](store.md)
