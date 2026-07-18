# pnpm-lock.yaml

The lockfile format: top-level sections, what each field means, and how `lockfileVersion` 9.0 differs from earlier versions. For the pipeline that writes this file, see [install.md](install.md); for the workspace config that feeds it, see [workspace.md](workspace.md).

> **Source of truth**: <https://github.com/pnpm/spec/blob/master/lockfile/9.0.md> (lockfileVersion 9.0, "ComVer"). Verified against pnpm v9–v11.

## Purpose

`pnpm-lock.yaml` captures the fully-resolved dependency graph: which exact version of every package is installed, from where, with which peers, and what each workspace project declared. It is the input to `pnpm install --frozen-lockfile` and the artifact you commit for reproducible installs.

## Top-Level Sections

```yaml
lockfileVersion: '9.0'

settings:
  autoInstallPeers: true
  excludeLinksFromLockfile: false

catalogs:
  react17:
    react: 17.0.2

importers:
  .:
    dependencies:
      react:
        specifier: ^18.2.0
        version: 18.2.0
    devDependencies:
      typescript:
        specifier: ^5.0.0
        version: 5.4.5
  packages/widget:
    dependencies:
      react:
        specifier: catalog:react17
        version: 17.0.2

packages:
  react@18.2.0:
    resolution: {integrity: sha512-...=}
    engines: {node: '>=0.10.0'}
    peerDependencies:
      react-dom: ^18.2.0
    hasBin: false

snapshots:
  react@18.2.0:
    dependencies:
      loose-envify: 1.4.0
  react@17.0.2:
    dependencies:
      loose-envify: 1.4.0
      object-assign: 4.1.1
```

### `lockfileVersion`

A string like `'9.0'`. pnpm refuses to install from a lockfile with a newer major version than it understands; it will migrate older lockfiles on write. The major version bumps when the on-disk format changes incompatibly.

| Version | pnpm range | Notes |
|---------|-----------|-------|
| `9.0` | pnpm v9+ | Current. Separated `packages` (manifest info) from `snapshots` (resolved graph). |
| `6.0` | pnpm v8 | Single `packages` map holding both manifest and resolved deps; keys include integrity hashes. |
| `5.x` | pnpm v6–v7 | Older format. |

### `settings`

Auto-detected settings that affect resolution. pnpm writes these so that a subsequent `--frozen-lockfile` install can verify the lockfile was generated with compatible settings. Includes `autoInstallPeers`, `excludeLinksFromLockfile`, and other resolution-affecting flags.

### `catalogs`

Catalog definitions used by `catalog:` specifiers (see [catalogs.md](catalogs.md)). Only present when catalogs are defined in `pnpm-workspace.yaml`.

### `importers`

Maps the **relative path** of each workspace project (the root is `.`) to its declared dependencies. Each entry has `dependencies`, `devDependencies`, `optionalDependencies`, `peerDependencies` — each a map of name → `{ specifier, version }`:

- **`specifier`** — the exact range string from that project's `package.json` (e.g. `^18.2.0`, `catalog:`, `workspace:*`, `file:./local`).
- **`version`** — the resolved version, or a path/URL for non-registry deps. For `catalog:` specifiers this is the concrete version the catalog resolved to.

This is the per-project view: which version each project *asked for* and which it *got*.

### `packages`

Maps a **dependency id** of the form `<name>@<version>` to that package's manifest-derived metadata:

| Field | Meaning |
|-------|---------|
| `resolution` | `{integrity: sha512-..., tarball?: <url>}` — how to verify/fetch the tarball. |
| `engines` | `{node: '>=14', ...}` — engine requirements. |
| `peerDependencies` | Declared peer deps and their ranges. |
| `peerDependenciesMeta` | `{<name>: {optional: true}}` — optional peer markers. |
| `dependencies` / `optionalDependencies` | Declared deps with their ranges. |
| `hasBin` | Whether the package installs binaries. |
| `bin` | The bin map. |
| `deprecated` | Deprecation message, if any. |
| `os` / `cpu` / `libc` | Platform constraints. |
| `patchedDependencies` | References to applied patches (see [overrides-patch.md](overrides-patch.md)). |

No resolved peer context here — `packages` is the "what is this package" view, independent of who uses it.

### `snapshots`

Maps a **dependency path** (the id plus peer context) to the **resolved** versions of its dependencies:

```yaml
snapshots:
  react-dom@18.2.0(react@18.2.0):
    dependencies:
      loose-envify: 1.4.0
      scheduler: 0.23.0
    transitivePeerDependencies:
      - react
```

The key `react-dom@18.2.0(react@18.2.0)` is `<pkg>@<ver>` plus a peer suffix `(peer@ver[, peer2@ver2, ...])` describing which peers this instance was resolved against. **Each distinct peer set produces a separate snapshot** — this is how pnpm records peer dependency isolation (see [peer-dependencies.md](peer-dependencies.md)).

Snapshot fields:

| Field | Meaning |
|-------|---------|
| `dependencies` | Map of dep name → resolved exact version. |
| `optionalDependencies` | Same, for optional deps. |
| `transitivePeerDependencies` | List of peer deps that this package's transitive deps declare but this package doesn't directly satisfy. Informational. |

## How to Read a Lockfile Entry

To find out what `foo@1.2.3` actually is and what it depends on:

1. Look up `packages.foo@1.2.3` for the manifest view (resolution, declared ranges, peers).
2. Look up `snapshots.foo@1.2.3` (or `foo@1.2.3(<peers>)` for a specific peer set) for the resolved versions of its dependencies.

The split exists so that one `packages` entry can serve many peer contexts — `foo@1.2.3` is one package, but `foo@1.2.3(react@17)` and `foo@1.2.3(react@18)` are two snapshots with potentially different resolved sub-dependencies.

## Lockfile-Related Settings and Flags

| Setting / flag | Default | Effect |
|----------------|---------|--------|
| `lockfile` | `true` | Read/write `pnpm-lock.yaml`. `false` disables it. |
| `--frozen-lockfile` | off | Don't modify the lockfile; fail if it's out of sync with manifests. CI. |
| `--lockfile-only` | off | Update only the lockfile; don't touch `node_modules`. |
| `--no-lockfile` | off | Ignore the lockfile entirely for this install. |
| `--fix-lockfile` | off | Repair a broken lockfile by re-resolving. |
| `--update-checksums` | off | Recompute integrity hashes (after manual store edits). |
| `preferFrozenLockfile` | `true` | In a workspace, prefer the frozen lockfile when present. |
| `lockfileIncludeTarballUrl` | `false` | Include the full tarball URL in `resolution` (default is integrity hash only). |
| `gitBranchLockfile` | `false` | Write branch-specific lockfiles (e.g. `pnpm-lock.yaml` per git branch) to reduce merge conflicts. |
| `mergeGitBranchLockfilesBranchPattern` | — | When to auto-merge branch lockfiles on install. |
| `sharedWorkspaceLockfile` | `true` | One lockfile at the workspace root (not per project). See [workspace.md](workspace.md). |

## Edge Cases and Gotchas

- **Merge conflicts.** v9's split of `packages` and `snapshots` reduces them (snapshot keys are the only peer-suffixed ones), but a re-resolve after a merge is the cleanest fix: `pnpm install --fix-lockfile`.
- **`--frozen-lockfile` and a dirty lockfile.** pnpm compares the lockfile against current manifests and *settings*; changing a resolution-affecting setting (e.g. `autoInstallPeers`) without regenerating the lockfile will fail frozen installs. Regenerate, commit, then CI.
- **`catalog:` in the lockfile.** The `importers` section records `specifier: catalog:react17` and `version: <concrete>`. If you change a catalog range, the lockfile's `version` for that importer must update — `pnpm install` does this; `--frozen-lockfile` will fail until you do.
- **`workspace:` protocol.** Recorded in `specifier`; `version` is the resolved workspace package's version (or `link:../path` for linked packages). See [workspace.md](workspace.md).
- **Patches.** A patched package's `packages` entry references the patch file under `patchedDependencies` at the top level (and in `pnpm-workspace.yaml`). See [overrides-patch.md](overrides-patch.md).
- **Don't hand-edit.** The format is stable and readable but generated. Use `pnpm install --fix-lockfile` or `--lockfile-only` to make changes.

## References

- Lockfile 9.0 spec: <https://github.com/pnpm/spec/blob/master/lockfile/9.0.md>
- <https://pnpm.io/cli/install#--frozen-lockfile>
- <https://pnpm.io/settings#lockfile>

## Related

- Pipeline that writes it → [install.md](install.md)
- Workspace config that feeds it → [workspace.md](workspace.md)
- Peer suffixes in snapshot keys → [peer-dependencies.md](peer-dependencies.md)
- Patch entries → [overrides-patch.md](overrides-patch.md)
