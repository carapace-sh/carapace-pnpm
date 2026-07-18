# The node_modules Layout

How pnpm arranges `node_modules`: the isolated (symlinked virtual store) layout, the hoisted layout, and the Plug'n'Play layout. This is the single most distinctive pnpm design decision. For the store that feeds the layout, see [store.md](store.md); for peer-driven multiple copies, see [peer-dependencies.md](peer-dependencies.md).

> **Source of truth**: <https://pnpm.io/symlinked-node-modules-structure>, <https://pnpm.io/settings#node_modules_layout>. Verified against pnpm v10–v11.

## The Three Layouts

pnpm supports three `node_modules` layouts, selected by the `nodeLinker` setting (or `--node-linker` flag):

| `nodeLinker` | Layout | When to use |
|--------------|--------|-------------|
| `isolated` (default) | Symlinked virtual store. Strict isolation. | Default; works for almost all projects. |
| `hoisted` | Flat `node_modules` like npm/Yarn Classic. | React Native, serverless bundles, tools that don't follow symlinks. |
| `pnp` | Yarn Berry-style Plug'n'Play. No `node_modules`. | When you want PnP (also set `symlink: false`). |

## Isolated Layout (default)

```
node_modules/
├── foo -> ./.pnpm/foo@1.0.0/node_modules/foo        # symlink: direct dep
├── bar -> ./.pnpm/bar@1.0.0/node_modules/bar        # symlink: direct dep
└── .pnpm/                                            # the virtual store
    ├── foo@1.0.0/
    │   └── node_modules/
    │       ├── foo/        # hard-linked from the global store
    │       └── bar -> ../../bar@1.0.0/node_modules/bar   # symlink to dep
    ├── bar@1.0.0/
    │   └── node_modules/
    │       └── bar/        # hard-linked from the global store
    └── node_modules/       # created by hoisting (see below)
```

### Properties

- **Root `node_modules`** contains only the project's **direct** dependencies, as **symlinks** into `node_modules/.pnpm/<pkg>@<ver>/node_modules/<pkg>`.
- **`node_modules/.pnpm/`** (the **virtual store**) is where files actually live. Each `foo@1.0.0/node_modules/foo/` is hard-linked from the global content-addressable store.
- **Per-package isolation**: `foo@1.0.0/node_modules/` contains symlinks to `foo`'s declared dependencies only. `foo` cannot see packages it doesn't declare — no phantom dependencies.
- **Node.js compatibility**: Node's resolution algorithm follows the **real path** of a symlinked module, so `foo`'s `require('bar')` resolves inside `foo@1.0.0/node_modules/`, finding only `foo`'s own `bar`. This is the key trick that makes isolation work without a custom resolver.

### Why this saves disk space

- The same version of a package is hard-linked from the store **once per project** (the virtual store), not copied once per place it's used.
- Across projects on the same machine, the store holds each file once. Hard links dedupe everything.

### Why this prevents bugs

- **No phantom dependencies**: a file that `require`s a package not in its own `package.json` will fail, even if a sibling package installed it. npm's flat layout would let it succeed — until that sibling is removed.
- **No version mismatches across the tree**: each package sees exactly the version it declared (or the one resolved for its peer set).

## Hoisting

By default pnpm still **hoists** dependencies — to `node_modules/.pnpm/node_modules/`, **not** the root. This is the `hoist: true` / `hoistPattern: ['*']` default. It exists so packages with undeclared dependencies on common transitive packages (a frequent bug in the npm ecosystem) can still resolve them — but only via the virtual store's hoisted bucket, not from the application's root.

### Hoisting settings

| Setting | Default | Effect |
|---------|---------|--------|
| `hoist` | `true` | Enable hoisting to `node_modules/.pnpm/node_modules`. |
| `hoistPattern` | `['*']` | Which packages to hoist to `.pnpm/node_modules`. Glob list. |
| `publicHoistPattern` | `[]` | Which packages to hoist to the **root** `node_modules` (accessible to app code). Empty by default = strict. |
| `shamefullyHoist` | `false` | Shorthand for `publicHoistPattern: ['*']` — flat layout like npm. |
| `hoistingLimits` | `none` | Controls hoisting depth when `nodeLinker: hoisted`. Values: `none`, `workspaces`, `dependencies`. |
| `hoistWorkspacePackages` | `true` | Symlink workspace packages to the hoisted location too. |

### `shamefullyHoist` / `--shamefully-hoist`

When you can't fix phantom dependencies upstream, `shamefullyHoist: true` (or `pnpm install --shamefully-hoist`) makes the root `node_modules` flat like npm's. This defeats pnpm's strictness guarantee but is a pragmatic escape hatch for:

- React Native (Metro bundler and many RN packages assume a flat layout).
- Serverless bundlers that don't follow symlinks.
- Legacy codebases riddled with undeclared imports.

Prefer `publicHoistPattern` with a specific glob (`publicHoistPattern: ['*eslint*', '*types*']`) over `shamefullyHoist: true` — it keeps the strictness where you can.

## Hoisted Layout (`nodeLinker: hoisted`)

A flat `node_modules` using Yarn's hoisting algorithm. All transitive dependencies are hoisted to the root.

```yaml
# pnpm-workspace.yaml
nodeLinker: hoisted
```

Use when:

- A tool fundamentally breaks on symlinks (some bundlers, some native add-on toolchains).
- You're migrating from npm/Yarn Classic and need a stopgap.
- A package's install script assumes a flat layout.

You lose pnpm's strictness and most of its disk-space advantage (the store still dedupes across projects, but within a project everything is hard-linked into one flat tree).

## PnP Layout (`nodeLinker: pnp`)

No `node_modules` directory at all. pnpm writes Yarn Berry-style `.pnp.cjs` and zip archives. Requires tooling that supports PnP (Yarn's loader, `pnp-webpack-plugin`, etc.).

```yaml
nodeLinker: pnp
symlink: false
```

Not pnpm's primary mode. Use only if you're committed to the PnP ecosystem.

## Layout-Related Settings

| Setting | Default | Effect |
|---------|---------|--------|
| `nodeLinker` | `isolated` | Which layout (see above). |
| `modulesDir` | `node_modules` | Directory name for the install root. |
| `virtualStoreDir` | `node_modules/.pnpm` | Where the virtual store lives. Can be set outside `node_modules` for unusual setups. |
| `virtualStoreDirMaxLength` | `120` (Linux/macOS), `60` (Windows) | Truncate long virtual store paths to avoid ENAMETOOLONG. |
| `virtualStoreOnly` | `false` | If `true`, only create `node_modules/.pnpm` and don't symlink direct deps to the root. For unusual tooling. |
| `symlink` | `true` | Use symlinks. Set `false` with `nodeLinker: pnp` or when symlinks aren't supported. |
| `enableModulesDir` | `true` | If `false`, don't write `node_modules` at all (write lockfile only). |
| `packageImportMethod` | `auto` | How to get files from store to virtual store: `auto`, `hardlink`, `copy`, `clone`, `clone-or-copy`. `auto` picks hardlink on same filesystem, copy otherwise. |

### `packageImportMethod` values

| Value | Behavior |
|-------|----------|
| `auto` | `hardlink` if store and project on same filesystem, else `copy`. Default. |
| `hardlink` | Hard link. Fast, zero-copy, but requires same filesystem. |
| `copy` | Copy. Slower, uses more disk, but works across filesystems (Docker bind mounts with different fs). |
| `clone` | Filesystem clone (reflink on Linux, clonefile on macOS). Copy-on-write; fast and disk-efficient on supported filesystems. |
| `clone-or-copy` | Try clone; fall back to copy. |

## Diagnosing Layout Problems

### "Cannot find module X" that worked under npm

The package isn't declared in your `package.json` and isn't hoisted to the root. Fixes, in order of preference:

1. Add the missing dependency to `package.json` (correct fix — it's a phantom dependency).
2. Add it to `publicHoistPattern` if it's a transitive you genuinely need at the root (rare).
3. `shamefullyHoist: true` as a last resort.

### Symlinks not followed by a bundler

Some bundlers (older Webpack configs, some Rollup plugins) refuse to follow symlinks. Either configure the bundler to follow them, or use `nodeLinker: hoisted` for that project.

### Path-too-long errors on Windows

Set `virtualStoreDirMaxLength: 60` (or lower). The default on Windows is already 60.

## References

- <https://pnpm.io/symlinked-node-modules-structure>
- <https://pnpm.io/settings#node_modules_layout>
- <https://pnpm.io/motivation#creating-a-non-flat-node_modules-directory>

## Related

- Store that feeds the layout → [store.md](store.md)
- Why there are multiple `foo@1.0.0_<peer>` dirs → [peer-dependencies.md](peer-dependencies.md)
- Config keys → [config.md](config.md)
