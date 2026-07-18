# Configuration

Where pnpm settings live, the v11 config split, how settings are resolved, and a reference to the most common settings. For the workspace file's workspace-specific keys, see [workspace.md](workspace.md); for env vars, see [env-vars.md](env-vars.md).

> **Source of truth**: <https://pnpm.io/settings>, <https://pnpm.io/npmrc>, <https://pnpm.io/next/configuring>. Verified against pnpm v10–v11.

## The v11 Config Split

Before v11, pnpm read all settings (auth and non-auth) from `.npmrc` files, like npm. In v11 the configuration was split:

| Setting type | Project file | User (global) file |
|--------------|--------------|--------------------|
| **All non-auth settings** | `pnpm-workspace.yaml` | `~/.config/pnpm/config.yaml` |
| **Auth & registry credentials** | `.npmrc` | `~/.config/pnpm/auth.ini` |

`.npmrc` at the project level now holds **only auth** (registry tokens, certificates). Everything else — `nodeLinker`, `hoist`, `autoInstallPeers`, `overrides`, `catalog`, `packageExtensions`, etc. — lives in `pnpm-workspace.yaml`.

### Why the split

- YAML supports nested structures (maps, lists) which `overrides`, `catalogs`, `packageExtensions`, and `packageConfigs` need. INI (`.npmrc`) can only do flat key=value.
- Auth settings are secrets and benefit from a distinct, more restrictive file (no env-var expansion in project `.npmrc` since v11.5.3 — see "Security" below).
- One settings file per workspace instead of settings scattered across `.npmrc`, `package.json`, and the workspace file.

### Migration

`pnpm setup` and `pnpm install` will migrate old `.npmrc` non-auth settings into `pnpm-workspace.yaml`/`config.yaml` on first run of v11. Auth stays in `.npmrc`.

## Precedence (lowest to highest)

1. Built-in defaults.
2. User global config: `~/.config/pnpm/config.yaml` (path overridable via `XDG_CONFIG_HOME`).
3. Project config: `pnpm-workspace.yaml` in the current workspace root.
4. Environment variables: `pnpm_config_<key>` (or `PNPM_CONFIG_<key>`).
5. CLI flags.

A higher layer overrides a lower one for the same key. **Note (v11)**: pnpm no longer reads `npm_config_*` env vars for its own settings — use `pnpm_config_*`. It does still *populate* `npm_config_*` for the benefit of scripts that expect them (see [run-exec-dlx.md](run-exec-dlx.md)).

## `.npmrc` — Auth Only

Project `.npmrc` (next to `pnpm-workspace.yaml`) and user `~/.config/pnpm/auth.ini` hold:

```ini
//registry.npmjs.org/:_authToken=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//registry.npmjs.org/:always-auth=true
//registry.npmjs.org/:cafile=/path/to/ca.pem
//my-registry.company.com/:_authToken=...
```

Other auth-related keys: `cert`, `key`, `ca` (inline PEM strings). Registry URLs themselves (`registry=...`) are non-auth and go in `pnpm-workspace.yaml` under `registries` (v11).

### Security note (v11.5.3+)

Environment variables in **project** `.npmrc` files are **not expanded**. This prevents a malicious `.npmrc` in a cloned repo from exfiltrating env vars. Put tokens in user-level auth files or pass via `pnpm_config_*` env vars / CLI.

## `pnpm config`

```bash
pnpm config get nodeLinker                      # read
pnpm config set nodeLinker hoisted              # set in project config
pnpm config set nodeLinker hoisted --location global   # set in user config
pnpm config delete nodeLinker                   # delete
pnpm config list                                # list all
pnpm config list --json                         # as JSON
```

`--location global` (or `-g`) targets `~/.config/pnpm/config.yaml`; the default targets the project. `set`/`delete` rewrite the YAML file.

## Settings Reference (Most Common)

Grouped by concern. Defaults shown. For exhaustive lists, see <https://pnpm.io/settings>.

### Node-modules layout (see [node-modules-layout.md](node-modules-layout.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `nodeLinker` | `isolated` | `isolated` \| `hoisted` \| `pnp`. |
| `modulesDir` | `node_modules` | Install root directory name. |
| `virtualStoreDir` | `node_modules/.pnpm` | Virtual store path. |
| `virtualStoreDirMaxLength` | 120 / 60 (Win) | Max virtual store path segment length. |
| `virtualStoreOnly` | `false` | Only create `.pnpm`, no root symlinks. |
| `symlink` | `true` | Use symlinks (set false with `pnp`). |
| `enableModulesDir` | `true` | Write `node_modules` at all. |
| `packageImportMethod` | `auto` | `auto` \| `hardlink` \| `copy` \| `clone` \| `clone-or-copy`. |

### Hoisting (see [node-modules-layout.md](node-modules-layout.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `hoist` | `true` | Hoist to `.pnpm/node_modules`. |
| `hoistPattern` | `['*']` | What to hoist to `.pnpm/node_modules`. |
| `publicHoistPattern` | `[]` | What to hoist to root `node_modules`. |
| `shamefullyHoist` | `false` | `publicHoistPattern: ['*']` (npm-flat). |
| `hoistingLimits` | `none` | `none` \| `workspaces` \| `dependencies`. |
| `hoistWorkspacePackages` | `true` | Symlink workspace packages to hoisted location. |

### Peer dependencies (see [peer-dependencies.md](peer-dependencies.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `autoInstallPeers` | `true` | Automatically install non-conflicting peers. |
| `dedupePeerDependents` | `true` | Dedupe packages whose peers overlap. |
| `dedupePeers` | `false` | Dedupe peer deps themselves. |
| `strictPeerDependencies` | `false` | Fail on any unmet/missing peer (default is warn). |
| `resolvePeersFromWorkspaceRoot` | `true` | Use root's peers to satisfy sub-package peers. |
| `peerDependencyRules.ignoreMissing` | `[]` | Suppress "missing peer" warnings for listed packages. |
| `peerDependencyRules.allowedVersions` | `{}` | Override acceptable peer version ranges. |
| `peerDependencyRules.allowAny` | `[]` | Allow any version for listed peer deps. |

### Lockfile (see [lockfile.md](lockfile.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `lockfile` | `true` | Read/write lockfile. |
| `preferFrozenLockfile` | `true` | In workspaces, prefer the frozen lockfile. |
| `lockfileIncludeTarballUrl` | `false` | Include tarball URL in resolution. |
| `gitBranchLockfile` | `false` | Per-branch lockfiles. |
| `sharedWorkspaceLockfile` | `true` | One lockfile at workspace root. |

### Store (see [store.md](store.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `storeDir` | auto | Store directory. |
| `verifyStoreIntegrity` | `true` | Verify hashes on read. |
| `frozenStore` | `false` | Never write to store (read-only CI). |
| `enableGlobalVirtualStore` | `false` | Use global virtual store for global installs (v11). |

### Workspace (see [workspace.md](workspace.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `linkWorkspacePackages` | `false` | `false` \| `true` \| `deep`. |
| `injectWorkspacePackages` | `false` | Inject built output instead of symlinking. |
| `dedupeInjectedDeps` | `true` | Dedupe across consumers. |
| `saveWorkspaceProtocol` | `rolling` | `rolling` \| `true` \| `false`. |
| `preferWorkspacePackages` | `false` | Prefer workspace over registry on match. |
| `includeWorkspaceRoot` | `false` | Include root in `-r` runs. |

### Dependency resolution / supply chain

| Setting | Default | Effect |
|---------|---------|--------|
| `overrides` | `{}` | Force versions. See [overrides-patch.md](overrides-patch.md). |
| `packageExtensions` | `{}` | Extend package manifests. |
| `allowedDeprecatedVersions` | `{}` | Suppress deprecation warnings per package. |
| `minimumReleaseAge` | `1440` (v11) | Min minutes since publish before a version is allowed. |
| `minimumReleaseAgeExclude` | `[]` | Packages exempt from min release age. |
| `minimumReleaseAgeStrict` | `false` | Fail if no version meets the min age. |
| `trustPolicy` | `off` | `off` \| `no-downgrade` (block downgrades of trusted packages). |
| `blockExoticSubdeps` | `true` | Only direct deps may use exotic (git/file/url) sources. |

### Catalogs (see [catalogs.md](catalogs.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `catalog` | — | Default catalog map. |
| `catalogs` | — | Named catalogs. |
| `catalogMode` | `manual` | `manual` \| `strict` \| `prefer`. |
| `cleanupUnusedCatalogs` | `false` | Remove unused catalog entries on install. |

### Patching (see [overrides-patch.md](overrides-patch.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `patchedDependencies` | `{}` | Map of `pkg@ver: patchfile`. |
| `allowUnusedPatches` | `false` | Don't fail when a registered patch's package is absent. |

### Scripts & shell (see [run-exec-dlx.md](run-exec-dlx.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `enablePrePostScripts` | `true` | Run `pre*`/`post*` lifecycle scripts. |
| `scriptShell` | — | Custom shell for `pnpm run`/`exec`. |
| `shellEmulator` | `false` | Use pnpm's JS bash-like shell instead of system shell. |
| `ignoreScripts` | `false` | Skip install scripts globally. |
| `allowBuilds` | `—` | Map of package matchers → `true`/`false` controlling which deps may run install scripts (v11; replaces v10's `onlyBuiltDependencies`/`neverBuiltDependencies`). Unlisted packages are disallowed. |
| `strictDepBuilds` | `true` (v11) | Exit non-zero if any unlisted dependency has build scripts. `false` downgrades to a warning. |
| `dangerouslyAllowAllBuilds` | `false` | Allow all packages (including transitive) to run install scripts without review (v10.9+). |

### Hooks (see [hooks.md](hooks.md))

| Setting | Default | Effect |
|---------|---------|--------|
| `pnpmfile` | `['.pnpmfile.mjs']` | Pnpmfile path(s). |
| `ignorePnpmfile` | `false` | Skip pnpmfile hooks. |
| `globalPnpmfile` | `null` | Global pnpmfile path. |

### Registry & network

| Setting | Default | Effect |
|---------|---------|--------|
| `registries` (v11) / `registry` | `https://registry.npmjs.org/` | Default registry; v11 uses a `registries` map. |
| `namedRegistries` (v11) | — | Named registry aliases. |
| `fetchRetries` | `2` | Network retry count. |
| `fetchRetryFactor` | `10` | Exponential backoff factor. |
| `fetchRetryMaxtimeout` | `60000` | Max retry timeout (ms). |
| `fetchTimeout` | `60000` | Per-request timeout (ms). |
| `networkConcurrency` | `16` | Max concurrent network requests. |
| `strictSsl` | `true` | Verify TLS certificates. |

### Caching

| Setting | Default | Effect |
|---------|---------|--------|
| `dlxCacheMaxAge` | `1440` (1 day) | Max age of `pnpm dlx` cached packages (minutes). |
| `modulesCacheMaxAge` | `10080` (7 days) | Max age of unused `node_modules` before pruning (minutes). |

## Edge Cases and Gotchas

- **A setting in `.npmrc` that's not auth.** Since v11, pnpm ignores non-auth settings in `.npmrc`. Move it to `pnpm-workspace.yaml`. The migration usually does this; manual edits may not.
- **`npm_config_*` env vars don't work anymore.** Use `pnpm_config_*`. pnpm populates `npm_config_*` for scripts only.
- **Project config overriding global.** A `pnpm-workspace.yaml` setting beats the user `config.yaml`. To inspect effective config: `pnpm config list`.
- **Type coercion in YAML.** `nodeLinker: hoisted` is a string; `hoist: false` is a bool; `hoistPattern: ['*eslint*']` is a list. YAML's bareword rules mean `nodeLinker: hoisted` is fine but `registry: https://...` is also fine (no quotes needed for these). Quote strings with special chars.
- **Settings that change resolution invalidate the lockfile.** Changing `autoInstallPeers`, `overrides`, `nodeLinker`, or `--no-optional` requires regenerating the lockfile; `--frozen-lockfile` will fail until you do.

## References

- All settings: <https://pnpm.io/settings>
- Auth / `.npmrc`: <https://pnpm.io/npmrc>
- v11 config split: <https://pnpm.io/next/configuring>

## Related

- Workspace file → [workspace.md](workspace.md)
- Env vars → [env-vars.md](env-vars.md)
- Layout keys → [node-modules-layout.md](node-modules-layout.md)
- Peer keys → [peer-dependencies.md](peer-dependencies.md)
- Hooks keys → [hooks.md](hooks.md)
