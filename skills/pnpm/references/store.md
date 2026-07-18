# The Content-Addressable Store

pnpm's global store: where package files actually live, how it's addressed, where it's located, and the `pnpm store` commands. For how files get from the store into `node_modules`, see [node-modules-layout.md](node-modules-layout.md); for the install stage that populates the store, see [install.md](install.md).

> **Source of truth**: <https://pnpm.io/cli/store>, <https://pnpm.io/settings#store>, <https://pnpm.io/global-packages>. Verified against pnpm v10–v11.

## What the Store Is

A **content-addressable** directory: every file is stored once, addressed by a hash of its contents. When pnpm installs a package, it doesn't copy the tarball into your project — it fetches the tarball into the store, extracts it there, and **hard-links** (or clones, or copies) the files from the store into your project's `node_modules/.pnpm/<pkg>@<ver>/`.

Consequences:

- **One copy per file, system-wide.** If 20 projects on your machine use `lodash@4.17.21`, the store holds it once; each project hard-links to it.
- **Verifiable integrity.** The store is keyed by content hash; tampering is detectable (`pnpm store status`).
- **Fast installs.** Once a package is in the store, installing it elsewhere is just hard-linking — no network, no copy.

## Store Location (auto-detected)

pnpm resolves the store directory in this priority order:

1. **`$PNPM_HOME/store`** — if `PNPM_HOME` is set.
2. **`$XDG_DATA_HOME/pnpm/store`** — if `XDG_DATA_HOME` is set.
3. **Platform default:**
   - Linux: `~/.local/share/pnpm/store`
   - macOS: `~/Library/pnpm/store`
   - Windows: `~/AppData/Local/pnpm/store`
4. **Filesystem root** — if no home directory is on the target disk (e.g. a CI runner with no persistent home), the store is created at the filesystem root (`/mnt/.pnpm-store`, etc.).

You can override it explicitly:

```yaml
# pnpm-workspace.yaml
storeDir: /path/to/store
```

Or via env: `pnpm_config_store_dir=/path/to/store`.

### Why the store must be on the same filesystem as `node_modules`

Hard links don't cross filesystem boundaries. If the store is on a different filesystem than the project, pnpm falls back to copying (slower, uses more disk). For Docker, mount the store on the same volume as the workspace, or set `packageImportMethod: copy`.

## The Global Virtual Store (v11+)

Since v11, **global installs** (`pnpm add -g`, `pnpm dlx`) no longer use a single flat global `node_modules`. Instead they use a **global virtual store** at `{storeDir}/links`:

- Each globally-installed package is hard-linked into a directory named by the hash of its dependency graph.
- Identical dependency graphs are shared across all global installs and all `pnpm dlx` invocations on the machine.
- Each global package gets its own isolated `node_modules`, so global packages no longer conflict over peer dependency versions.

This is the same isolation model as project installs, applied globally.

## `pnpm store` Commands

### `pnpm store status`

Checks whether any package in the store has been modified since it was fetched. Exits non-zero if corruption is detected. Use in CI to catch accidental edits to the store.

```bash
pnpm store status
```

### `pnpm store add <pkg...>`

Fetches packages into the store without modifying any project. Useful for warming a CI/Docker store before the actual install.

```bash
pnpm store add react react-dom lodash@4.17.21
```

### `pnpm store prune`

Removes packages from the store that are no longer referenced by any project on the machine. Safe to run periodically; orphaned entries accumulate after dependency updates, branch switches, or project deletions.

```bash
pnpm store prune
```

Note: prune can only see projects whose `node_modules` currently exist. If you deleted a project's `node_modules` before pruning, its store entries won't be considered referenced and may be pruned (they'll just be re-fetched on next install).

### `pnpm store path`

Prints the active store directory.

```bash
pnpm store path
# /home/user/.local/share/pnpm/store
```

## Store-Related Settings

| Setting | Default | Effect |
|---------|---------|--------|
| `storeDir` | auto-detected (see above) | Store directory. |
| `verifyStoreIntegrity` | `true` | Verify file integrity against stored hashes on read. |
| `frozenStore` | `false` | If `true`, never write to the store; fail if a package is missing. Use in read-only CI environments. |
| `packageImportMethod` | `auto` | How files get from store to `node_modules`. See [node-modules-layout.md](node-modules-layout.md). |
| `enableGlobalVirtualStore` | `false` (default; `true` for global/dlx in v11) | Use the global virtual store for global installs. |

## Edge Cases and Gotchas

- **Cross-filesystem installs are slow.** If `storeDir` and your project are on different filesystems, pnpm copies instead of hard-links. Symptom: installs are much slower than expected and disk usage is higher. Fix: move the store onto the same filesystem, or accept `packageImportMethod: copy`.
- **Docker bind mounts.** A Docker bind mount of the host store into a container is usually a different filesystem from the container's working dir. Use a named volume, or `pnpm fetch` into a layer-local store, or set `packageImportMethod: copy`.
- **`pnpm store prune` and `node_modules`.** Prune considers a package referenced only if some project on the machine currently has it linked into `node_modules`. If you blow away all `node_modules` dirs, prune will empty the store. Re-install before pruning, or just re-fetch afterward — the store is a cache, not a source of truth.
- **Corruption after a crash.** If a fetch was interrupted, the store may have partial files. `pnpm install --force` re-fetches; `pnpm store status` detects it.
- **Disk usage.** The store grows monotonically until pruned. On active machines, prune monthly. The store is also the single biggest disk-space win of using pnpm — one `lodash` for every project.
- **Read-only CI store.** Set `frozenStore: true` in CI to guarantee the install never mutates the (shared, mounted) store. Combined with `--frozen-lockfile`, this makes CI installs reproducible and side-effect-free.

## References

- <https://pnpm.io/cli/store>
- <https://pnpm.io/settings#store>
- <https://pnpm.io/global-packages>

## Related

- Resulting layout → [node-modules-layout.md](node-modules-layout.md)
- Fetch stage → [install.md](install.md)
- Env vars affecting store location → [env-vars.md](env-vars.md)
