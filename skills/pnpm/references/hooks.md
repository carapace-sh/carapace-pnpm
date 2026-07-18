# Hooks — `.pnpmfile.mjs` / `.pnpmfile.cjs`

The pnpmfile hook system: mutating packages at install time, custom resolvers/fetchers, and finders. For the settings that control the pnpmfile, see [config.md](config.md); for the install pipeline that invokes hooks, see [install.md](install.md).

> **Source of truth**: <https://pnpm.io/pnpmfile>. Verified against pnpm v10–v11.

## The Pnpmfile

A file named `.pnpmfile.mjs` (ESM) or `.pnpmfile.cjs` (CommonJS) placed next to the lockfile (workspace root). It exports a `hooks` object (and optionally `resolvers`, `fetchers`, `finders`).

```javascript
// .pnpmfile.mjs
export default {
  hooks: {
    readPackage(pkg, context) {
      // mutate pkg, return it
      return pkg
    },
  },
}
```

```javascript
// .pnpmfile.cjs
module.exports = {
  hooks: {
    readPackage(pkg, context) {
      return pkg
    },
  },
}
```

The file is loaded once per install. It runs in Node, not in a sandbox — it can `require`/`import` anything in the project.

## Lifecycle Hooks

### `hooks.readPackage(pkg, context)`

Called for **every** package manifest pnpm parses (your own, workspace packages, and every registry dep), before resolution. Return the (possibly mutated) manifest.

Use cases:

- Add a missing dependency a package forgot to declare.
- Remove a dependency that breaks your build.
- Patch a package's `bin` field.
- Force a package to use a different version of its dep.

```javascript
hooks: {
  readPackage(pkg, context) {
    if (pkg.name === 'some-broken-pkg') {
      pkg.dependencies['missing-dep'] = '^1.0.0'
    }
    // Strip postinstall from a package whose script is broken
    if (pkg.name === 'pkg-with-bad-postinstall') {
      delete pkg.scripts?.postinstall
    }
    return pkg
  },
}
```

The `context` has the lockfile and resolved-graph state; rarely needed for simple mutations.

### `hooks.updateConfig(config)`

Since v10.8. Modify pnpm's config at install time. Use to compute settings programmatically (e.g. branch-aware overrides).

```javascript
hooks: {
  updateConfig(config) {
    if (process.env.BRANCH === 'main') {
      config.overrides = { ...config.overrides, 'some-pkg': '2.0.0' }
    }
    return config
  },
}
```

### `hooks.afterAllResolved(lockfile, context)`

Called after resolution, with the in-memory lockfile object, before it's serialized to `pnpm-lock.yaml`. Mutate it and return it.

Use cases:

- Add custom fields to package entries.
- Remove packages from the lockfile.
- Programmatically rewrite resolutions.

```javascript
hooks: {
  afterAllResolved(lockfile) {
    // example: log the package count
    console.log(Object.keys(lockfile.packages).length)
    return lockfile
  },
}
```

Rare in practice — most needs are covered by `readPackage` + `overrides`.

### `hooks.beforePacking(pkg)`

Since v10.28. Called before `pnpm pack` / `pnpm publish` produces the tarball. Mutate the manifest that will be packed (e.g. strip internal scripts, rewrite `bin`).

```javascript
hooks: {
  beforePacking(pkg) {
    delete pkg.scripts?.dev
    delete pkg.scripts?.test
  },
}
```

See [publish-pack.md](publish-pack.md).

### `hooks.preResolution(options)`

Called after lockfiles are read, before dependency resolution. Read-only-ish: use to fail fast on unsupported environments, or to inspect what's about to be resolved.

### `hooks.importPackage(destinationDir, options)`

Customize how a package's files are written from the store into `node_modules`. The default behavior is hard-link (or copy per `packageImportMethod`). Override to, e.g., post-process files after linking.

Returns the final layout. Advanced; rarely needed.

## Custom Resolvers and Fetchers (v11+)

Top-level exports (not under `hooks`):

```javascript
// .pnpmfile.cjs
module.exports = {
  resolvers: [myResolver],
  fetchers: [myFetcher],
}

const myResolver = {
  canResolve(wantedDependency) {
    return wantedDependency.name?.startsWith('internal:')
  },
  resolve(wantedDependency, opts) {
    return {
      resolution: { type: 'internal', id: wantedDependency.name.slice(9) },
      pkgId: wantedDependency.name,
    }
  },
  shouldRefreshResolution(depPath, pkgSnapshot) {
    return false
  },
}

const myFetcher = {
  canFetch(pkgId, resolution) {
    return resolution.type === 'internal'
  },
  async fetch(cafs, resolution, opts, fetchers) {
    // return a manifest + files to the cafs (content-addressable file store)
    return { manifest: {...}, filesIndex: {...} }
  },
}
```

Use to implement private protocols (`internal:...`, `git+ssh` variants, company-internal registries with custom layouts). A resolver decides whether it can handle a `wantedDependency` and produces a `resolution`; a fetcher turns a `resolution` into actual files in the store.

## Finders (v10.16+)

Named predicates for `pnpm list --find-by <name>` / `pnpm why --find-by <name>`:

```javascript
// .pnpmfile.mjs
export const finders = {
  react17: (ctx) => {
    const manifest = ctx.readManifest()
    return manifest.peerDependencies?.react === '^17.0.0'
  },
  hasScripts: (ctx) => Object.keys(ctx.readManifest().scripts ?? {}).length > 0,
}
```

Then: `pnpm list --find-by react17` lists every package whose peer deps match `react@^17`. `ctx` exposes the manifest and the resolved graph.

## Pnpmfile Settings

| Setting | Default | Effect |
|---------|---------|--------|
| `pnpmfile` | `['.pnpmfile.mjs']` | Path(s) to pnpmfile(s). Can be an array to load multiple. |
| `ignorePnpmfile` | `false` | Skip the pnpmfile on this install (`pnpm install --ignore-pnpmfile`). |
| `globalPnpmfile` | `null` | Path to a pnpmfile applied to every project on the machine. |

## Edge Cases and Gotchas

- **`readPackage` is called for every package, including yours.** Guard with `if (pkg.name === '...')` or you'll mutate things you didn't intend.
- **Mutating the manifest is not the same as overriding.** `readPackage` changes what the package *declares*; `overrides` (see [overrides-patch.md](overrides-patch.md)) changes what pnpm *resolves*. For "force version X of dep Y everywhere," prefer `overrides`.
- **`afterAllResolved` runs every install.** Expensive work here slows every install. Cache if needed.
- **`--ignore-pnpmfile` in CI.** If your CI uses `--frozen-lockfile`, the pnpmfile's `readPackage`/`afterAllResolved` still run (they're part of resolution). Use `--ignore-pnpmfile` if you want to skip them — but then any mutation they perform won't happen, and the install may diverge from what the lockfile expects.
- **Pnpmfile must be valid for the current pnpm version.** The hook signatures and the lockfile object shape are not part of a stable public API — they evolve. Check the docs when upgrading pnpm.
- **ESM vs CJS.** `.pnpmfile.mjs` is ESM (use `export default`); `.pnpmfile.cjs` is CommonJS (use `module.exports`). Pick based on your project's `"type"` in `package.json`.

## References

- <https://pnpm.io/pnpmfile>

## Related

- Install pipeline that runs hooks → [install.md](install.md)
- Pnpmfile settings → [config.md](config.md)
- `beforePacking` and publish → [publish-pack.md](publish-pack.md)
- Overrides as an alternative to `readPackage` → [overrides-patch.md](overrides-patch.md)
