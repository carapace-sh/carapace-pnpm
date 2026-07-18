# Peer Dependency Resolution

How pnpm resolves peer dependencies, why it creates multiple copies of the same package version, and the settings that control peer behavior. For the node_modules layout that results, see [node-modules-layout.md](node-modules-layout.md); for the lockfile keys that record peer context, see [lockfile.md](lockfile.md).

> **Source of truth**: <https://pnpm.io/how-peers-are-resolved>, <https://pnpm.io/settings#peer_dependencies_settings>. Verified against pnpm v10–v11.

## The Problem

A package `foo` declares `peerDependencies: { react: "^18" }`. It doesn't bundle `react` — it expects the consumer to provide it. But different consumers in the same dependency tree may provide different `react` versions:

- The root app uses `react@18.2.0`.
- A transitive dep `bar` uses `react@17.0.2` (legacy).

If `foo` is a single instance, which `react` does it see? With npm's flat layout, whichever was hoisted last — buggy and nondeterministic. pnpm solves this by creating **one instance of `foo` per peer set**.

## Peer-Isolated Instances

pnpm keys each virtual-store entry by `<pkg>@<version>(<peer>@<ver>[, ...])`:

```
node_modules/.pnpm/
├── foo@1.0.0_react@18.2.0/
│   └── node_modules/
│       ├── foo/         # hard-linked from store
│       └── react -> ../../react@18.2.0/node_modules/react
├── foo@1.0.0_react@17.0.2/
│   └── node_modules/
│       ├── foo/         # same files, different hard-link target dir
│       └── react -> ../../react@17.0.2/node_modules/react
```

Two `foo@1.0.0` dirs, identical file contents, but each symlinked to a different `react`. The store still holds `foo`'s files once (content-addressable); only the virtual-store dirs multiply. This is the source of pnpm's "why are there so many copies of the same version?" — they're cheap (hard links), but the directory entries exist.

In the lockfile, this shows up as separate `snapshots` keys (see [lockfile.md](lockfile.md)):

```yaml
snapshots:
  foo@1.0.0(react@18.2.0):
    dependencies: { react: 18.2.0, ... }
  foo@1.0.0(react@17.0.2):
    dependencies: { react: 17.0.2, ... }
```

## `autoInstallPeers` (default: `true`)

When `true`, pnpm automatically installs peer dependencies that aren't explicitly declared and don't conflict with anything else. This is the "just works" default — you usually don't need to manually add `react` to satisfy a peer.

When `false`, peers must be declared explicitly in `package.json`. Stricter; closer to old npm behavior.

## `strictPeerDependencies` (default: `false`)

| Value | Behavior on unmet/missing peer |
|-------|-------------------------------|
| `false` (default) | Warn; continue. |
| `true` | Fail the install. |

Defaulting to warn avoids breakage on the many packages with over-constrained peers. Turn on `strictPeerDependencies` if you want guarantees (e.g. a library where a wrong peer means a wrong build).

## Other Peer Settings

| Setting | Default | Effect |
|---------|---------|--------|
| `dedupePeerDependents` | `true` | When two consumers would create separate peer-isolated instances with identical resolved peer sets, share one. Reduces instance count. |
| `dedupePeers` | `false` | Dedupe peer deps themselves (more aggressive; can change resolved versions). |
| `resolvePeersFromWorkspaceRoot` | `true` | Use the workspace root's installed peers to satisfy sub-packages' peer requirements. Usually what you want in a monorepo. |
| `peerDependencyRules.ignoreMissing` | `[]` | Suppress "missing peer" warnings for specific packages: `['@types/*', 'eslint']`. |
| `peerDependencyRules.allowedVersions` | `{}` | Widen an over-constrained peer range: `{ 'react-dom': '16 || 17 || 18' }`. |
| `peerDependencyRules.allowAny` | `[]` | Allow any version for listed peer deps (no range check). |

### `peerDependencyRules` examples

```yaml
# pnpm-workspace.yaml
peerDependencyRules:
  ignoreMissing:
    - '@types/react'           # don't warn that @types/react isn't installed
    - 'eslint'                 # plugins expect eslint as peer
  allowedVersions:
    'react-dom': '16 || 17 || 18'   # accept react-dom 16/17/18 as the peer
    'react': '>=17'                  # accept react >=17 even if pkg says ^18
  allowAny:
    - 'eslint'                 # accept any eslint version as the peer
```

## How Resolution Works (Sketch)

1. Build the full dependency graph from `package.json` files, ignoring peers.
2. For each package with peer deps, find all consumers in the graph.
3. For each consumer, determine the peer version it provides (from its own deps or its ancestors' deps).
4. Group consumers by the set of peer versions they provide. Each group gets one peer-isolated instance of the package.
5. If `autoInstallPeers` is on, install missing non-conflicting peers at the root first, so step 3 has something to find.

With `dedupePeerDependents`, instances whose peer sets resolve to the same versions are merged.

## Edge Cases and Gotchas

- **"Why are there three `foo@1.0.0_*` directories?"** Three distinct peer sets in your tree. Use `pnpm why foo` to see who pulls each one. They're hard links, not full copies — cheap on disk.
- **`react-dom` peer mismatch.** The classic: `react@18` installed, but a transitive needs `react-dom@17`. pnpm will create `react-dom@17_xxx_react@17` and `react-dom@18_xxx_react@18`. Usually fine, but some libraries assume a single React instance across the tree (React contexts break with two copies). Fix by aligning React versions across the workspace, or by `overrides` (see [overrides-patch.md](overrides-patch.md)).
- **"Missing peer" warnings you can't fix.** Some packages declare peers they don't really need (e.g. `@types/*` for optional type defs). Add them to `peerDependencyRules.ignoreMissing`.
- **`autoInstallPeers: false` is stricter but more work.** Every peer must be in your `package.json`. Good for libraries; annoying for apps.
- **`resolvePeersFromWorkspaceRoot: true` can mask sub-package peer issues.** The root satisfies a peer for a sub-package; if you later publish the sub-package alone, it breaks. Set `false` for publishable workspace packages.
- **Peer ranges wider than what's installed.** A peer `react: '*'` is satisfied by any installed React. pnpm picks the one the consumer's tree provides; with multiple, it creates multiple instances.

## References

- <https://pnpm.io/how-peers-are-resolved>
- <https://pnpm.io/settings#peer_dependencies_settings>

## Related

- Resulting layout → [node-modules-layout.md](node-modules-layout.md)
- Snapshot keys → [lockfile.md](lockfile.md)
- Forcing a peer version → [overrides-patch.md](overrides-patch.md)
