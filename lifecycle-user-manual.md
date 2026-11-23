# Bindu Studio Lifecycle — Product Import & Activation Manual

This guide documents the **current** `bindu-studio lifecycle` command surface.

`lifecycle` is for **product artifacts and activation**.
It is not the runtime config/plugin rollout command path.

## Scope Boundaries

- Use `lifecycle` for:
  - importing update bundles (`import-update`)
  - activating root version (`activate-root`)
  - enqueueing node product activation (`activate-node`)
  - lifecycle status/versions/baseline/backup
- Use `runtime` for:
  - runtime desired snapshots
  - runtime config/plugin rollout (`runtime update wizard`)
- Use `plugin` for:
  - certify/apply/remove/list/who-uses plugin workflows

## Current Lifecycle Commands

Top-level under `bindu-studio lifecycle`:

- `status`
- `import-update`
- `activate-root`
- `activate-node`
- `list-versions`
- `versions`
- `baseline` (`status|diff|accept`)
- `backup` (`list|create`)

Hidden/internal commands are intentionally excluded from operator docs.

## Recipe A — Import Product Update Bundle

```bash
bindu-studio lifecycle import-update ./binduUpdate-<version>.tar.gz
```

What it does:

- imports root and node product artifacts into `releases/`
- imports plugin bundles in the pack as staged plugin artifacts (not assigned)

What it does not do:

- does not activate root automatically
- does not activate nodes automatically

## Recipe B — Activate Root

```bash
bindu-studio lifecycle activate-root --to <version>
# or default target resolution
bindu-studio lifecycle activate-root
```

Use `--yes` to skip interactive confirmation.

## Recipe C — Activate Nodes (Product)

```bash
# use current active root version
bindu-studio lifecycle activate-node --to active --yes --auth user:pass

# same meaning: --to defaults to active
bindu-studio lifecycle activate-node --yes --auth user:pass

# force specific imported version for selected nodes
bindu-studio lifecycle activate-node --to <version> --nodes node-a,node-b --yes --auth user:pass

# preview only
bindu-studio lifecycle activate-node --to active --nodes node-a,node-b --dry-run --json --auth user:pass
```

Notes:

- omitting `--to` means `--to active`
- `active` means the product version currently active on root
- `activate-node` enqueues async node activation intents.
- `activate-node` sends `NodeUpdate` only.
- `activate-node` never rolls out runtime config or plugins.
- `--dry-run` previews without enqueue.

## Recipe D — Inspect Product State

```bash
bindu-studio lifecycle status
bindu-studio lifecycle status --json

bindu-studio lifecycle list-versions
bindu-studio lifecycle versions --auth user:pass
```

## Recipe E — Baseline Management

```bash
bindu-studio runtime baseline status
bindu-studio runtime baseline diff
bindu-studio runtime baseline accept --yes
```

Use baseline commands to track mesh drift relative to last accepted snapshot.

## Recipe F — Backups (Root)

```bash
bindu-studio lifecycle backup list
bindu-studio lifecycle backup create --note "pre-upgrade"
```

This command group covers backup listing/creation only.

## Runtime Rollout (Config + Plugins)

Do **not** use `lifecycle` for runtime config/plugin rollout.
Use:

```bash
bindu-studio runtime desired commit --note "config/plugins rollout"
bindu-studio runtime update wizard --nodes <node-a,node-b> --yes
```

This sends `NodeRuntimeUpdate` messages and applies runtime deltas on nodes.

Important runtime update facts:
- This path is node-only; root is excluded.
- `runtime desired commit` materializes the config artifact automatically from the full root `runtime/config` directory.
- Runtime update can deliver config only, plugins only, or both together.
- Config is included only when node baseline differs from runtime desired baseline.
- After a successful runtime update apply, the node restarts so config and plugin state converge from a clean process start.
- Runtime update and node update failures emit `ErrorEvent` with `error_type=RUNTIME_UPDATE_ERROR`.
- Nodes also expose `config_version` through `SystemVersion` when available, and the mesh version modal shows it.

## Runtime Config Backup / Restore

Use this command family to snapshot and restore the full root `runtime/config` directory.

List available config backups:

```bash
bindu-studio runtime config-backup list
```

Create a config snapshot:

```bash
bindu-studio runtime config-backup create --note "before message type edits"
```

This stores a full snapshot under the configured backup destination in:

```text
config-backups/<timestamp>/
```

Restore a config snapshot:

```bash
bindu-studio runtime config-restore <backup-id>
```

By default restore is a dry-run. Execute the restore with:

```bash
bindu-studio runtime config-restore <backup-id> --execute --yes
```

Operational notes:
- restore replaces the full `runtime/config` directory, not individual files
- the current config directory is first moved aside to `runtime/config.pre-restore-<timestamp>`
- the replacement uses local directory move/copy semantics and is designed to behave atomically at the directory level
- these are local root operations; they do not require `--auth`

## Quick Recipes By Operation

### Product release: update root only

```bash
bindu-studio lifecycle import-update ./binduUpdate-<version>.tar.gz
bindu-studio lifecycle activate-root --to <version> --yes
```

### Product release: update all nodes to current active root version

```bash
bindu-studio lifecycle activate-node --yes --auth user:pass
```

### Product release: update selected nodes to an explicit version

```bash
bindu-studio lifecycle activate-node --to <version> --nodes node-a,node-b --yes --auth user:pass
```

### Runtime only: config rollout

```bash
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config-only rollout"
bindu-studio runtime update wizard --nodes node-a,node-b --yes
```

### Runtime only: plugin rollout

```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin apply --plugin my-plugin --target node --node node-a
bindu-studio plugin apply --plugin my-plugin --target node --node node-b
bindu-studio runtime desired commit --note "plugin-only rollout"
bindu-studio runtime update wizard --nodes node-a,node-b --yes
```

### Runtime only: config + plugins together

```bash
bindu-studio runtime baseline accept --yes
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin apply --plugin my-plugin --target node --node node-a
bindu-studio runtime desired commit --note "config+plugin rollout"
bindu-studio runtime update wizard --nodes node-a --yes
```

Rule of thumb:
- `activate-root` and `activate-node` are product release commands
- `runtime update wizard` is the runtime config/plugin command

## Troubleshooting

- `import-update` fails:
  - verify tarball path and rebuild the update bundle.
- `activate-root` not switching:
  - check `lifecycle list-versions`, then retry with explicit `--to`.
- `activate-node` enqueued but node not updated:
  - verify node presence/connectivity and auth credentials.
- update failed but only `runtime/runtime-info/last-nodeupdate.json` changed:
  - query recent `ErrorEvent` rows with `error_type=RUNTIME_UPDATE_ERROR` to inspect the failed stage and error payload.
- `activate-node` trying to roll out runtime config/plugins:
  - that is incorrect; use `runtime desired commit` plus `runtime update wizard` for runtime changes.
- Need config/plugin rollout instead of product rollout:
  - use `runtime desired ...` + `runtime update wizard`.
