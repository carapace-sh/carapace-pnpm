# Workspaces (Monorepos)

The `pnpm-workspace.yaml` format, workspace member selection, recursive execution, and the workspace-specific settings. For the global config split (settings moved into `pnpm-workspace.yaml` in v11), see [config.md](config.md); for catalogs/overrides/patches defined in the workspace file, see the dedicated references.

> **Source of truth**: <https://pnpm.io/workspaces>, <https://pnpm.io/pnpm-workspace_yaml>. Verified against pnpm v10–v11.

## What a Workspace Is

A pnpm workspace is a monorepo: a single root project with multiple packages under it, sharing one lockfile (by default) and one install. The root has a `pnpm-workspace.yaml` listing the member packages; each member has its own `package.json`.

```
my-monorepo/
├── pnpm-workspace.yaml     # lists members
├── package.json            # root manifest
├── pnpm-lock.yaml          # shared lockfile (sharedWorkspaceLockfile: true)
└── packages/
    ├── widget/
    │   └── package.json
    ├── app/
    │   └── package.json
    └── shared/
        └── package.json
```

## `pnpm-workspace.yaml` — Member List

```yaml
packages:
  - 'packages/*'           # all direct children of packages/
  - 'components/**'         # recursively, all under components/
  - 'apps/*'
  - '!**/test/**'           # exclude any path matching this
  - 'tools/*'
```

Globs are evaluated relative to the file's directory. `!` prefixes exclude. A path is a workspace member if it contains a `package.json` and matches a positive pattern without matching a `!` exclusion.

## `pnpm-workspace.yaml` — Other Top-Level Keys

Since v11, `pnpm-workspace.yaml` is **also** the home for most non-auth settings (see [config.md](config.md) for the full settings list). Workspace-specific keys:

| Key | Type | Purpose |
|-----|------|---------|
| `packages` | string[] | Member globs (above). |
| `catalog` | map | Default catalog. See [catalogs.md](catalogs.md). |
| `catalogs` | map<name, map> | Named catalogs. |
| `catalogMode` | `manual` \| `strict` \| `prefer` | Catalog enforcement. |
| `overrides` | map | Force dependency versions. See [overrides-patch.md](overrides-patch.md). |
| `packageExtensions` | map | Extend package manifests. |
| `patchedDependencies` | map | Patch files. See [overrides-patch.md](overrides-patch.md). |
| `packageConfigs` | map \| list | Per-project setting overrides (v11). |
| `allowedDeprecatedVersions` | map | Suppress deprecation warnings per package. |
| `minimumReleaseAge` | number | Min minutes since publish before a version is allowed. |
| `minimumReleaseAgeExclude` | string[] | Packages exempt from min release age. |

Plus all the general settings (`nodeLinker`, `hoist`, `autoInstallPeers`, etc.) — see [config.md](config.md).

## Recursive Execution (`-r` / `--recursive`)

Most commands take `-r` to run across all workspace members:

```bash
pnpm -r install               # install for the whole workspace
pnpm -r run build             # run "build" in every package that has it
pnpm -r run test --no-bail    # run tests everywhere, don't stop on first failure
pnpm -r exec tsc --noEmit     # run tsc in every package
pnpm -r --filter app-* build  # run build in packages matching app-*
```

### Recursive flags

| Flag | Effect |
|------|--------|
| `--recursive`, `-r` | Run in all workspace packages. |
| `--filter <selector>` | Restrict to matching packages. See [cli.md](cli.md). |
| `--workspace-root`, `-w` | Run against the root, not the packages. |
| `--include-workspace-root` | Include the root in `-r` runs (excluded by default). |
| `--no-bail` | Continue after a package's command fails; report worst exit code at end. |
| `--parallel` | Run packages' commands in parallel (unordered output). |
| `--stream` | Stream each package's output prefixed with its name. |
| `--aggregate-output` | Buffer each package's output and print it whole at the end. |
| `--resume-from <pkg>` | Start the recursive run from a specific package (topological order). |
| `--report-summary` | Write each package's exit code to a JSON file for inspection. |
| `--if-present` | Skip packages that don't define the script (instead of erroring). |

### Topological order

`pnpm -r run build` runs builds in dependency order: a package builds after its workspace dependencies build. `--parallel` disables this. `--resume-from` restarts a partially-failed run from a known point.

## Workspace Dependencies (the `workspace:` protocol)

Workspace packages depend on each other using the `workspace:` protocol:

```json
{
  "dependencies": {
    "shared": "workspace:*",
    "widget": "workspace:^1.2.3",
    "utils": "workspace:~"
  }
}
```

| Range | Meaning |
|-------|---------|
| `workspace:*` | Any version of the workspace package. |
| `workspace:^` / `workspace:~` | Use the package's own version, prefixed with `^` / `~`. |
| `workspace:^1.2.3` | Match the workspace package at this exact range. |

On publish, `workspace:` ranges are replaced with the workspace package's concrete version (controlled by `saveWorkspaceProtocol`). This is what makes workspace packages publishable as if they were independent.

### `saveWorkspaceProtocol`

| Value | Behavior on `pnpm add` of a workspace package |
|-------|-----------------------------------------------|
| `rolling` (default) | Write `workspace:*` if the target's version is a range, else `workspace:^<ver>`. |
| `true` | Always write `workspace:*`. |
| `false` | Never write `workspace:`; use the concrete version. |

## Linking Workspace Packages

| Setting | Default | Effect |
|---------|---------|--------|
| `linkWorkspacePackages` | `false` | `true` symlinks workspace deps; `deep` hard-links their files instead. |
| `injectWorkspacePackages` | `false` | `true` copies (injects) the workspace package's built output into the consumer, instead of symlinking. Good for deployed bundles. |
| `dedupeInjectedDeps` | `true` | Dedupe injected deps across consumers. |
| `preferWorkspacePackages` | `false` | Prefer workspace packages over registry ones when ranges match. |
| `hoistWorkspacePackages` | `true` | Also symlink workspace packages to the hoisted location. |

### Symlink vs inject

- **Symlink** (`linkWorkspacePackages: true`): consumer's `node_modules/shared` → `../../packages/shared`. Edits to `shared`'s source are immediately visible. Standard dev workflow.
- **Inject** (`injectWorkspacePackages: true`): pnpm runs the workspace package's build and copies the output into the consumer's `node_modules`. Used when consumers can't tolerate symlinks (some bundlers, serverless deploys).

## `packageConfigs` (v11+)

Per-project overrides for any setting:

```yaml
packageConfigs:
  packages/widget:
    saveExact: true
  - match: ["packages/app-a", "packages/app-b"]
    nodeLinker: hoisted
```

A map form keys by project path; a list form with `match` applies the same config to several projects at once. Lets one workspace mix layouts or save policies.

## Shared vs Per-Package Lockfile

| Setting | Default | Effect |
|---------|---------|--------|
| `sharedWorkspaceLockfile` | `true` | One `pnpm-lock.yaml` at the root. |
| (set to `false`) | | Per-package lockfiles. Rare; use when workspace packages are published and built independently with no shared install. |

With the default, the root lockfile's `importers` section has one entry per workspace project (see [lockfile.md](lockfile.md)).

## Edge Cases and Gotchas

- **`--workspace-root` vs `--include-workspace-root`.** `-w` runs *only* in the root; `--include-workspace-root` runs in the root *and* the packages (with `-r`). They are not synonyms.
- **A package not detected.** Check that it has a `package.json` and that its directory matches a positive glob and no `!` exclusion. A typo'd glob silently matches nothing.
- **`workspace:` not replaced on publish.** If a published package still has `workspace:` in its `package.json`, the publish didn't run the manifest rewrite. Check `saveWorkspaceProtocol` and run `pnpm publish` (not a manual `npm publish`).
- **Recursive script ordering.** `pnpm -r run build` is topological, but only over workspace deps. Cross-package ordering via registry deps is not enforced.
- **One install for the whole workspace.** Adding a dep to one package (`pnpm --filter app add lodash`) updates the shared lockfile and links into just that package's `node_modules`. The root `node_modules` only has the root's direct deps.

## References

- <https://pnpm.io/workspaces>
- <https://pnpm.io/pnpm-workspace_yaml>
- <https://pnpm.io/catalogs>
- <https://pnpm.io/filtering>

## Related

- Config keys → [config.md](config.md)
- Lockfile `importers` → [lockfile.md](lockfile.md)
- Catalogs → [catalogs.md](catalogs.md)
- Overrides & patches → [overrides-patch.md](overrides-patch.md)
- Recursive flags detail → [cli.md](cli.md) and [run-exec-dlx.md](run-exec-dlx.md)
