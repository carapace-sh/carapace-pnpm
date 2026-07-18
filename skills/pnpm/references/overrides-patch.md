# Overrides, Patches, and Package Extensions

How to force dependency versions (`overrides`), patch package source (`pnpm patch`), and extend package manifests (`packageExtensions`). These are the three mechanisms for changing what the registry gives you. For the pnpmfile alternative, see [hooks.md](hooks.md); for the lockfile entries these produce, see [lockfile.md](lockfile.md).

> **Source of truth**: <https://pnpm.io/settings#overrides>, <https://pnpm.io/cli/patch>, <https://pnpm.io/package_extensions>. Verified against pnpm v10–v11.

## Overrides

Force specific dependency versions across the entire dependency graph. Defined in `pnpm-workspace.yaml`:

```yaml
overrides:
  # Every "foo" in the graph resolves to ^1.0.0
  foo: ^1.0.0

  # Replace foo with a fork
  quux: npm:@myorg/quux@^1.0.0

  # Only override foo when it's required at ^2.1.0
  bar@^2.1.0: 3.0.0

  # Nested: override zoo only inside qar@1
  qar@1>zoo: 2

  # Remove a dependency from a package's manifest
  foo@1.0.0>bar: -

  # Override a peer dependency of react-dom
  react-dom>react: 18.1.0
```

### Selector syntax

| Selector | Matches |
|----------|---------|
| `<name>` | Every instance of `<name>` anywhere in the graph. |
| `<name>@<range>` | Only instances whose required range matches `<range>`. |
| `<parent>@<prange>><name>` | `<name>` only when required by `<parent>` (optionally filtered by `<parent>`'s range `<prange>`). |
| `<parent>><name>: -` | Remove `<name>` from `<parent>`'s deps entirely. |
| `npm:<other>@<range>` | Replace the matched package with `<other>` (an alias/fork). |

### What overrides do vs. don't

- **Do**: change the resolved version of a package, anywhere in the graph, regardless of who requires it.
- **Do not**: add a package that nothing requires (use `pnpm add` for that).
- **Do not**: change a package's own files (use `patch` for that).
- **Do not**: add a field to a package's manifest (use `packageExtensions` for that).

### Overrides vs. `readPackage` vs. `resolutions`

| Mechanism | Scope | Where defined |
|-----------|-------|---------------|
| `overrides` | Whole graph | `pnpm-workspace.yaml` |
| `hooks.readPackage` | Per-package manifest mutation | `.pnpmfile.mjs` |
| `resolutions` (yarn) | Whole graph | `package.json` (yarn convention) |

Prefer `overrides` — it's declarative, lives in config, and is recorded in the lockfile. Use `readPackage` only when you need to mutate non-version manifest fields (e.g. delete a script).

## `packageExtensions`

Some packages under-declare their dependencies (a common npm ecosystem bug: a package uses a dep but doesn't list it, relying on hoisting). `packageExtensions` adds fields to a package's manifest as if it had declared them:

```yaml
packageExtensions:
  react-redux:
    peerDependencies:
      react-dom: '*'
  some-loader:
    optionalDependencies:
      '@types/node': '*'
    dependencies:
      'missing-but-used': '^1.0.0'
```

You can extend `dependencies`, `optionalDependencies`, `peerDependencies`, `peerDependenciesMeta`. The extension is applied to **every** version of the named package.

Use this when:

- A package is missing a peer/dep that you know it needs (avoids "Cannot find module" under strict isolation).
- You want to suppress a missing-peer warning by declaring the peer on the package's behalf.

## `allowedDeprecatedVersions`

Suppress deprecation warnings for specific versions:

```yaml
allowedDeprecatedVersions:
  request: '2.88.0'      # don't warn that request@2.88.0 is deprecated
  lodash: '*'
```

Use sparingly — deprecation warnings exist for a reason. Useful when a transitive dep pins a deprecated version you can't immediately upgrade.

## Patching (`pnpm patch`)

Modify a dependency's source code. The patch is a `.patch` file applied at install time; the patched files are hard-linked into `node_modules` like any other.

### Workflow

```bash
# 1. Extract the package into a temp dir for editing
pnpm patch express@4.18.1
# pnpm prints the temp dir path, e.g.:
#   You can now edit the following folder: /tmp/xxxx/express

# 2. Edit files in that directory
#    (cd /tmp/xxxx/express && $EDITOR index.js)

# 3. Generate the patch file and register it
pnpm patch-commit /tmp/xxxx/express
```

`pnpm patch-commit` writes `patches/express@4.18.1.patch` (by default under `patches/`) and adds an entry to `patchedDependencies` in `pnpm-workspace.yaml`:

```yaml
patchedDependencies:
  express@4.18.1: patches/express@4.18.1.patch
```

### `pnpm patch` flags

| Flag | Effect |
|------|--------|
| `--edit-dir <path>` | Use a specific directory for the edit workspace instead of a temp dir. |
| `--ignore-existing` | Don't reuse an existing edit dir for the same package. |

### Registration forms

```yaml
patchedDependencies:
  express@4.18.1: patches/express@4.18.1.patch   # exact version
  foo@^2.0.0: patches/foo-2.patch                 # any matching range
  foo: patches/foo-1.patch                        # catch-all (any version)
```

### `allowUnusedPatches`

```yaml
allowUnusedPatches: false   # default; fail install if a registered patch's package is absent
```

Set `true` to silently skip patches whose target package isn't in the graph (e.g. after removing a dep that had a patch).

### What patching does at install time

1. After fetching the package into the store, pnpm applies each registered patch to a per-instance copy.
2. The patched files are what get hard-linked into `node_modules/.pnpm/<pkg>@<ver>/`.
3. The patch is part of the lockfile's `packages` entry, so the same patch is applied reproducibly from the lockfile.

### Patch vs. fork vs. overrides

| Need | Mechanism |
|------|-----------|
| Change a resolved version | `overrides` |
| Replace a package with your fork | `overrides` with `npm:<fork>@<ver>` |
| Edit a package's source code | `pnpm patch` |
| Add a missing dep/peer to a package | `packageExtensions` |
| Mutate a package's manifest (e.g. delete a script) | `.pnpmfile.mjs` `readPackage` |

Patches are the only way to change a package's actual code without publishing a fork.

## Edge Cases and Gotchas

- **Overrides affect the whole graph, including transitive deps you don't see.** `overrides: { foo: ^2 }` upgrades every `foo` to `^2`, even ones pulled by a transitive that declared `foo: ^1`. This can break the transitive if `foo@2` removed an API it used. Use the `@<range>` or `parent>name` selector forms to scope narrowly.
- **Removing a dep with `-`.** `foo@1.0.0>bar: -` deletes `bar` from `foo@1.0.0`'s manifest. If `foo`'s code actually requires `bar` at runtime, it will crash. Use only when you know `bar` is dead code.
- **Patch files are diffs.** They break when the upstream version changes its file layout. Pin the patched version (`express@4.18.1`, not `express@^4`) for stability. Re-run `pnpm patch` after upgrades.
- **`pnpm patch-commit` rewrites `pnpm-workspace.yaml`.** Don't edit the file concurrently. The patch file path is relative to the workspace root.
- **`packageExtensions` applies to all versions of a package.** If a future version fixes the missing dep, your extension becomes a no-op (safe) but still listed. Review periodically.
- **Patches and the store.** Patched files live in the virtual store, not the global content-addressable store (the patch makes them distinct from the upstream tarball's files). Disk savings are slightly reduced for patched packages.

## References

- Overrides: <https://pnpm.io/settings#overrides>
- Patching: <https://pnpm.io/cli/patch>
- Package extensions: <https://pnpm.io/package_extensions>

## Related

- Pnpmfile alternative → [hooks.md](hooks.md)
- Lockfile entries for patches/overrides → [lockfile.md](lockfile.md)
- Workspace file (where these live) → [workspace.md](workspace.md)
