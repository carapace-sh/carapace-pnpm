# pnpm vs npm vs Yarn

How pnpm differs from npm, Yarn Classic, and Yarn Berry — by feature and by mental model. Use this to explain why pnpm behaves differently, or when migrating from another package manager. For pnpm-specific details, follow the cross-references.

> **Source of truth**: <https://pnpm.io/motivation>, <https://pnpm.io/feature-comparison>. Verified against pnpm v10–v11, npm v10, Yarn Classic v1, Yarn Berry v4.

## Mental Model

| Manager | Core idea |
|---------|-----------|
| **npm** | Flat `node_modules`. Copy every package into every project. Hoist to resolve conflicts. |
| **Yarn Classic** | Like npm, but with a lockfile first, workspaces, and a per-project cache. Flat `node_modules`. |
| **Yarn Berry** | Plug'n'Play (no `node_modules`) by default; zip-archived cache; constraints; `patch:` protocol. |
| **pnpm** | Content-addressable global store + symlinked virtual store per project. One copy of each file system-wide. Strict isolation by default. |

pnpm's bet: a symlinked layout with a content-addressable store is strictly better than flat `node_modules` — it saves disk, speeds up installs, and prevents phantom-dependency bugs — while remaining compatible with Node's module resolution algorithm. Yarn Berry makes a different bet (PnP, no `node_modules` at all), which is faster still but requires tooling support.

## Feature Comparison

| Feature | pnpm | npm | Yarn Classic | Yarn Berry |
|---------|------|-----|--------------|------------|
| **Default layout** | Isolated (symlinked virtual store) | Flat (hoisted) | Flat (hoisted) | PnP (no `node_modules`) |
| **Disk efficiency** | Content-addressable store; one copy per version system-wide | Copy per project | Copy per project | Zip cache; one copy per project |
| **Install speed** | Very fast (hard-link + symlink) | Slow (copy) | Fast (cache) | Fast (zip) |
| **Strict isolation** | Yes (only declared deps accessible) | No (all hoisted deps accessible) | No | Yes |
| **Phantom dependencies** | Prevented | Possible | Possible | Prevented |
| **Peer deps** | Auto-installed; isolated per peer set (multiple copies) | Auto-installed; hoisted | Auto-installed; hoisted | Auto-installed; isolated |
| **Lockfile** | `pnpm-lock.yaml` (YAML) | `package-lock.json` (JSON) | `yarn.lock` (custom) | `yarn.lock` (YAML) + `.pnp.cjs` |
| **Workspace lockfile** | Shared by default (`sharedWorkspaceLockfile`) | Per-package | Shared | Shared |
| **Workspaces config** | `pnpm-workspace.yaml` | `workspaces` in `package.json` | `workspaces` in `package.json` | `workspaces` in `package.json` + `.yarnrc.yml` |
| **Config file** | `pnpm-workspace.yaml` (v11; was `.npmrc`) | `.npmrc` (INI) | `.yarnrc` (INI) | `.yarnrc.yml` (YAML) |
| **Auth file** | `.npmrc` / `auth.ini` (v11) | `.npmrc` | `.yarnrc` / `.npmrc` | `.yarnrc.yml` |
| **Overrides** | `overrides` in `pnpm-workspace.yaml` | `overrides` in `package.json` | `resolutions` in `package.json` | `resolutions` in `package.json` |
| **Catalogs / version constants** | Yes (`catalog:`) | No | No | Yes (constraints) |
| **Patching** | `pnpm patch` + `patch-commit` | No native | No native | `patch:` protocol |
| **Hooks** | `.pnpmfile.mjs` (readPackage, afterAllResolved, custom resolvers/fetchers) | No | No | Plugin system |
| **Plug'n'Play** | Optional (`nodeLinker: pnp`) | No | No | Default |
| **Run command** | `pnpm run` / `pnpm exec` / `pnpm dlx` | `npm run` / `npx` | `yarn run` / `yarn` | `yarn run` / `yarn dlx` |
| **One-off packages** | `pnpm dlx` (alias `pnpx`) | `npx` | `yarn dlx` | `yarn dlx` |
| **Global installs** | Isolated per-package (v11 global virtual store) | Single global `node_modules` | Single global `node_modules` | Not native |
| **Env var config** | `pnpm_config_*` (v11) | `npm_config_*` | `yarn_*` | `yarn_*` |
| **Runtime management** | `pnpm runtime set node\|deno\|bun` | External (`nvm`/`fnm`) | No | `yarn set version` (Yarn itself) |
| **Publish** | Native (v11) | Native | Native | Native |
| **Security: min release age** | `minimumReleaseAge` | No | No | No |
| **Security: trusted deps** | `allowBuilds` (v11; was `onlyBuiltDependencies` in v10) + `strictDepBuilds` | No (all scripts run) | No | No |
| **Trusted builds** | `--allow-build` (v10+) | — | — | — |
| **Multi-platform installs** | `--cpu`/`--os`/`--libc` | No | No | No |

## Key Differentiators

### 1. The content-addressable store

pnpm's single most differentiating feature. The store holds each file once, addressed by hash. Projects hard-link from it. On a machine with many JS projects, this saves gigabytes. No other mainstream package manager does this (Yarn Berry's zip cache is per-project, not system-wide).

### 2. Symlinked, isolated `node_modules`

The default layout makes only declared dependencies accessible. This prevents phantom-dependency bugs (a file `require`s a package it didn't declare, works by accident because a sibling hoisted it, then breaks when the sibling is removed). npm and Yarn Classic's flat layouts allow this. Yarn Berry's PnP also prevents it, but by removing `node_modules` entirely.

### 3. Peer dependency isolation

pnpm creates a separate instance of a package for each distinct set of peers it's resolved against. This is more correct (each consumer sees the peers it expects) at the cost of more directory entries in `node_modules/.pnpm` (they're hard links, cheap on disk). npm/Yarn Classic hoist peers, which is "one copy" but can be wrong. See [peer-dependencies.md](peer-dependencies.md).

### 4. Catalogs

A workspace-wide registry of version ranges, referenced from `package.json` as `catalog:`. Upgrade a dep everywhere by editing one line. Unique to pnpm (Yarn Berry's constraints are similar in spirit but different in design). See [catalogs.md](catalogs.md).

### 5. The v11 config split

Settings moved from `.npmrc` to `pnpm-workspace.yaml` (YAML, supporting nested structures for `overrides`/`catalogs`/`packageExtensions`). Auth stayed in `.npmrc`/`auth.ini`. `npm_config_*` env vars are no longer read for pnpm's own config; use `pnpm_config_*`. See [config.md](config.md).

### 6. Trusted dependencies (v10+)

By default, pnpm **does not run install scripts** for dependencies. You opt in per-package (`--allow-build`, or the `allowBuilds` map in `pnpm-workspace.yaml` since v11 — replacing the v10 `onlyBuiltDependencies` list). `strictDepBuilds` defaults to `true`, so unlisted build scripts hard-fail the install. npm and Yarn run all install scripts by default — a significant supply-chain attack surface. See [config.md](config.md).

### 7. Supply-chain controls

`minimumReleaseAge` (block versions newer than N minutes), `trustPolicy: no-downgrade` (block downgrades of trusted packages), `blockExoticSubdeps` (only direct deps can use git/file/url sources). These are pnpm-specific.

## Migration Notes

### From npm

- `npm install` → `pnpm install` (mostly drop-in).
- `npm install -D` → `pnpm add -D`.
- `npx` → `pnpm dlx` (or `pnpm exec` if already installed).
- `npm run` → `pnpm run` (or bare `pnpm <script>`).
- `package-lock.json` → delete it; `pnpm install` writes `pnpm-lock.yaml`.
- `overrides` in `package.json` → `overrides` in `pnpm-workspace.yaml`.
- Phantom deps may now fail. Add them to `package.json` (correct fix) or `shamefullyHoist: true` (escape hatch). See [node-modules-layout.md](node-modules-layout.md).
- `npm_config_*` env vars: rename to `pnpm_config_*` (v11).

### From Yarn Classic

- `yarn` → `pnpm install`.
- `yarn add` → `pnpm add`.
- `yarn run` → `pnpm run`.
- `yarn dlx` → `pnpm dlx`.
- `resolutions` in `package.json` → `overrides` in `pnpm-workspace.yaml`.
- `yarn.lock` → delete; `pnpm install` writes `pnpm-lock.yaml`.
- `workspaces` in `package.json` → `packages` in `pnpm-workspace.yaml`.
- Same phantom-deps caveat as npm.

### From Yarn Berry

- `yarn install` → `pnpm install`.
- `yarn dlx` → `pnpm dlx`.
- `resolutions` → `overrides`.
- `patch:` protocol → `pnpm patch` / `patch-commit` (see [overrides-patch.md](overrides-patch.md)).
- `.yarnrc.yml` → `pnpm-workspace.yaml` (see [config.md](config.md)).
- If you relied on PnP, set `nodeLinker: pnp` (see [node-modules-layout.md](node-modules-layout.md)); otherwise pnpm's isolated layout is the default and you get a real `node_modules` back.
- Yarn Berry's constraints → pnpm's `catalogs` (different model; see [catalogs.md](catalogs.md)).

## When to Pick pnpm

- **Disk space matters** (many projects on one machine; CI with limited cache).
- **You want strict dependency isolation** (catch phantom deps before they ship).
- **You have a monorepo** and want catalogs, shared lockfile, topological recursive runs.
- **You want supply-chain controls** (min release age, trusted deps, no-downgrade).
- **You want a real `node_modules`** (Yarn Berry's PnP breaks some tooling; pnpm's isolated layout works with everything that works with npm).

## When to Pick Something Else

- **Yarn Berry** if you're committed to PnP and the zip-cache model and your tooling supports it.
- **npm** if you want the lowest-friction, official option and don't need pnpm's strictness or disk savings.
- **Yarn Classic** — legacy; migrate to pnpm or Yarn Berry.

## References

- <https://pnpm.io/motivation>
- <https://pnpm.io/feature-comparison>
- <https://pnpm.io/symlinked-node-modules-structure>

## Related

- The layout → [node-modules-layout.md](node-modules-layout.md)
- The store → [store.md](store.md)
- Config differences → [config.md](config.md)
- Peer isolation → [peer-dependencies.md](peer-dependencies.md)
