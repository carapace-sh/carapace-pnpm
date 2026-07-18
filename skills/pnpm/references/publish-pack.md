# `pnpm publish` and `pnpm pack`

Publishing to a registry and creating tarballs. Covers the native publish implementation (v11+) and the flags that control both commands. For `beforePacking` hooks, see [hooks.md](hooks.md); for workspace `workspace:` protocol rewriting, see [workspace.md](workspace.md).

> **Source of truth**: <https://pnpm.io/cli/publish>, <https://pnpm.io/cli/pack>. Verified against pnpm v10–v11.

## `pnpm publish`

Publishes the current package (or every workspace package with `-r`) to the registry. Since v11, pnpm has a **native** publish implementation — it no longer delegates to `npm publish`.

```bash
pnpm publish                      # publish the current package
pnpm publish --tag next           # publish under the "next" dist-tag
pnpm publish --access public      # for scoped packages (@scope/pkg)
pnpm publish -r                   # publish every workspace package (topological)
pnpm publish -r --batch           # batch-publish, pausing for confirmation
pnpm publish --dry-run            # show what would be published; don't upload
pnpm publish --no-git-checks      # skip the git-cleanliness checks
pnpm publish --otp 123456         # 2FA one-time password
pnpm publish --provenance         # attach SLSA provenance (npm provenance)
```

### Flags

| Flag | Effect |
|------|--------|
| `--tag <tag>` | Dist-tag for this publish (default `latest`). |
| `--access <public\|restricted>` | For scoped packages: public or restricted. |
| `--otp <code>` | 2FA one-time password. Also via `PNPM_CONFIG_OTP` env. |
| `--dry-run` | Pack and show the manifest/tarball; don't upload. |
| `--no-git-checks` | Skip checks that the working tree is clean and on the publish branch. |
| `--publish-branch <branch>` | Require being on this branch (default `master`/`main`). |
| `--force` | Publish even if the version already exists (rare; registry may refuse). |
| `--batch` | (`-r`) Batch-publish with a confirmation prompt per package. |
| `--report-summary` | Write a JSON summary of published packages. |
| `--json` | Emit JSON output. |
| `--skip-manifest-obfuscation` | Don't strip dev/test fields from the published manifest. |
| `--provenance` | Attach SLSA build provenance (GitHub Actions, npm provenance). |
| `--recursive`, `-r` | Publish all workspace packages. |

### Git checks (default on)

Before publishing, pnpm verifies:

1. The working tree is clean (no uncommitted changes).
2. The current branch matches `--publish-branch` (default `master` or `main`).
3. The current commit is pushed to the remote.

`--no-git-checks` skips all three. Use in CI where the build runs on a detached HEAD or a non-master branch.

### Lifecycle scripts

pnpm runs the standard npm lifecycle scripts around publish:

- `prepublishOnly` — before the tarball is built and uploaded. Most common place for `pnpm run build` + tests.
- `prepare` — before publish if not already run.
- `prepack` / `postpack` — around tarball creation.
- `publish` / `postpublish` — after the upload.

`prepublishOnly` is the recommended hook for "build before publish." (`prepublish` runs on install too, historically; avoid it.)

### Workspace `workspace:` rewriting

When publishing a workspace package, pnpm rewrites every `workspace:` range in its `package.json` to the concrete version of the referenced workspace package. So `"shared": "workspace:*"` becomes `"shared": "^1.4.0"` (per `saveWorkspaceProtocol`) in the published tarball. See [workspace.md](workspace.md).

`catalog:` ranges are similarly rewritten to their concrete versions. See [catalogs.md](catalogs.md).

### Recursive publish

`pnpm publish -r` publishes every workspace package in topological order (dependencies before dependents). `--batch` adds a per-package confirmation prompt. Without `--batch`, pnpm publishes all without prompting (use `--dry-run` first to preview).

### Manifest obfuscation

By default, pnpm strips fields from the published `package.json` that are only useful in development: `scripts` (other than lifecycle ones the registry uses), `devDependencies`, etc. `--skip-manifest-obfuscation` keeps the full manifest. This is a pnpm-specific nicety; npm publishes the whole `package.json`.

## `pnpm pack`

Creates a tarball without publishing. By default writes `<name>-<version>.tgz` to the current directory.

```bash
pnpm pack                       # create <name>-<version>.tgz in cwd
pnpm pack --pack-destination ./dist   # write tarball to ./dist
pnpm pack --out ./my-tarball.tgz      # explicit output path
pnpm pack --json                # emit JSON describing the tarball
pnpm pack --dry-run             # list files that would be packed; don't write
pnpm pack -r                    # pack every workspace package
pnpm pack --pack-gzip-level 9   # max compression
```

### Flags

| Flag | Effect |
|------|--------|
| `--pack-destination <dir>` | Directory to write tarballs into. |
| `--out <path>` | Explicit output path (single-package only). |
| `--pack-gzip-level <n>` | gzip compression level (1–9). |
| `--json` | Emit JSON with tarball path, name, version, files. |
| `--dry-run` | Show what would be packed; don't write. |
| `--recursive`, `-r` | Pack every workspace package. |
| `--skip-manifest-obfuscation` | Keep the full manifest in the tarball. |

### What gets packed

The tarball includes:

- All files in the package directory matching the `files` field in `package.json` (or, if `files` is absent, everything not in `.npmignore`/`.gitignore`).
- `package.json` (with `workspace:`/`catalog:` ranges rewritten — same as publish).
- `README*`, `LICENSE*`, `CHANGELOG*` (always included by convention).

Use `pnpm pack --dry-run` to see the exact file list before packing.

## `beforePacking` Hook

```javascript
// .pnpmfile.mjs
export default {
  hooks: {
    beforePacking(pkg) {
      // mutate the manifest that will be packed/published
      delete pkg.scripts?.dev
      delete pkg.scripts?.test
      delete pkg.devDependencies
    },
  },
}
```

Runs before both `pnpm pack` and `pnpm publish`. See [hooks.md](hooks.md).

## Publishing to a Custom Registry

Set the registry per-publish or via config:

```bash
pnpm publish --registry=https://my-registry.com/
```

```yaml
# pnpm-workspace.yaml (v11)
registries:
  default: https://my-registry.com/
namedRegistries:
  company: https://registry.company.com/
```

Auth lives in `.npmrc` (or `~/.config/pnpm/auth.ini`) — see [config.md](config.md).

## Provenance

`--provenance` attaches a SLSA build provenance statement to the published package, allowing consumers to verify the package was built from a specific source commit in a specific CI environment. Requires:

- Publishing from GitHub Actions.
- The `id-token: write` permission in the workflow.
- A registry that supports provenance (npm public registry does).

## Edge Cases and Gotchas

- **`prepublishOnly` is the place to build.** Don't use `prepublish` — it also runs on `pnpm install`, which would build on every install.
- **Version already exists.** The registry rejects re-publish of an existing version. Bump the version first (`npm version patch` or your tooling). `--force` only bypasses pnpm's own checks, not the registry's.
- **Scoped package access.** Scoped packages (`@scope/pkg`) default to restricted (private) on publish. Use `--access public` to publish publicly.
- **`workspace:` not rewritten.** If a published package still shows `workspace:*`, you published with `npm publish` instead of `pnpm publish`, or `saveWorkspaceProtocol: false` is set in a way that prevented rewriting. Re-publish with `pnpm publish`.
- **`--no-git-checks` in CI.** Most CI runs on a detached HEAD or a non-master branch. `--no-git-checks` is usually required there.
- **`pack` and `.npmignore`.** If both `.npmignore` and `.gitignore` exist, `.npmignore` takes precedence for packing. If only `.gitignore` exists, pnpm uses it. The `files` field in `package.json` overrides both.
- **Manifest obfuscation and downstream tooling.** Some tools read `scripts` from published packages (rare). If something breaks, `--skip-manifest-obfuscation` keeps the full manifest.
- **Native publish (v11) and npm CLI plugins.** pnpm's native publish doesn't support npm's `--provenance-file` or some npm-specific flags. Check the pnpm docs if you relied on an `npm publish`-only flag.

## References

- <https://pnpm.io/cli/publish>
- <https://pnpm.io/cli/pack>

## Related

- `beforePacking` hook → [hooks.md](hooks.md)
- `workspace:`/`catalog:` rewriting → [workspace.md](workspace.md), [catalogs.md](catalogs.md)
- Registry & auth config → [config.md](config.md)
