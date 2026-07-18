---
name: pnpm
description: >
  Use when working with pnpm — the fast, disk-space-efficient Node.js package manager.
  Covers the symlinked node_modules layout (isolated/hoisted/pnp, hoist,
  shamefullyHoist, nodeLinker, virtualStoreDir), the content-addressable global store,
  pnpm-lock.yaml (lockfileVersion 9.0, importers, packages, snapshots),
  pnpm-workspace.yaml monorepos (-r, --filter, workspace: protocol), catalogs
  (catalog: protocol), overrides, packageExtensions, patching (pnpm patch /
  patch-commit), peer dependency isolation (autoInstallPeers, strictPeerDependencies),
  .pnpmfile hooks (readPackage, afterAllResolved, custom resolvers/fetchers), the CLI
  (install, add, update, run, exec, dlx, create, publish, pack, list, why, outdated,
  audit, store, fetch, prune, config, env, runtime), configuration (pnpm-workspace.yaml,
  .npmrc auth, pnpm_config_*, the v11 config split, minimumReleaseAge, trustPolicy,
  allowBuilds), env variables (PNPM_HOME, XDG_DATA_HOME, PNPM_PACKAGE_NAME),
  and how pnpm differs from npm, Yarn Classic, and Yarn Berry.
user-invocable: true
---

# pnpm In-Depth Reference

Comprehensive reference for [pnpm](https://pnpm.io), the fast, disk-space-efficient package manager for Node.js. Covers v10–v11 (current as of 2026). Source of truth: <https://pnpm.io> and <https://github.com/pnpm/spec>.

## Data Flow

```
package.json + pnpm-workspace.yaml + pnpm-lock.yaml + .npmrc
  → resolve (build dependency graph, apply overrides, peer resolution)
    → fetch (download tarballs into the content-addressable global store, keyed by hash)
      → import (hard-link or copy packages from store → node_modules/.pnpm/<pkg>@<ver>/node_modules/<pkg>)
        → symlink (root node_modules/<pkg> → virtual store; per-package node_modules → deps)
          → lifecycle scripts (preinstall / install / postinstall, unless --ignore-scripts)
```

The three-stage design (resolve → fetch → link) is what makes pnpm fast and disk-efficient: the store holds each file once, and projects hard-link from it instead of copying. `pnpm install --frozen-lockfile` skips resolution and uses the lockfile directly; `pnpm fetch` does only resolve + fetch (for Docker layers).

## Sub-Resources

Load the reference that matches your task. When in doubt, load multiple references.

| Keywords | Reference |
|----------|----------|
| command, CLI, subcommand, alias, global flags, --recursive, -r, --filter, --workspace, -w, --save-dev, -D, --save-optional, -O, --save-exact, -E, --prod, -P, --help, --version, exit code | [references/cli.md](references/cli.md) |
| install, i, add, remove, rm, uninstall, update, up, upgrade, prune, fetch, frozen-lockfile, --offline, --prefer-offline, --lockfile-only, --force, --fix-lockfile, --ignore-scripts, --resolution-only, --shamefully-hoist, --update-checksums, install pipeline, resolve, fetch, link, lifecycle, three-stage | [references/install.md](references/install.md) |
| node_modules, layout, isolated, hoisted, pnp, nodeLinker, virtual store, .pnpm, symlink, hard link, hoist, hoistPattern, publicHoistPattern, shamefullyHoist, hoistingLimits, hoistWorkspacePackages, packageImportMethod, virtualStoreDir, modulesDir, enableModulesDir, symlink, phantom dependency, flat node_modules | [references/node-modules-layout.md](references/node-modules-layout.md) |
| store, content-addressable, global store, storeDir, hard link, verifyStoreIntegrity, frozenStore, store path, store add, store prune, store status, global virtual store, PNPM_HOME, XDG_DATA_HOME, packageImportMethod, clone | [references/store.md](references/store.md) |
| lockfile, pnpm-lock.yaml, lockfileVersion, 9.0, ComVer, importers, packages, snapshots, specifier, version, resolution, integrity, tarball, dependencies, optionalDependencies, peerDependencies, transitivePeerDependencies, dependencyPath, peer suffix, frozen lockfile, lockfile-only, gitBranchLockfile, mergeGitBranchLockfiles, lockfileIncludeTarballUrl | [references/lockfile.md](references/lockfile.md) |
| workspace, monorepo, pnpm-workspace.yaml, packages glob, sharedWorkspaceLockfile, linkWorkspacePackages, injectWorkspacePackages, dedupeInjectedDeps, saveWorkspaceProtocol, preferWorkspacePackages, includeWorkspaceRoot, packageConfigs, workspace root, recursive, --filter, --workspace-root | [references/workspace.md](references/workspace.md) |
| config, settings, pnpm-workspace.yaml, config.yaml, .npmrc, auth, registry, _authToken, pnpm_config_, PNPM_CONFIG_, config split, XDG_CONFIG_HOME, registries, namedRegistries, settings reference, all settings | [references/config.md](references/config.md) |
| peer dependency, peerDependencies, peer suffix, autoInstallPeers, strictPeerDependencies, dedupePeerDependents, dedupePeers, resolvePeersFromWorkspaceRoot, peerDependencyRules, ignoreMissing, allowedVersions, allowAny, peer resolution, peer set, isolated peers | [references/peer-dependencies.md](references/peer-dependencies.md) |
| hook, pnpmfile, .pnpmfile.mjs, .pnpmfile.cjs, readPackage, afterAllResolved, beforePacking, preResolution, importPackage, updateConfig, custom resolver, custom fetcher, finder, finders, ignorePnpmfile, globalPnpmfile, canResolve, resolve, canFetch, fetch | [references/hooks.md](references/hooks.md) |
| override, overrides, packageExtensions, allowedDeprecatedVersions, patch, patch-commit, patchedDependencies, allowUnusedPatches, replace package, remove dependency, nested override, fork | [references/overrides-patch.md](references/overrides-patch.md) |
| catalog, catalogs, catalog:, catalog:react17, catalogMode, manual, strict, prefer, cleanupUnusedCatalogs, version range, semver, centralize versions | [references/catalogs.md](references/catalogs.md) |
| run, exec, dlx, pnpx, create, script, package.json scripts, lifecycle scripts, preinstall, postinstall, npm_config_, npm_command, bin, node_modules/.bin, PATH, --if-present, --no-bail, --parallel, --stream, --aggregate-output, --resume-from, --report-summary, --shell-mode, --package, --allow-build | [references/run-exec-dlx.md](references/run-exec-dlx.md) |
| publish, pack, tarball, registry, --tag, --access, --no-git-checks, --publish-branch, --force, --batch, --skip-manifest-obfuscation, --provenance, --dry-run, --otp, --pack-destination, --pack-gzip-level, native publish, --report-summary | [references/publish-pack.md](references/publish-pack.md) |
| environment variable, PNPM_HOME, XDG_DATA_HOME, XDG_CONFIG_HOME, PNPM_PACKAGE_NAME, npm_command, npm_config_, pnpm_config_, PNPM_CONFIG_OTP, NODE_PATH, storeDir, config file location | [references/env-vars.md](references/env-vars.md) |
| npm, yarn, yarn classic, yarn berry, comparison, difference, migration, why pnpm, flat node_modules, hoisted, resolutions, workspaces, lockfile, PnP, Plug'n'Play | [references/vs-npm-yarn.md](references/vs-npm-yarn.md) |

## Quick Guide

- **What is pnpm's mental model and how does it differ from npm/yarn?** → [references/vs-npm-yarn.md](references/vs-npm-yarn.md) and [references/node-modules-layout.md](references/node-modules-layout.md)
- **How does the install pipeline work (resolve → fetch → link)?** → [references/install.md](references/install.md)
- **How does the symlinked node_modules layout work?** → [references/node-modules-layout.md](references/node-modules-layout.md)
- **What is the content-addressable store and where is it?** → [references/store.md](references/store.md)
- **How do I read/understand pnpm-lock.yaml?** → [references/lockfile.md](references/lockfile.md)
- **How do I set up a monorepo / workspace?** → [references/workspace.md](references/workspace.md)
- **Where do settings live now (.npmrc vs pnpm-workspace.yaml, v11 split)?** → [references/config.md](references/config.md)
- **How does peer dependency isolation work and why are there multiple copies?** → [references/peer-dependencies.md](references/peer-dependencies.md)
- **How do I write a .pnpmfile hook (readPackage, afterAllResolved)?** → [references/hooks.md](references/hooks.md)
- **How do I force/override a dependency version?** → [references/overrides-patch.md](references/overrides-patch.md)
- **How do I patch a dependency's source?** → [references/overrides-patch.md](references/overrides-patch.md)
- **How do catalogs work and what is the `catalog:` protocol?** → [references/catalogs.md](references/catalogs.md)
- **How do `pnpm run`, `pnpm exec`, `pnpm dlx`, and `pnpm create` differ?** → [references/run-exec-dlx.md](references/run-exec-dlx.md)
- **How does `pnpm publish` work and what changed in v11?** → [references/publish-pack.md](references/publish-pack.md)
- **What environment variables does pnpm read (PNPM_HOME, pnpm_config_*)?** → [references/env-vars.md](references/env-vars.md)
- **Which command/flag do I need for X?** → [references/cli.md](references/cli.md)
- **What does `--frozen-lockfile` do vs `--lockfile-only`?** → [references/install.md](references/install.md)
- **What is `node_modules/.pnpm` and why is everything symlinked there?** → [references/node-modules-layout.md](references/node-modules-layout.md)
- **How do I fix "phantom dependency" / "unmet peer dependency" errors?** → [references/peer-dependencies.md](references/peer-dependencies.md) and [references/node-modules-layout.md](references/node-modules-layout.md)
- **How do I make pnpm behave like npm (flat node_modules)?** → [references/node-modules-layout.md](references/node-modules-layout.md) (`shamefullyHoist` / `nodeLinker: hoisted`)
- **What does `--shamefully-hoist` do?** → [references/node-modules-layout.md](references/node-modules-layout.md)
- **How do I run a one-off package without installing it (`pnpm dlx`)?** → [references/run-exec-dlx.md](references/run-exec-dlx.md)
- **How do I run a script across all workspace packages (`pnpm -r run`)?** → [references/run-exec-dlx.md](references/run-exec-dlx.md) and [references/workspace.md](references/workspace.md)

## Cross-Project References

- For **carapace** (the shell completion engine) internals — Action API, traverse engine, shell formatters — use the **carapace-dev** skill. The pnpm completer in `carapace-pnpm` uses carapace to generate completions for pnpm's CLI; this skill documents pnpm itself, not the completer.
- For **shell quoting/expansion** rules that affect how `pnpm run`/`pnpm exec` pass arguments to scripts, see the **bash**, **zsh**, or **fish** skill as appropriate for the target shell. pnpm's `--shell-mode` and `scriptShell` settings interact with these.
- For **npm registry / package.json / semver** semantics that pnpm inherits (semver range syntax, `package.json` fields, registry HTTP API), see the npm docs at <https://docs.npmjs.com>. pnpm is npm-compatible for these; this skill documents only pnpm-specific behavior.
- For **Node.js module resolution** (`require.resolve`, the node_modules lookup algorithm, ESM resolution), see the Node.js docs at <https://nodejs.org/api/modules.html>. pnpm's symlinked layout is designed to be compatible with Node's algorithm — see [references/node-modules-layout.md](references/node-modules-layout.md) for how.
