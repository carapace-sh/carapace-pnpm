# Catalogs

The `catalog:` protocol — reusable version range constants defined once in `pnpm-workspace.yaml` and referenced from any workspace package's `package.json`. For the workspace file, see [workspace.md](workspace.md); for the lockfile representation, see [lockfile.md](lockfile.md).

> **Source of truth**: <https://pnpm.io/catalogs>. Verified against pnpm v10–v11 (catalogs landed in v9).

## What Catalogs Solve

In a monorepo without catalogs, the same dep appears in many `package.json` files with the same range, and upgrading means editing every file:

```jsonc
// packages/app/package.json
{ "dependencies": { "react": "^18.2.0" } }
// packages/widget/package.json
{ "dependencies": { "react": "^18.2.0" } }
// packages/shared/package.json
{ "dependencies": { "react": "^18.2.0" } }
```

Catalogs let you define the range once and reference it by name:

```yaml
# pnpm-workspace.yaml
catalog:
  react: ^18.2.0
  redux: ^5.0.1
```

```jsonc
// packages/app/package.json
{ "dependencies": { "react": "catalog:", "redux": "catalog:" } }
// packages/widget/package.json
{ "dependencies": { "react": "catalog:" } }
```

To upgrade React across the monorepo, change one line in `pnpm-workspace.yaml` and run `pnpm install`.

## The `catalog:` Protocol

In a `package.json` dependency range, the literal string `catalog:` means "use the version from the default catalog." On publish, pnpm replaces `catalog:` with the concrete resolved range so consumers don't need to know about catalogs.

```json
{
  "dependencies": {
    "react": "catalog:",
    "redux": "catalog:"
  }
}
```

## Named Catalogs

When different parts of the monorepo need different versions of the same package (e.g. one app on React 17, another on React 18), use **named catalogs**:

```yaml
# pnpm-workspace.yaml
catalogs:
  react17:
    react: ^17.0.2
    react-dom: ^17.0.2
  react18:
    react: ^18.2.0
    react-dom: ^18.2.0
```

Reference a named catalog with `catalog:<name>`:

```json
// packages/legacy-app/package.json
{ "dependencies": { "react": "catalog:react17", "react-dom": "catalog:react17" } }

// packages/new-app/package.json
{ "dependencies": { "react": "catalog:react18", "react-dom": "catalog:react18" } }
```

The default catalog is `catalog:` (no name); named catalogs are `catalog:<name>`.

## `catalogMode`

| Mode | Behavior |
|------|----------|
| `manual` (default) | A `catalog:` specifier is resolved normally. A direct version range (e.g. `^18.2.0`) in `package.json` is also allowed — catalogs are opt-in per dependency. |
| `strict` | Every dependency that has a catalog entry **must** use `catalog:`. A direct range that matches a cataloged package is an error. Forces all cataloged deps through the catalog. |
| `prefer` | Like `manual`, but `pnpm install` will rewrite direct ranges to `catalog:` for any dep that has a catalog entry. Soft migration to `strict`. |

```yaml
catalogMode: strict
```

## `cleanupUnusedCatalogs`

```yaml
cleanupUnusedCatalogs: false   # default; leave unused catalog entries alone
cleanupUnusedCatalogs: true    # remove catalog entries that no package references
```

Set `true` to keep the catalog tidy automatically.

## Adding a Dependency to the Catalog

```bash
# Add to the default catalog and to this package's dependencies
pnpm add --save-catalog react

# Add to a named catalog
pnpm add --save-catalog react --catalog react18
```

`--save-catalog` writes the entry into `pnpm-workspace.yaml`'s `catalog` (or named catalog) and adds `"react": "catalog:"` (or `"catalog:react18"`) to `package.json`.

## How Catalogs Appear in the Lockfile

The lockfile's `importers` section records both the specifier and the resolved version:

```yaml
importers:
  packages/app:
    dependencies:
      react:
        specifier: catalog:
        version: 18.2.0
  packages/legacy-app:
    dependencies:
      react:
        specifier: catalog:react17
        version: 17.0.2
```

And the catalog definitions themselves appear under `catalogs` at the top level. See [lockfile.md](lockfile.md).

## On Publish

When `pnpm publish` packs a package, it rewrites every `catalog:` (and `catalog:<name>`) range in its `package.json` to the concrete version the catalog resolved to. Consumers of the published package see normal semver ranges — they don't need pnpm or your catalog.

This is why catalogs are workspace-internal: they're a development-time convenience, erased on publish.

## Edge Cases and Gotchas

- **`catalog:` in a published package.** If you see `catalog:` in a published package's `package.json`, the publish didn't rewrite it — you published manually (`npm publish`) instead of `pnpm publish`, or `saveWorkspaceProtocol` is misconfigured. Re-publish with `pnpm publish`.
- **A package not in any catalog.** `catalog:` for a package with no catalog entry is an error. Add it to the catalog first (`pnpm add --save-catalog <pkg>`), or use a direct range.
- **`strict` mode and transitive deps.** `catalogMode: strict` only constrains your workspace packages' direct `package.json` ranges, not transitive deps from the registry.
- **Renaming a catalog.** If you rename a named catalog, every `catalog:<oldname>` reference breaks. Use `catalogMode: prefer` to auto-rewrite, or sed across `package.json` files.
- **Catalogs vs. overrides.** Catalogs set the *requested* range for your workspace packages. Overrides *force* a version across the whole graph (including transitives). They're complementary: catalogs for "what we ask for," overrides for "what we get regardless of who asks."
- **One version per catalog entry.** A catalog entry is a single range. If you need two versions of `react` in the same monorepo, use named catalogs (above), not two entries in one catalog.

## References

- <https://pnpm.io/catalogs>

## Related

- Workspace file → [workspace.md](workspace.md)
- Lockfile `catalogs` and `importers` → [lockfile.md](lockfile.md)
- Overrides (complementary) → [overrides-patch.md](overrides-patch.md)
