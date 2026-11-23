# Bindu How-To Guide
Operational procedures for Updates, Plugins, and Node Installation using bindu-studio

This guide provides canonical, operator-level procedures for managing Bindu runtime, plugins, and node lifecycle using bindu-studio.

## Auth Contract

Most `bindu-studio` admin commands below require Basic auth against `bindu-root`.

Recommended shell setup:

```bash
export BINDU_STUDIO_AUTH=console:<password>
```

If you do not export it, add `--auth console:<password>` to the commands that talk to the root admin API.

Bindu follows a lifecycle model where:

- Root is the control plane
- Nodes are execution plane
- Artifacts are imported first, then activated explicitly
- Node rollout is asynchronous intent enqueue + eventual delivery

Canonical runtime layout and launcher contract:

- `<BaseDir>/bindu-root` and `<BaseDir>/bindu-node` are stable launchers.
- Launchers read `<BaseDir>/runtime/active.json`.
- Launchers exec runtime binaries from `<BaseDir>/releases/<version>/...`.
- Runtime state lives under `<BaseDir>/runtime/...` only.
- Release payloads are self-contained per version, including schemas.

Important version note:

- `./bindu-root --version` and `./bindu-node --version` print launcher identity, not the active runtime release.
- The active runtime release is the one pointed by `<BaseDir>/runtime/active.json`.
- To validate runtime version, run the binary under `releases/<active_version>/` directly.

All operations are driven from the root unless explicitly stated otherwise.

## Three Operational Processes (Do Not Mix)

1) Initial installation
- Install root and first node bundle.
- Use `mesh node-deploy` + node installer for first node provisioning.
- For plugins on fresh nodes, assign/import the plugin first, then build installer bundles with `mesh node-deploy`.

2) Product update (binary/runtime version)
- Goal: update `bindu-root` / `bindu-node` product version.
- Use `lifecycle import-update` + `activate-root` + `activate-node`.
- Message path to nodes: `NodeUpdate`.

3) Runtime update (config + plugins)
- Goal: update runtime components on already deployed nodes.
- Use `runtime update wizard`.
- Message path to nodes: `NodeRuntimeUpdate`.
- This path does not perform product binary rollout.
- The interactive `plugin deploy` wizard belongs here: it is a rollout path for nodes that are already installed.

---

# 1. How to apply a node binary update bundle (OTA)

## Goal

Apply a node binary/runtime release update bundle with explicit activation.

## Prerequisites

- Update bundle already built (example: update-bundle-0.0.6.tar.gz)
- You are on the root machine
- bindu-studio is available

Build note (wizard):

```
./scripts/build-update-bundle.sh --wizard --from-source
```

In wizard mode, CORE is always included (`bindu-root + public/console + node bundles`).
`tools/studio/*` is optional extra payload.

## Step 1 — Import update bundle

Run on root:

bindu-studio lifecycle import-update /path/to/binduUpdate-<version>.tar.gz

Installed-root recipe (explicit order, using local `./bin` path):

```bash
./bin/bindu-studio lifecycle import-update ~/binduUpdate-<version>.tar.gz --auth console:<password>
./bin/bindu-studio lifecycle activate-root --auth console:<password>
./bin/bindu-studio lifecycle activate-node --to active --auth console:<password>
```

Important order:
- Activate root before activating nodes.
- `activate-root` switches active runtime and triggers root restart.
- If root is installed as a service, restart is automatic by the service manager.
- If root is running manually, restart/relaunch root before `activate-node`.

This stages root and node bundles for activation.
If the update pack contains plugin bundles, they are imported as staged candidates (not deployed/activated).

## Step 2 — Activate root (wizard)

Run on root:

bindu-studio lifecycle activate-root

If `--to` is omitted, Studio opens a version picker (newest-first), suggests the latest imported version, and asks for confirmation.

Non-interactive:

bindu-studio lifecycle activate-root --to <version> --yes

## Step 3 — Activate nodes (default: root active)

Run on root:

bindu-studio lifecycle activate-node

If `--to` is omitted, target version defaults to root active (`--to active`).
Studio prints a summary before enqueue:

- root active version
- target version
- target nodes (`all` unless `--nodes` is set)
- dry-run mode

Then it asks for confirmation.

Non-interactive:

bindu-studio lifecycle activate-node --to active --yes

Optional explicit version/targets:

bindu-studio lifecycle activate-node --to <version> --nodes node-1,node-2 --yes

Canonical semantics:

- update send is intent enqueue, not delivery
- no presence gating
- no skip

Nodes then:

- receive `NodeUpdate` control message via MQTT (mTLS)
- parse target artifact location (`delivery.base_url` + `artifact_path`) and expected SHA-256
- pull binary package out-of-band via HTTPS over mTLS (`GET /artifacts/...`)
- verify SHA-256 hash
- restart
- emit new MessageState

> [!NOTE]
> **Transport Architecture**: The update command (`NodeUpdate`) travels through the **MQTT control plane**, while the actual heavy binary archive is fetched point-to-point via the **HTTPS data plane** (`/artifacts/...`). Heavy binaries are never transmitted over the MQTT message broker.

This flow sends `NodeUpdate` (binary OTA).  
For config/plugins runtime components, use the runtime update wizard in section 2.

## Step 4 — Verify update

Check node log:

journalctl -u bindu-node -f


or manual run:

./bindu-node


Look for:

runtime.identity role=node version=...


Or check MessageState in Console.

You can also run:

bindu-studio lifecycle status
bindu-studio lifecycle list-versions

---

# 2. How to update node runtime components (config + plugins)

## Goal

Deliver runtime component changes to nodes:

- runtime config from root runtime desired state
- plugins already deployed/assigned from root
- no node binary/version rollout

## Required before running the wizard

You should run `runtime update wizard` only after runtime desired state reflects your intent:

- Root is running and reachable by `bindu-studio` with valid auth.
- Target nodes already exist in mesh (non-root nodes only).
- Plugin code changes were already published on root (`plugin apply --bundle ...`) and assigned to targets (`plugin apply --plugin ... --target node ...`) when needed.
- Config changes were already committed into runtime desired state.

If runtime desired has no delta for a node, the planner returns `no-changes` for that node and nothing is enqueued for it.

What `runtime update wizard` decides automatically:

1) Config update
- Included only if node baseline differs from runtime desired baseline.
- Not forced on every run.

2) Plugin update
- Plugin list is computed from runtime desired state + mesh assignments for each node.
- Manual plugin selection is not part of this wizard.
- If plugin candidate/version is already aligned on a node, it is skipped.

3) Target nodes
- Uses `--nodes` if provided.
- Otherwise uses interactive selection or all eligible non-root nodes.
- Sends `NodeRuntimeUpdate` intents only to selected nodes.

## Step 1 — Prepare runtime desired state

Ensure runtime desired reflects what nodes should run:

- Config changes committed in runtime desired snapshot.
- Plugin versions/assignments already managed with `bindu-studio plugin ...`.
- `runtime desired commit` now materializes the runtime config artifact automatically under `runtime/artifacts/config/<baseline>/config.tar.gz`, so config-only runtime updates do not require manual staging.
- The config artifact is built from the full root `runtime/config` directory, not only from `mesh.json` or `messageTypesConfiguration.json`.

Optional check:

```
bindu-studio runtime desired show
```

## Runtime Config Backup / Restore

Snapshot the current root `runtime/config` tree before editing:

```bash
bindu-studio runtime config-backup create --note "before config changes"
```

List snapshots:

```bash
bindu-studio runtime config-backup list
```

Preview a restore:

```bash
bindu-studio runtime config-restore <backup-id>
```

Apply a restore:

```bash
bindu-studio runtime config-restore <backup-id> --execute --yes
```

These commands are local to root and do not require `--auth`.

Commit runtime desired snapshot:

```
bindu-studio runtime desired commit --note "config/plugins rollout"
```

Config-only rollout sequence:

```
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config-only rollout"
bindu-studio runtime update wizard --nodes <node-a,node-b> --yes
```

Config + plugin rollout sequence:

```
bindu-studio plugin import ./my-plugin.tar.gz
bindu-studio plugin register --plugin my-plugin --target node --node <node-id>
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config+plugins rollout"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

## Step 2 — Send runtime update intents

Run on root:

```
bindu-studio runtime update wizard
```

What happens when you type `APPLY`:

1. Studio asks for confirmation: `Type APPLY to confirm runtime update delivery`.
2. After `APPLY`, Studio resolves target nodes from mesh:
   - if `--nodes` was provided: uses those node IDs.
   - if `--nodes` was omitted and terminal is interactive: prompts `all eligible nodes` or manual selection.
   - root is excluded automatically.
3. Studio sends `NodeRuntimeUpdate` intents to root API (`/api/v1/lifecycle/runtime-updates/send`).
4. Root enqueues updates asynchronously; node apply/restart happens later on node side.

Common flag variants:

```
bindu-studio runtime update wizard --desired-id <desired-id> --yes
bindu-studio runtime update wizard --dry-run --json
```

Flag semantics:

- `--desired-id <id>`: use that exact runtime desired snapshot ID instead of current runtime desired.
- `--yes`: skip the `Type APPLY` prompt (useful for non-interactive/CI runs).
- `--dry-run`: compute planning only; do not enqueue any runtime update.
- `--json`: print machine-readable output only.

Non-interactive note:
- If command runs without TTY and without `--yes` (and not `--dry-run`), Studio aborts with `confirmation required; rerun with --yes`.

Useful additions:

```
bindu-studio runtime update wizard --nodes <node-a,node-b> --yes
bindu-studio runtime update wizard --desired-id <id> --nodes <node-a,node-b> --yes --json
```

Canonical semantics:

- message type: `NodeRuntimeUpdate`
- runtime-only payload (config/plugins), no node binary bundle
- root is never a runtime-update target; this path is node-only
- enqueue intent first, delivery is asynchronous
- node applies when artifact arrives (download/verify/apply/restart)
- `config` is included only when the node-reported baseline differs from runtime desired baseline
- `plugins` are included only when mesh assignment + desired candidate/version differ from the node state
- `remove_plugins` is included when a plugin is no longer assigned to that node but is still materialized on disk
- when a plugin rollout changes code/version, the node converges to the target version and prunes older runtime/staged versions of that same plugin ID
- if only config differs, the plan is config-only
- if only plugins differ, the plan is plugin-only
- if both differ, both travel in the same `NodeRuntimeUpdate`
- runtime removal cleans the node plugin runtime directory and purges staged candidates for that plugin

## Step 3 — Verify runtime update

Check node logs for runtime apply/restart cycle and then confirm steady state:

```
journalctl -u bindu-node -f
bindu-studio lifecycle status
```

`SystemVersion`/`NodePresence` should resume normally after restart.

---

# 3. How to write a plugin

## Goal

Create a new plugin bundle ready for deployment.

## Canonical sequence (root + nodes + plugins)

Use this exact order for first-time rollout where plugins must run on root and on new nodes:

1. Install and start root.
2. Add node(s) in mesh and apply:
   - `bindu-studio mesh add`
   - `bindu-studio mesh apply`
3. Create plugin and validate:
   - `bindu-studio plugin init ...`
   - `bindu-studio plugin doctor ...`
4. Certify with platform coverage:
   - `bindu-studio plugin certify ... --targets-from-mesh --target $(go env GOOS)/$(go env GOARCH)`
5. Publish plugin code to root:
   - `bindu-studio plugin apply --bundle ... --target root`
6. Assign plugin to node(s):
   - `bindu-studio plugin apply --plugin <id> --target node --node <node-id>`
7. Build node installer bundles:
   - `bindu-studio mesh node-deploy --nodes <node-id-1>,<node-id-2>`
8. Install/start each node from generated bundle.
   - Assigned plugins start automatically on first node startup.
   - For already deployed nodes, use `bindu-studio runtime update wizard --nodes ... --yes`.

## Step 1 — Create plugin project

Run:

bindu-studio plugin init --profile ai-safe


Follow prompts to create plugin skeleton.

Or use non-interactive:

bindu-studio plugin init --no-interactive --id <plugin-id> --out <MessageTypeOut>


This creates a plugin source directory named `<plugin-id>`.

## Step 2 — Implement plugin logic

Modify plugin source code according to plugin type.

Plugin typically:

- emits messages
- consumes messages
- performs scheduled work

## Step 3 — Certify plugin (build bundle)

Run:

bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz

Target flags:
- `--target` uses `os/arch` (slash syntax), repeatable:
  - `--target linux/amd64 --target linux/arm64 --target linux/armv7`
- `--targets-from-mesh` includes only platforms present in mesh (not all Linux permutations).
- For Raspberry Pi 32-bit OS, use canonical `linux/armv7` (legacy `linux/arm` is compatibility alias).

Recommended for mixed root/node deployments:

bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --targets-from-mesh

Recommended before certify:

bindu-studio plugin doctor --source ./my-plugin --pre-release --strict


This produces deployable plugin bundle.

---

# 4. How to deploy plugin to root

## Goal

Deploy plugin so it runs on root.

## Step 1 — Apply plugin bundle to root

Run on root:

bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root


This imports, activates, and assigns the plugin to root.

## Step 2 — Verify

Run:

bindu-studio plugin list


Or check root log:

plugin.start.enabled plugin=<plugin-id>


---

# 5. How to deploy plugin to a node for the first time

## Goal

Deploy plugin so it runs on a specific node.

## Step 1 — Ensure plugin code is active in root repository

Run on root:

bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root

## Step 2 — Assign plugin to node (assignment only)

Run on root:

bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>

## Step 3 — Deliver plugin to node runtime

For already deployed nodes:

bindu-studio runtime update wizard --nodes <node-id> --yes

For nodes not installed yet:

- keep the assignment in mesh
- build node bundle with `bindu-studio mesh node-deploy --nodes <node-id>`
- install with `bindu-install-node`

Node will automatically:

- download plugin
- start plugin

## Step 4 — Verify

Check node log:

plugin.start.enabled plugin=<plugin-id>


Or inspect MessageState:

fields.lifecycle.plugins_deployed


---

# 6. How to update a plugin in root

## Goal

Update plugin code already active in root, keeping assignments.

## Step 1 — Re-certify from updated source

Run (transparent replacement flow):

bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json

If plugin is (or will be) assigned to nodes, include mesh platforms:

bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH) \
  --json


## Step 2 — Re-apply bundle to root

Run:

bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin who-uses --plugin my-plugin

Expected behavior:
- same plugin `id` is replaced with new active candidate
- existing root/node assignments stay
- old non-active staged candidates are auto-cleaned
- `remove` / `undeploy` are not required for normal code update


---

# 7. How to update a plugin in a node

## Goal

Upgrade plugin running on node.

## Step 1 — Publish new plugin code once

Run on root:

bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root

## Step 2 — Keep/add assignment for the node

bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>

## Step 3 — Roll out runtime update to the node

bindu-studio runtime update wizard --nodes <node-id> --yes

Node will converge automatically.

## Step 4 — Verify

Check node MessageState:

fields.lifecycle.plugins_deployed

If `mesh node-deploy` fails with missing Linux runtime target:

plugin <id>@<version> missing runtime target linux/amd64 (runtime targets: linux/arm64)

re-run certify with:

bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH)

then apply again and retry node-deploy.


---

# 7B. Release vs Runtime update cheat sheet

Use this when you only need the right command family.

## Update `bindu-root`

```bash
bindu-studio lifecycle import-update ./binduUpdate-<version>.tar.gz
bindu-studio lifecycle activate-root --to <version> --yes
```

## Update `bindu-node` release on all nodes

```bash
bindu-studio lifecycle activate-node --yes --auth user:pass
```

Equivalent explicit form:

```bash
bindu-studio lifecycle activate-node --to active --yes --auth user:pass
```

## Update `bindu-node` release on selected nodes

```bash
bindu-studio lifecycle activate-node --to <version> --nodes node-a,node-b --yes --auth user:pass
```

## Roll out runtime config only

```bash
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config-only rollout"
bindu-studio runtime update wizard --nodes node-a,node-b --yes
```

## Roll out plugins only

```bash
bindu-studio runtime desired commit --note "plugin rollout"
bindu-studio runtime update wizard --nodes node-a,node-b --yes
```

## Roll out config and plugins together

```bash
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config+plugin rollout"
bindu-studio runtime update wizard --nodes node-a,node-b --yes
```

Rule:
- `lifecycle activate-node` = release/product OTA only
- `runtime update wizard` = runtime config/plugins only


---

# 8. How to install Bindu in a node

## Goal

Install bindu-node on a machine and connect to root.

## Prerequisites

- Root running
- Root IP known
- Installer binary available

Example:

bindu-install-node


## Step 1 — Run installer

Run on node machine:

sudo ./bindu-install-node --signing-url https://<root-ip>:8443/api/v1/tls/sign-node --install-service


Follow prompts.

Installer will:

- create TLS identity
- install runtime
- configure connection to root

## Step 2 — Start node

Run:

sudo systemctl start bindu-node


Or manual:

./bindu-node


## Step 3 — Verify

Check log:

NodePresence emitted


Or on root:

bindu-studio mesh tree --all


Node should appear (use mesh tree):

bindu-studio mesh tree --all

---

# 9. How to self-update bindu-studio (studio-only)

## Goal

Update only `bindu-studio` without running `activate-root` or `activate-node`.

## Step 1 — Build studio self-update bundle

Run on build machine:

```
./scripts/build-studio-update-bundle.sh --version <version>
```

Important:
- Default source is `dist/root-installer-current/tools/studio`.
- If you only ran `build-tools.sh`, prefer explicit source:

```
./scripts/build-studio-update-bundle.sh --version <version> --source ./bin/tools/studio
```

- Or refresh default source first:

```
./scripts/build-root-installer-bundle.sh --release-label <version>
./scripts/build-studio-update-bundle.sh --version <version>
```

Optional:

```
./scripts/build-studio-update-bundle.sh --version <version> --target linux-amd64 --target linux-arm64
```

Default output:

```
dist/updates/binduStudioUpdate-<version>.tar.gz
```

## Step 2 — Import and apply self-update

Run on root machine:

```
bindu-studio self-update import ./dist/updates/binduStudioUpdate-<version>.tar.gz
```

This validates the current platform payload and atomically replaces:

```
<BaseDir>/bin/bindu-studio
```

No root/node runtime activation is required for this flow.

---

# 10. How to delete a plugin completely

## Goal

Remove a plugin version from:

- mesh assignments
- runtime materialization (`plugins.d/bpr`)
- staged plugin artifacts

## Canonical sequence (by id)

Run on root:

```bash
bindu-studio plugin who-uses --plugin my-plugin
bindu-studio plugin remove --plugin my-plugin --target all --uninstall --purge
```

## Notes

- `remove` without `--uninstall --purge` only removes assignments.
- Candidate-level cleanup commands are internal/advanced and hidden from normal CLI help.

---

# Troubleshooting

## Node not updating

Check:

bindu-studio lifecycle status

Look for an explicit skip reason (for example: missing_candidate_artifact, missing_artifact_sha, already_on_version, no_presence).

Then check:

bindu-node log

Verify connectivity to root.

## Plugin not running

Check:

bindu-studio plugin list


Verify plugin registered.

## Node not visible

Check:

NodePresence messages


Ensure TLS and network connectivity.

---

# Summary

bindu-studio manages:

- runtime lifecycle
- plugin lifecycle
- node lifecycle

Root defines desired state.

Nodes converge automatically.

Bindu maintains deterministic, observable system state.
