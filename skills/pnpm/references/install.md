# The Install Pipeline

How `pnpm install`, `pnpm add`, `pnpm remove`, and `pnpm update` work internally, and the flags that control each stage. For the resulting `node_modules` layout, see [node-modules-layout.md](node-modules-layout.md); for the lockfile written by this pipeline, see [lockfile.md](lockfile.md).

> **Source of truth**: <https://pnpm.io/cli/install>, <https://pnpm.io/cli/add>, <https://pnpm.io/cli/remove>, <https://pnpm.io/cli/update>, <https://pnpm.io/cli/fetch>, <https://pnpm.io/cli/prune>, <https://pnpm.io/cli/dedupe>.

## The Three Stages

```
1. RESOLVE   — read manifests, build the dependency graph, apply overrides, resolve peers
2. FETCH     — download tarballs into the content-addressable store (skip if present)
3. IMPORT    — hard-link (or copy) each package from the store into node_modules/.pnpm
   + LINK   — symlink root node_modules/<pkg> and per-package node_modules/<dep>
   + SCRIPTS — run preinstall/install/postinstall unless --ignore-scripts
```

Each stage is independent and can be skipped or frozen:

| Stage | Skip with | Force-only with |
|-------|-----------|-----------------|
| Resolve | `--frozen-lockfile` (use lockfile graph as-is) | `--lockfile-only` (resolve, write lockfile, don't fetch/link) |
| Fetch | `--offline` (store only) | `pnpm fetch` (resolve + fetch, no link) |
| Import/Link | `--lockfile-only`, `--resolution-only` | — |
| Scripts | `--ignore-scripts` | — |

## `pnpm install`

Installs the project. Reads `package.json`, `pnpm-workspace.yaml`, and `pnpm-lock.yaml`; resolves; fetches; links.

### Key flags

| Flag | Effect |
|------|--------|
| `--frozen-lockfile` | Don't touch `pnpm-lock.yaml`; fail if it doesn't satisfy `package.json`. Use in CI. |
| `--lockfile-only` | Update the lockfile but don't touch `node_modules`. Fast. |
| `--offline` | Never hit the network; fail if a package isn't in the store. |
| `--prefer-offline` | Use the store; go to network only if missing. |
| `--force` | Re-fetch even if the store has the package; rebuild the link tree. |
| `--fix-lockfile` | Repair a corrupted lockfile (re-resolve). |
| `--update-checksums` | Recompute integrity hashes (after manual store edits). |
| `--no-lockfile` | Don't read or write `pnpm-lock.yaml`. |
| `--resolution-only` | Re-resolve and write the lockfile; don't fetch or link. |
| `--shamefully-hoist` | Flat `node_modules` like npm for this install. See [node-modules-layout.md](node-modules-layout.md). |
| `--ignore-scripts` | Skip `preinstall`/`install`/`postinstall`. |
| `--ignore-pnpmfile` | Skip `.pnpmfile.mjs` hooks. See [hooks.md](hooks.md). |
| `--prod`, `-P` | Only production deps. |
| `--dev`, `-D` | Only dev deps. |
| `--no-optional` | Skip optional deps. |
| `--no-runtime` | Don't set up the runtime (Node). |
| `--reporter <ndjson\|default\|silent>` | Output format. |
| `--dry-run` | Report what would happen; don't modify anything. |
| `--merge-git-branch-lockfiles` | Merge branch-specific lockfiles. |
| `--cpu`, `--os`, `--libc` | Install only the binaries for these platforms (smaller install). |

### What triggers a full re-resolve

- `package.json` changed since the lockfile was written.
- `pnpm-workspace.yaml` changed (catalogs, overrides, package list).
- A flag that changes resolution (e.g. `--no-optional`, `--prod`) was passed and the lockfile wasn't generated with it.
- `--force` was passed.

`--frozen-lockfile` short-circuits all of these: if the lockfile is present and valid, pnpm uses it verbatim.

## `pnpm add`

Adds a package and installs it. Updates `package.json` and `pnpm-lock.yaml`, then links.

```bash
pnpm add react react-dom           # to dependencies
pnpm add -D typescript             # to devDependencies
pnpm add -O electron               # to optionalDependencies
pnpm add --save-peer react         # to peerDependencies
pnpm add -E lodash@4.17.21         # exact pin, no ^
pnpm add -g prettier               # global
pnpm add --workspace ./pkg-a       # add a workspace package by path
pnpm add --save-catalog react      # add as "react": "catalog:"
pnpm add --allow-build=esbuild     # allow esbuild to run install scripts and add to allowBuilds (v10+)
pnpm add -D @types/node --cpu x64 --os linux   # platform-specific dev dep
```

### Save-target flags

| Flag | `package.json` field |
|------|----------------------|
| (none) | `dependencies` |
| `-D` / `--save-dev` | `devDependencies` |
| `-O` / `--save-optional` | `optionalDependencies` |
| `--save-peer` | `peerDependencies` |
| `-E` / `--save-exact` | Pin exact version |
| `--save-catalog` | Use `catalog:` protocol (see [catalogs.md](catalogs.md)) |
| `--workspace` | Require a workspace package; fail otherwise |
| `--allow-build=<pkgs>` | Mark listed package(s) as trusted to run install scripts; also writes them to `allowBuilds` (see [config.md](config.md)) |

### Installing from non-registry sources

```bash
pnpm add github:user/repo          # Git URL (default branch)
pnpm add github:user/repo#v1.2.0   # Git tag/branch/commit
pnpm add ./local-pkg               # Local path
pnpm add file:./local-pkg          # Explicit file: protocol
pnpm add https://example.com/x.tgz # Tarball URL
pnpm add npm:@scope/pkg@^1.0.0     # Alias: install @scope/pkg under a different name
```

## `pnpm remove`

Removes a package and uninstalls it. Aliases: `rm`, `uninstall`, `un`.

```bash
pnpm remove lodash          # from dependencies
pnpm remove -D typescript   # from devDependencies
pnpm remove -O electron     # from optionalDependencies
pnpm remove -g prettier     # from the global install
pnpm remove -r lodash       # from every workspace package
```

Removes the package from `package.json`, updates the lockfile, and prunes the link tree. The store entry is **not** pruned (that's `pnpm store prune`).

## `pnpm update`

Updates packages within their semver range. Aliases: `up`, `upgrade`.

```bash
pnpm update                  # update everything within range
pnpm update react            # update react within its declared range
pnpm update --latest react   # jump to the latest version (ignores range, rewrites package.json)
pnpm update -L               # --latest for everything
pnpm update -i               # interactive; pick from a list
pnpm update -r typescript    # across all workspace packages
pnpm update -g               # global packages
pnpm update --no-save        # update node_modules but don't touch package.json
```

| Flag | Effect |
|------|--------|
| `--latest`, `-L` | Ignore the declared range; jump to latest, update `package.json`. |
| `--interactive`, `-i` | Show an interactive picker. |
| `--no-save` | Update `node_modules` but don't modify `package.json`/lockfile. |
| `--workspace` | Only update workspace packages. |
| `-r`, `-g`, `-P`, `-D`, `--no-optional` | As elsewhere. |

Without `--latest`, `pnpm update` respects the semver range in `package.json` and only moves to the newest version that satisfies it.

## `pnpm fetch`

Resolve + fetch only — no manifest read, no link. Designed for Docker multi-stage builds: copy the lockfile in, `pnpm fetch` to populate the store in one cached layer, then copy the rest of the source and `pnpm install --frozen-lockfile --offline`.

```dockerfile
COPY pnpm-lock.yaml ./
RUN pnpm fetch
COPY . .
RUN pnpm install --frozen-lockfile --offline
```

Flags: `--dev`/`-D`, `--prod`/`-P` to scope which packages to fetch.

## `pnpm prune`

Removes packages from `node_modules` that aren't referenced by the current install (e.g. after switching from `devDependencies` to only `dependencies`). Flags: `--prod`, `--no-optional`.

This prunes the project's `node_modules`, **not** the global store — use `pnpm store prune` for that.

## `pnpm dedupe`

Removes older versions of a dependency from the lockfile when a newer already-resolved version also satisfies the older range. Reduces duplicate copies in `node_modules/.pnpm`. Use `--check` to exit non-zero if deduplication would change anything (CI gate).

## Edge Cases and Gotchas

- **`--frozen-lockfile` vs `--lockfile-only`**: the former refuses to write the lockfile (CI safety); the latter writes it but skips linking (fast pre-commit). They are opposites in intent.
- **Platform-specific installs**: `--cpu`/`--os`/`--libc` install only the optional dependencies matching those platforms. Useful for Docker images, but `node_modules` will be incomplete for other platforms.
- **`--force` is heavy**: it re-fetches packages already in the store and rebuilds the link tree. Don't use it as a daily "try again" — `pnpm install --frozen-lockfile` after a `git clean -fdx node_modules` is usually enough.
- **Scripts and `--ignore-scripts`**: when ignored, native add-ons won't compile. Packages that depend on their install scripts at runtime will then fail at runtime, not at install.
- **`pnpm add` with `-g`** installs into the global virtual store (v11+). The old "single global `node_modules`" behavior is gone; see [store.md](store.md).

## References

- <https://pnpm.io/cli/install>
- <https://pnpm.io/cli/add>
- <https://pnpm.io/cli/remove>
- <https://pnpm.io/cli/update>
- <https://pnpm.io/cli/fetch>
- <https://pnpm.io/cli/prune>
- <https://pnpm.io/cli/dedupe>

## Related

- Resulting layout → [node-modules-layout.md](node-modules-layout.md)
- Lockfile written here → [lockfile.md](lockfile.md)
- Store used by fetch → [store.md](store.md)
- Hooks invoked during install → [hooks.md](hooks.md)
