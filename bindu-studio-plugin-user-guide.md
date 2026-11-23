# Bindu Studio Plugin User Guide (Canonical)

## UX Contract (Single Model)

Bindu Studio exposes one operator model for plugins:
- Build bundle: `plugin certify`
- Deploy plugin: `plugin apply` (or interactive `plugin deploy`)
- Remove plugin: `plugin remove` (or interactive `plugin undeploy`)

Low-level lifecycle commands remain internal/hidden for compatibility and are not part of normal operator UX.

Important execution split:
- `plugin deploy` is the interactive rollout path for nodes that are already installed.
- `mesh node-deploy` is the first-install/provisioning path for nodes that do not exist yet or are being reinstalled from zero.
- If your goal is "import plugin into root, assign it in mesh, then build installer bundles with plugins included", use `plugin apply` or `plugin import` + `plugin register`, then `mesh node-deploy`.
- Do not treat `plugin deploy` as the primary provisioning path for fresh node installation.

## Auth Contract

Most `bindu-studio` commands in this guide talk to `bindu-root` over the admin API and therefore require Basic auth.

Use one of these two patterns:

```bash
export BINDU_STUDIO_AUTH=console:<password>
```

or pass auth explicitly per command:

```bash
--auth console:<password>
```

If an example below omits `--auth`, assume `BINDU_STUDIO_AUTH` is already exported.

## Intent-First Model (Operator)

Think in intentions, not internals:

1) Deploy new plugin
- `plugin init` -> `plugin certify` -> `plugin apply --bundle ...`

2) Update existing plugin (same `id`)
- `plugin certify` -> `plugin apply --bundle ...`
- `apply` replaces runtime for that `id` and keeps current assignments.
- Old staged candidates for that plugin are auto-cleaned (active candidate is preserved).

3) Assign existing plugin to more targets
- `plugin apply --plugin <id> --target root|node|all ...`
- Assignment only, no code change.

Important nuance:
- `plugin apply --bundle ...` is a root deployment action. It may activate and run the plugin on root.
- `plugin activate` is also a root deployment action.
- For node-only plugins that must not run on root, use `plugin import` + `plugin register` + runtime rollout.

4) Remove plugin permanently
- `plugin remove --plugin <id> --target all --uninstall --purge`

## Recommended Flow

1) Scaffold
```bash
bindu-studio plugin init --profile ai-safe
```

### Interactive `plugin init` (what it is and what it is not)

`plugin init` interactive mode is a **scaffold generator**.  
Its goal is to produce a valid starting project (contract + sample logic), not a release-final implementation.

Wizard sequence:
1. Plugin identity (`id`, optional name, version, output directory)
2. Message contract selection (input/output message types from registry)
3. Optional README query notes + optional read-only query SDK scaffold
4. Optional cron schedules (preset or manual cron)
5. File generation under the selected output directory

Expected output of interactive init:
- `plugin.json` with declared `in/out` and optional schedules
- `main.go` with generated author zones
- `README.md`
- `go.mod`
- `examples/in` and `examples/out`

Important:
- Generated `main.go` can include `TODO` markers in author zones by design.
- This is expected for a scaffold and means "implement decision here".

2) Validate
```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
```

### Why `doctor --pre-release --strict` may fail right after init

If scaffold `main.go` still contains `TODO` markers:
- `--pre-release` reports `W_PRERELEASE_TODO_MARKERS`
- `--strict` promotes warnings to errors
- Result: doctor fails until TODO blocks are replaced

Practical sequence after interactive init:
1. Run non-strict doctor first:
```bash
bindu-studio plugin doctor --source ./my-plugin
```
2. Replace TODO sections in `main.go` with explicit logic.
3. Run release gate:
```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
```
4. Then certify/apply.

Wizard input note:
- In `Add another schedule? (y/N)`, only `y/yes` continues.
- Numeric input like `2` is treated as "no".

3) Certify
```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json
```

### Certify targets (important)

- `--target` format is `os/arch` (with `/`, not comma).
- `--target` is repeatable (one platform per flag).
- `--targets-from-mesh` includes exactly the platforms declared in current `mesh.json`.
- `--targets-from-mesh` does **not** mean “all possible Linux platforms”; it only includes platforms present in mesh nodes.
- For Raspberry Pi 32-bit OS, use `linux/armv7` (canonical target). Legacy `linux/arm` is treated as compatibility alias.
- If no target flags are provided, certify builds only for current host (`GOOS/GOARCH`).

For mixed platforms (root + nodes), certify with explicit or mesh-derived targets:
```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh
```

Examples:

```bash
# explicit targets
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --target linux/amd64 \
  --target linux/arm64 \
  --target linux/armv7

# mesh targets + keep host target for local root runs (example: mac host)
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH)
```

### Semantic Closure & New Message Types (`--config`)

`bindu-studio plugin certify` strictly validates semantic closure: every message type declared in `plugin.json` (`out` and `in`) must be registered with a valid JSON Schema.

- **Default behavior**: `plugin certify` resolves schemas from the active runtime file (`runtime/config/messageTypesConfiguration.json`).
- **New or offline schemas**: When authoring a new MAS solution where the plugin emits brand-new message types defined in a local `poc_message_types.json` that has not yet been merged into the cluster runtime, `plugin certify` will fail with an `unknown message type` error unless pointed to the local file.
- **Solution**: Pass the `--config` flag pointing to your local schema definitions:
  ```bash
  bindu-studio plugin certify \
    --source ./my-plugin \
    --out ./my-plugin.tar.gz \
    --target linux/amd64 \
    --config path/to/poc_message_types.json
  ```
- **Workspace Resolution**: In developer workspaces or CI where `bindu-studio` is executed directly via `go run ./cmd/bindu-studio`, pass `--base-dir .` alongside `--config` so the tool can locate supporting runtime schema assets:
  ```bash
  go run ./cmd/bindu-studio plugin certify \
    --base-dir . \
    --source ./my-plugin \
    --config ./poc_message_types.json \
    --dry-run
  ```

4) Deploy (root, only if root must run the plugin)
```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```

Root operational note:
- if `bindu-root` is not managed as a service, restart/relaunch it manually after a root-targeted plugin apply so the active root runtime is in a known converged state.
- if `bindu-root` runs as a service, prefer the service-managed path in production.

5) Assign (specific node)
```bash
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
```

6) Inspect
```bash
bindu-studio plugin list
bindu-studio plugin who-uses --plugin my-plugin
```

## Message Samples (Studio Admin Workflow)

`message-samples` is a Studio operator workflow for:
- exporting persisted examples from SQL through `POST /api/v1/messages/sql`
- replaying those examples through `bindu/in` over the console MQTT WebSocket listener

Commands:

```bash
bindu-studio message-samples export --auth console:SuperSecret1
bindu-studio message-samples export --auth console:SuperSecret1 --types HelloWorld,HelloWorldStats
bindu-studio message-samples import --auth console:SuperSecret1
bindu-studio message-samples export --auth console:SuperSecret1 --dir ./message-samples --file message-samples.json
```

Notes:
- `export` uses the read-only SQL API only.
- `import` is Studio/admin tooling; it uses Studio auth plus `/api/v1/mqtt/credentials` to obtain MQTT WebSocket connection details and then republishes cleaned messages to `bindu/in`.
- replayed messages have server-managed fields removed before publish, including `$id` and `$arriveTime`.
- default sample file path is `./message-samples/message-samples.json` and can be changed with `--dir` / `--file`.

7) Remove
```bash
bindu-studio plugin remove --plugin my-plugin --target all --uninstall --purge
```

## Canonical End-to-End Sequence (Root + Nodes + Plugins)

Use this sequence when plugins must run on root and on new nodes.

1) Install and start root.

2) Add nodes to mesh and apply:
```bash
bindu-studio mesh add
bindu-studio mesh apply
bindu-studio mesh tree --all
```

3) Create plugin source:
```bash
bindu-studio plugin init --profile ai-safe --id my-plugin --outdir ./my-plugin
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
```

4) Certify plugin for node platforms (and host when needed):
```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH)
```

5) Publish plugin code in root:
```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```

6) Assign plugin to target nodes:
```bash
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id-1>
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id-2>
```

7) Build node install bundles:
```bash
bindu-studio mesh node-deploy --nodes <node-id-1>,<node-id-2> --out ./dist/nodes
```

8) Install each node with its bundle and start node runtime.
   - On first install, assigned plugins are included and start with node runtime.
   - For already deployed nodes, use runtime rollout:
```bash
bindu-studio runtime update wizard --nodes <node-id-1,node-id-2> --yes
```

Operator rule of thumb:
- New node or reinstall from zero: finish with `mesh node-deploy`.
- Existing installed node: finish with `runtime update wizard`.

## Canonical End-to-End Sequence (Node-Only Plugin, Root Must Not Run It)

Use this sequence when the plugin is delivered by root but must execute only on nodes.

1) Certify for the node platforms you need:
```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --target linux/amd64 \
  --target linux/armv7
```

2) Import the certified bundle into root artifacts only:
```bash
bindu-studio plugin import ./my-plugin.tar.gz
```

3) Register the plugin on target nodes without activating it on root:
```bash
bindu-studio plugin register --plugin my-plugin --target node --node <node-id-1>
bindu-studio plugin register --plugin my-plugin --target node --node <node-id-2>
```

4) Commit runtime desired state:
```bash
bindu-studio runtime desired commit --note "assign my-plugin to selected nodes"
```

5) Deliver runtime update only to those nodes:
```bash
bindu-studio runtime update wizard --nodes <node-id-1,node-id-2> --yes
```

Notes:
- Do not use `plugin activate` for node-only plugins; it deploys on root.
- Do not use `plugin apply --bundle ... --target root` for node-only plugins; it is a root deployment path.
- `plugin register` changes mesh assignment only; it does not execute the plugin.
- `plugin import` is the import-only path when root must store the certified bundle but must not run it.
- `plugin apply --bundle ...` is a combined path: it imports the bundle and may activate that candidate on root before writing mesh assignment.
- In `plugin apply`, `--target` controls mesh assignment scope. It does not convert `--bundle` into an import-only operation.
- `plugin deploy` is the guided rollout path after nodes already exist and `bindu-node` is already installed.

## Why Root Should Run As A Service In Production

Production root should run under a service manager.

Reasons:
- `activate-root` relies on service-managed restart for clean product activation.
- root-targeted plugin deployment is operationally safer when the root process is supervised and restartable.
- the mesh control plane, console, artifact serving, and node update APIs are long-running infrastructure services and should recover automatically after crashes or host reboot.
- service supervision gives one consistent operator surface for status, start, stop, and restart (`bindu-studio root-service ...`).
- manual root processes are easy to forget to relaunch, which creates avoidable downtime for nodes, console users, and runtime rollout operations.

## Operational Recipes (Node-Only Plugin Tutorial)

Use these exact recipes for tutorial plugins such as `bindu-beacon-gateway`.

### A) Assign and deploy to one node only

```bash
bindu-studio plugin import ./my-plugin.tar.gz
bindu-studio plugin register --plugin my-plugin --target node --node <node-id>
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "assign my-plugin to <node-id>"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

### A.1) Root does not have the plugin yet, but root must distribute it to a node without running it

Use this exact sequence when root needs the certified bundle in artifacts, but the plugin must execute only on the node:

```bash
bindu-studio plugin import ./my-plugin.tar.gz
bindu-studio plugin register --plugin my-plugin --target node --node <node-id>
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "import my-plugin and assign to <node-id>"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

Why this is the correct path:
- `plugin import` makes the bundle available on root for later node rollout.
- `plugin register` updates mesh assignment only.
- `runtime update wizard` delivers the assigned plugin to the already-installed node.
- `plugin apply --bundle ...` is not the import-only path; it may also activate the bundle on root.

Operational note:
- `plugin register` updates `runtime/config/mesh.json`.
- If Studio reports `runtime config drift detected (mesh.json)`, accept the new runtime baseline before `runtime desired commit`.
- The canonical command is `bindu-studio runtime baseline accept --yes`.

### B) Expand from one node to two nodes

If the plugin is already assigned to node A and must also run on node B:

```bash
bindu-studio plugin register --plugin my-plugin --target node --node <node-id-b>
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "assign my-plugin to node A and node B"
bindu-studio runtime update wizard --nodes <node-id-b> --yes
```

Use `runtime update wizard --nodes <node-id-a,node-id-b> --yes` if you want to re-rollout both.

### C) Retry rollout only on one node

If mesh assignment is already correct and only delivery/apply failed on one node:

```bash
bindu-studio runtime update wizard --nodes <node-id> --yes
```

No new `runtime desired commit` is needed unless mesh assignment changed since the last commit.

### C.0) Fleet rollout recipe for already-installed gateway nodes

Use this when the plugin must be distributed to many gateway nodes and you want one repeatable operator sequence.

1) Choose the target gateways.
- Use a comma-separated node list when you want a subset.
- Use `all` only if every eligible runtime node must converge now.

2) If root must store the new certified bundle but must not run it, import it:

```bash
bindu-studio plugin import ./my-plugin.tar.gz
```

3) Register the plugin on each selected gateway node:

```bash
for node in bindu-pi3-uno bindu-pi3-dos dell-debian1; do
  bindu-studio plugin register --plugin my-plugin --target node --node "$node"
done
```

4) Accept the runtime baseline after the mesh assignment change:

```bash
bindu-studio runtime baseline accept --yes
```

5) Commit the desired runtime snapshot:

```bash
bindu-studio runtime desired commit --note "rollout my-plugin to selected gateways"
```

6) Preview the rollout plan before delivery:

```bash
bindu-studio runtime update wizard --nodes bindu-pi3-uno,bindu-pi3-dos,dell-debian1 --dry-run --json
```

7) Deliver to the selected gateways:

```bash
bindu-studio runtime update wizard --nodes bindu-pi3-uno,bindu-pi3-dos,dell-debian1 --yes
```

8) If every eligible runtime node must receive the same desired snapshot, use `all`:

```bash
bindu-studio runtime update wizard --yes
```

Why this matters:
- This sequence is correct, but for large fleets it is too easy to get wrong by memory alone.
- The operator has to remember import vs apply, register, baseline accept, desired commit, dry-run, and delivery scope.
- For fleets of gateways/drones/trackers, the recommended UX is a dedicated wizard that:
  - lists eligible gateway nodes
  - allows `all` or a comma-separated selection
  - shows ineligible nodes and reasons such as `platform-missing`
  - asks for yes/no confirmation before baseline accept, desired commit, and rollout
  - encapsulates the manual command chain above
- Until that dedicated wizard exists, use this recipe exactly as written.

### C.1) Roll out config only

If only `runtime/config` changed on root and plugin assignment is already correct:

```bash
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config-only rollout"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

Notes:
- `runtime desired commit` materializes the config artifact automatically from the full root `runtime/config` directory.
- The same runtime update path is used; there is no separate config-only command.
- Root is not a runtime-update target.

### C.1.a) Back up and restore runtime config locally

Before risky config edits, take a local snapshot of `runtime/config`:

```bash
bindu-studio runtime config-backup create --note "before config rollout"
```

List available snapshots:

```bash
bindu-studio runtime config-backup list
```

Preview a restore:

```bash
bindu-studio runtime config-restore <backup-id>
```

Execute the restore:

```bash
bindu-studio runtime config-restore <backup-id> --execute --yes
```

Notes:
- these commands are local and do not require `--auth`
- restore replaces the full `runtime/config` directory
- the previous config tree is preserved as `runtime/config.pre-restore-<timestamp>`

### D) Remove from one node, keep on another

```bash
bindu-studio plugin remove --plugin my-plugin --target node --node <node-id-to-remove>
bindu-studio runtime desired commit --note "remove my-plugin from one node"
bindu-studio runtime update wizard --nodes <node-id-to-remove> --yes
```

Runtime effect on the node:

- the plugin is unassigned in mesh
- the next `NodeRuntimeUpdate` removes the runtime-materialized plugin directory from the node
- staged candidates for that plugin are also purged on that node

Then verify:

```bash
bindu-studio plugin who-uses --plugin my-plugin
```

### E) Remove from all nodes and purge staged candidates on root

```bash
bindu-studio plugin remove --plugin my-plugin --target all --uninstall --purge
bindu-studio runtime desired commit --note "remove my-plugin from all nodes"
bindu-studio runtime update wizard --nodes <node-id-1,node-id-2> --yes
```

`--purge` removes staged candidates from root artifacts; it does not SSH into nodes.

When a plugin rollout updates code to a newer version on a node, the node now keeps the deployed target version as the single runtime-materialized version for that plugin ID and prunes older runtime/staged versions for that plugin on the node.

### F) Verify assignment vs delivery

Use:

```bash
bindu-studio plugin who-uses --plugin my-plugin
```

Interpretation:

- `who-uses` shows mesh assignment only.
- Successful deploy is confirmed on the node by `nodeupdate.apply.plugin.ok`, `bpr.plugin.spawned`, and `nodeupdate.apply.ok`.
- A plugin can be assigned to two nodes while only one of them has completed rollout.

## Artifact Retention and Cleanup

This is the current behavior, as implemented today.

### On nodes

- Download temp directories under `runtime/artifacts/tmp/nodeupdate/*` are removed after successful node update execution.
- Downloaded `.tar.gz` files used during node update live inside that temp directory, so they disappear when the temp directory is removed.
- Staged plugin candidates under `runtime/artifacts/plugins/<id>/<version>/<candidate>/` are kept after deploy, but maintenance now prunes old residue there too.
- Active deployed plugins live under `runtime/plugins.d/bpr/<id>/<version>/`.
- Runtime update plugin backups may exist under `runtime/plugins.d/bpr/.bak/<id>/<version>/<timestamp>/`; maintenance keeps the latest 3 per plugin/version.
- Runtime update config backups may exist as `runtime/config.pre-update-<timestamp>`; maintenance keeps the latest 3.
- Runtime plugin candidate artifacts under `runtime/artifacts/plugins/<id>/<version>/<candidate>/` are pruned by maintenance down to the latest 3 per plugin, while preserving the active deployed candidate and candidates still referenced by current/recent runtime desired snapshots.
- Stale temp dirs left by interrupted runtime updates under `runtime/artifacts/tmp/nodeupdate/nodeupdate-*` are pruned by maintenance after 24h.
- Old product releases under `releases/<version>/` are retained with release retention:
  - keep current active release
  - keep previous activated releases up to a total of 3

### On root

- Root lazily materializes and serves:
  - `runtime/artifacts/node/<os>/<arch>/<version>.tar.gz`
  - `runtime/artifacts/plugins/<plugin-id>/<version>/<candidate>.tar.gz`
- Those root `.tar.gz` files are not removed immediately after a node downloads them.
- Old staged plugin candidates on root can be removed explicitly with:
  - `plugin remove --purge`
  - `plugin purge`
- Product release directories are bounded by release retention, but runtime artifact tarballs do not yet have a general retention policy here.

### Practical implication

- Node-update temp downloads are ephemeral.
- Active plugin artifacts remain persistent, but staged runtime-update residue is now bounded by maintenance retention.
- Old product releases are bounded.
- Old runtime artifact tarballs and plugin candidate caches are not fully auto-pruned yet; treat them as retained operational cache unless you purge them.

## Quickstart TL;DR

Use this when the plugin must run on root:

```bash
bindu-studio plugin init --profile ai-safe --id my-plugin --outdir ./my-plugin
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
```

Mental model:

`init -> doctor -> certify -> apply`

Use `--bundle` to publish new code on root, and `--plugin` to assign existing code to more targets.

Reference implementation plan (Beacon gateway):

- `docs/user/beacon-gateway-plugin-implementation-plan.md`

## Bugfix + Redeploy (Exact Behavior)

For a plugin already in use, the minimal and correct flow is:

1) Edit source code (do not run `init` again on the same project).
2) Re-certify bundle.
3) Re-apply bundle to the same targets.

Example (root):

```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```

If the plugin is assigned to nodes, run runtime rollout after apply:

```bash
bindu-studio runtime update wizard
```

### Special Case: Plugin Was Certified Host-Only, Then Node Deploy Fails

Symptom:
- `mesh node-deploy` fails with:
  - `plugin <id>@<version> missing runtime target linux/amd64 (runtime targets: linux/arm64)`

Cause:
- Previous `plugin certify` built only host target (default `GOOS/GOARCH`), so node platform binaries are missing.

Fix sequence (transparent update, same plugin id):

```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH) \
  --json
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin who-uses --plugin my-plugin
bindu-studio mesh node-deploy --nodes <node-id> --out ./dist/nodes
```

Expected behavior after `apply --bundle`:
- Active code for the same plugin `id` is replaced by the new candidate.
- Existing mesh assignments remain in place (root/nodes are preserved).
- Old non-active staged candidates for that plugin are auto-cleaned.
- `remove` / `undeploy` are **not** required for normal code update on the same plugin `id`.

## Assignment-Only to a New Node (Do Not Change Active Code)

Real case: plugin is already active and assigned to `root`; you add a new node and only want that node assignment included in `mesh node-deploy`.

Use `--plugin` (not `--bundle`):

```bash
bindu-studio plugin apply --plugin my-plugin --target node --node <new-node-id>
```

Then verify and build node bundle:

```bash
bindu-studio plugin who-uses --plugin my-plugin
bindu-studio mesh node-deploy --nodes <new-node-id> --out ./dist/nodes
```

Important behavior:
- `plugin apply --plugin <id>` on an already active plugin is assignment-only in practice (no candidate promotion).
- `plugin apply --bundle ...` may import/activate a new candidate for that plugin ID and can affect `root` if `root` uses that plugin.
- For node-only workflows, prefer `plugin register` over `plugin apply`.

## Large Mesh Rollout (Global / Subset / 10 Nodes)

### Global rollout (many nodes)

1) Certify plugin bundle.
2) Publish plugin code once:

```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```

3) Assign broadly (`--target all` or per-node as needed).
4) Deliver runtime to deployed nodes:

```bash
bindu-studio runtime update wizard
```

5) For not-yet-installed nodes, build installer bundles:

```bash
bindu-studio mesh node-deploy --all --out ./dist/nodes
```

### Subset rollout (example: 20 nodes)

```bash
while IFS= read -r node; do
  bindu-studio plugin apply --plugin my-plugin --target node --node "$node"
done < nodes-20.txt

bindu-studio mesh node-deploy --nodes "$(paste -sd, nodes-20.txt)"
bindu-studio runtime update wizard --nodes "$(paste -sd, nodes-20.txt)" --yes
```

### Example: 10 nodes, two plugins

Scenario:
- 5 nodes run `plugin1`
- 5 nodes run `plugin1` + `plugin2`

Flow:
1) `plugin1`: `init -> doctor -> certify -> apply --bundle` once.
2) Assign `plugin1` to first 5 nodes with `apply --plugin`.
3) Later build `plugin2`, apply bundle once, assign only to second group.
4) Use `mesh node-deploy` for first install bundles.
5) Use `runtime update wizard` for already deployed nodes.

## Redeploy the Same Plugin (Complete How-To)

Use this sequence when you fix a bug and want to replace the current plugin using the same `id`.

### Case A: plugin on root only (no nodes)

1) Edit plugin code.
2) Validate:
```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
```
3) Certify:
```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json
```
4) Re-apply on root:
```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```
   - `apply` promotes the new candidate and, when runtime is active, performs plugin stop/start automatically (no `remove/undeploy` required).
5) Verify:
```bash
bindu-studio plugin list
bindu-studio plugin who-uses --plugin my-plugin
```

#### Exact validated sequence (real example)

```bash
./bin/bindu-studio plugin doctor \
  --source /Users/carlosa/.bindu/root/custom-plugins/hello-world-plugin \
  --pre-release --strict

./bin/bindu-studio plugin certify \
  --source /Users/carlosa/.bindu/root/custom-plugins/hello-world-plugin \
  --out /Users/carlosa/.bindu/root/custom-plugins/hello-world-plugin.tar.gz \
  --json

./bin/bindu-studio plugin apply \
  --bundle /Users/carlosa/.bindu/root/custom-plugins/hello-world-plugin.tar.gz \
  --target root \
  --auth console:SuperSecret1

./bin/bindu-studio plugin list --auth console:SuperSecret1
```

How to confirm new code is active:
- `apply` output should include `Activated candidate <new_timestamp>`.
- `plugin list` should show the same timestamp as active candidate.
- If candidate changed (for example `20260221T002256Z` -> `20260221T183917Z`), redeploy is effective.
- If candidate does not change, bundle content was identical (hash deduplication).

Candidate interpretation:
- Format: `YYYYMMDDTHHMMSSZ` (UTC timestamp), for example `20260221T183917Z`.
- `plugin list` marks each staged candidate as `ACTIVE` or `STAGED`.
- In normal `plugin apply --bundle` flow, Studio auto-cleans old non-active staged candidates for the same plugin.

### Case B: plugin assigned to nodes (with or without root)

1) Edit + `doctor` + `certify` from updated source.
2) Re-apply plugin bundle on root:
```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```
3) Ensure node assignment is present (repeat per node as needed):
```bash
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
```
4) Commit runtime desired snapshot:
```bash
bindu-studio runtime desired commit --note "plugin rollout to nodes"
```
5) Roll out runtime update to already deployed nodes:
```bash
bindu-studio runtime update wizard
```
6) For explicit targeting / non-interactive runs:
```bash
bindu-studio runtime update wizard --nodes <node-a,node-b> --yes
bindu-studio runtime update wizard --desired-id <desired-id> --nodes <node-a,node-b> --yes
bindu-studio runtime update wizard --dry-run --json
```
Runtime update wizard flag meanings:
- `--desired-id`: use that runtime desired snapshot ID (otherwise current runtime desired).
- `--yes`: skip `Type APPLY` confirmation.
- `--dry-run`: plan only, no enqueue.
- `--json`: machine-readable output only.

What `APPLY` confirms:
- It confirms enqueueing `NodeRuntimeUpdate` intents to selected eligible non-root nodes.
- It does not mean node apply is already finished; delivery/apply remains asynchronous.

7) Verify final state:
```bash
bindu-studio plugin list
bindu-studio plugin who-uses --plugin my-plugin
```

Runtime update semantics for nodes:
- `runtime update wizard` sends `NodeRuntimeUpdate`.
- Nodes reconcile runtime desired components asynchronously.
- This includes config baseline changes (only when baseline changed) and assigned plugin candidate changes.
- `runtime desired commit` materializes the config artifact automatically, so runtime update can deliver config-only changes without separate manual staging.
- Node binary/version rollout is not part of `runtime update wizard` (that is `NodeUpdate` lifecycle path).
- Plugin selection is automatic from runtime desired snapshot + mesh assignment per node.

Lifecycle boundary:
- `lifecycle activate-root` and `lifecycle activate-node` are product release commands.
- `activate-node` must not be used for plugin rollout.
- `activate-node` must not be used for runtime config rollout.
- If the goal is plugin/config convergence, use `runtime desired commit` plus `runtime update wizard`.

Quick recipes:

1. Plugin-only rollout to an already installed node:
```bash
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
bindu-studio runtime desired commit --note "plugin-only rollout"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

1.a Plugin-only rollout to an already installed node when root must not run the plugin:
```bash
bindu-studio plugin import ./my-plugin.tar.gz
bindu-studio plugin register --plugin my-plugin --target node --node <node-id>
bindu-studio runtime desired commit --note "plugin-only rollout to node"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

2. Config-only rollout:
```bash
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit --note "config-only rollout"
bindu-studio runtime update wizard --nodes <node-id> --yes
```

3. Product release rollout for nodes:
```bash
bindu-studio lifecycle activate-node --yes --auth user:pass
```

### When to use `remove/undeploy`

- NOT for normal bugfix redeploy of the same plugin `id`.
- YES when you want to unassign/remove a plugin from targets, uninstall runtime (`--uninstall`), or purge staged artifacts.

## Command Truths (init / certify / import)

- `plugin init`:
  - Generates scaffold code and files for a new plugin project.
  - It does not deploy and does not update an already deployed plugin by itself.

- `plugin certify`:
  - Validates manifest/contract and message types.
  - Builds Go plugin binaries for selected targets and packages a deterministic certified `.tar.gz`.
  - If no target flags are provided, default target is current host (`GOOS/GOARCH`).
  - Use `--target os/arch` (repeatable, one per flag, syntax with `/`) or `--targets-from-mesh` to include mesh platforms.
  - `--targets-from-mesh` includes only platforms present in mesh, not all Linux permutations.

- `plugin import` with same `id/version`:
  - If bundle content hash is identical to an already staged candidate: returns `alreadyStaged`, no new candidate.
  - If bundle content hash is identical to active deployment: returns `alreadyActive`, no new candidate.
  - If content changed (same `id/version`): creates a new candidate under `runtime/artifacts/plugins/<id>/<version>/<candidate>`.
  - It does not overwrite active runtime by itself; `plugin apply --bundle` handles activation and assignment.
  - After `plugin apply --bundle`, non-active staged candidates for that plugin are auto-cleaned.

## Do I Need remove/undeploy For a Bugfix?

No.

- Normal bugfix path is `certify` + `apply`.
- `remove/undeploy` is only for unassigning/removing a plugin from targets, not for replacing code with a new candidate.

## Wizard Alternative (Human Operators)

Deploy wizard:
```bash
bindu-studio plugin deploy
```

`plugin deploy` now supports two interactive paths:
- `root + assignment` for the classic root deployment flow
- `nodes only` for fleet rollout where root stores the bundle, registers selected nodes, accepts baseline, commits runtime desired, previews rollout, and delivers runtime updates

For fleet rollout, the wizard lets the operator:
- choose `all` eligible nodes, or
- choose specific nodes by comma-separated numbers
- see `gateway` explicitly in the node labels
- review ineligible nodes before delivery
- review a per-node rollout summary before final delivery
- see runtime rollout summaries such as `plugin only`, `config only`, `config + plugin`, or `no changes`

Undeploy wizard:
```bash
bindu-studio plugin undeploy
```

Use wizards for interactive operation. Use `apply/remove` for scripted and repeatable operations.

## Visible Commands

- `plugin init`
- `plugin doctor`
- `plugin certify`
- `plugin apply`
- `plugin remove`
- `plugin deploy`
- `plugin undeploy`
- `plugin list`
- `plugin who-uses`

## Notes About Plugin Version

- Operationally, plugin assignment is by `plugin id`.
- Internal artifact version/candidate details are runtime implementation details and are intentionally not required for normal UX.
- If your team keeps a fixed plugin version (for example `1.0.0`), this model works without changing runtime architecture.

## AI-Safe Tutor Example

Generate the end-to-end tutorial scaffold:

```bash
bindu-studio plugin init \
  --no-interactive \
  --profile ai-safe \
  --template hello-world-dual-cron \
  --id hello-world-plugin \
  --outdir codex-plugins/hello-world-plugin
```

Then certify and deploy:

```bash
bindu-studio plugin doctor --source codex-plugins/hello-world-plugin --pre-release --strict
bindu-studio plugin certify --source codex-plugins/hello-world-plugin --out ./hello-world-plugin.tar.gz --json
bindu-studio plugin apply --bundle ./hello-world-plugin.tar.gz --target root
```

Companion docs:
- `public/console/docs/plugin-init-generated-code-tutor.md`
- `docs/user/plugin-init-generated-code-tutor.md`
