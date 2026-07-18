# Environment Variables

The environment variables pnpm reads for configuration and the ones it sets for scripts. For the config-file layer, see [config.md](config.md); for store-location variables, see [store.md](store.md); for the variables set during `pnpm run`/`exec`, see [run-exec-dlx.md](run-exec-dlx.md).

> **Source of truth**: <https://pnpm.io/next/configuring>, <https://pnpm.io/cli/run#environment-variables>. Verified against pnpm v10–v11.

## Variables pnpm Reads

### `pnpm_config_*` (primary config override)

pnpm reads environment variables prefixed with `pnpm_config_` (lowercase) or `PNPM_CONFIG_` (uppercase) and applies them as config overrides — above project config, below CLI flags.

```bash
pnpm_config_save_exact=true pnpm add lodash     # pin exact version this once
pnpm_config_node_linker=hoisted pnpm install    # use hoisted layout this once
pnpm_config_registry=https://my-registry.com/ pnpm install
```

Any setting from [config.md](config.md) can be set this way: replace dots/camelCase with underscores. `pnpm_config_frozen_lockfile=true`, `pnpm_config_auto_install_peers=false`, etc.

**v11 change**: pnpm no longer reads `npm_config_*` for its own settings. Use `pnpm_config_*`. (pnpm still *populates* `npm_config_*` for scripts that expect them — see below.)

### `PNPM_HOME`

Location of pnpm's home directory. Used for:

- Global installs (`pnpm add -g`).
- The bin shims that make global bins available on `PATH`.
- The store, if no other store location is configured: `$PNPM_HOME/store`.

Set by `pnpm setup` and typically added to `PATH`:

```bash
# ~/.bashrc / ~/.zshrc (after `pnpm setup`)
export PNPM_HOME="$HOME/.local/share/pnpm"
export PATH="$PNPM_HOME:$PATH"
```

If `PNPM_HOME` is set, the store defaults to `$PNPM_HOME/store` (overrides the platform default but is itself overridden by `XDG_DATA_HOME` or `storeDir`).

### `XDG_DATA_HOME`

Overrides the base directory for user data files. pnpm's store defaults to `$XDG_DATA_HOME/pnpm/store` when set (otherwise the platform default: `~/.local/share/pnpm/store` on Linux, `~/Library/pnpm/store` on macOS, `~/AppData/Local/pnpm/store` on Windows).

### `XDG_CONFIG_HOME`

Overrides the base directory for user config files. pnpm's global config defaults to `$XDG_CONFIG_HOME/pnpm/config.yaml` (otherwise `~/.config/pnpm/config.yaml`), and global auth to `$XDG_CONFIG_HOME/pnpm/auth.ini`.

### Store-location priority (recap)

1. `storeDir` in config (highest).
2. `$PNPM_HOME/store` (if `PNPM_HOME` set).
3. `$XDG_DATA_HOME/pnpm/store` (if `XDG_DATA_HOME` set).
4. Platform default (`~/.local/share/pnpm/store` on Linux, etc.).

See [store.md](store.md) for details.

## Variables pnpm Sets for Scripts

When pnpm runs a script (`pnpm run`, `pnpm exec`, lifecycle scripts during install), it populates:

| Variable | Value |
|----------|-------|
| `PATH` | `node_modules/.bin` (project + workspace roots) prepended to the existing `PATH`. |
| `NODE_PATH` | (Hoisted-layout only) includes the hoisted deps root so `require` finds them. Doesn't work with ESM. |
| `npm_lifecycle_event` | The current script name (e.g. `build`, `install`). |
| `npm_command` | The high-level command: `run-script`, `install`, `publish`, etc. |
| `npm_config_<key>` | Each resolved config value, populated for npm-script compatibility. Scripts reading `npm_config_registry` etc. keep working. |
| `pnpm_config_<key>` | The same config values, under pnpm's own prefix. |
| `PNPM_PACKAGE_NAME` | The current workspace package's name (in `pnpm -r` runs). |
| `PNPM_PACKAGE_DIR` | The current workspace package's directory (in `pnpm -r` runs). |
| `INIT_CWD` | The directory from which pnpm was invoked (may differ from the script's `cwd` in recursive runs). |
| `npm_lifecycle_event` | (Repeat) the script name. |
| `npm_package_json` | Path to the current `package.json`. |
| `npm_node_execpath` | Path to the Node binary. |

`pnpm exec` and `pnpm dlx` set a subset (the ones relevant to a one-off command, not lifecycle).

## Other Variables

| Variable | Purpose |
|----------|---------|
| `PNPM_CONFIG_OTP` | One-time password for 2FA publishing. Equivalent to `pnpm publish --otp`. |
| `npm_config_user_agent` | Set to `pnpm/<version> ...` so registries can detect pnpm. |
| `COREPACK_ENABLE_DOWNLOAD_PROMPT` | (Corepack-managed installs) controls whether Corepack prompts before downloading pnpm. |
| `NO_COLOR` | If set, pnpm disables ANSI colors (same as `--no-color`). |
| `FORCE_COLOR` | Force ANSI colors even when stdout isn't a TTY. |

## Variables No Longer Read (v11)

| Variable | v10 behavior | v11 behavior |
|----------|--------------|--------------|
| `npm_config_*` | Read as pnpm config overrides. | **Not read** for pnpm's own config. Use `pnpm_config_*`. pnpm still *writes* these for scripts. |
| `.npmrc` non-auth keys | Read as config. | **Not read**. Use `pnpm-workspace.yaml`. |
| Project `.npmrc` env-var expansion | `$VAR` expanded in project `.npmrc`. | **Not expanded** (security). Use user-level auth files or `pnpm_config_*` env vars. |

## Edge Cases and Gotchas

- **`npm_config_*` in scripts.** pnpm populates them for script compat, but if a script *also* tries to use them to configure pnpm (e.g. a script that re-invokes `pnpm install`), they won't affect pnpm's own config in v11. Use `pnpm_config_*` for that.
- **`PNPM_HOME` must be on `PATH`.** `pnpm setup` adds it; if you skip `pnpm setup` (e.g. a CI image that pre-installs pnpm via Corepack), set `PNPM_HOME` and add it to `PATH` manually, or global bins won't be found.
- **`XDG_DATA_HOME` and Docker.** Setting `XDG_DATA_HOME=/cache` in a Docker image puts the store at `/cache/pnpm/store`. Mount `/cache` as a named volume to persist the store across builds.
- **`NODE_PATH` and ESM.** `NODE_PATH` is ignored by ESM `import`. The hoisted-layout workaround it enables (`require` finding hoisted deps) doesn't help ESM code. Use the isolated layout or declare deps properly.
- **Env vars vs. config files.** `pnpm_config_*` overrides project config but not CLI flags. Precedence: defaults < user config < project config < `pnpm_config_*` env < CLI flags. See [config.md](config.md).
- **`PNPM_CONFIG_OTP` vs `--otp`.** Both work; the env var is handy when you don't want to type the OTP per publish in a script.
- **Project `.npmrc` and `${VAR}`.** Since v11.5.3, `${VAR}` in a project `.npmrc` is **literal** — not expanded. This is intentional (security: a cloned repo's `.npmrc` can't steal env vars). Put tokens in `~/.config/pnpm/auth.ini` or pass via `pnpm_config_*`.

## References

- <https://pnpm.io/next/configuring>
- <https://pnpm.io/cli/run#environment-variables>
- <https://pnpm.io/global-packages>

## Related

- Config-file layer → [config.md](config.md)
- Store location → [store.md](store.md)
- Script environment → [run-exec-dlx.md](run-exec-dlx.md)
