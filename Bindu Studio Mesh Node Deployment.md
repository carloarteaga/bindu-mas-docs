# Bindu Studio Mesh Node Deployment

Role of this document:

- Supporting operator workflow guide for node bundle assembly
- Best used after the canonical mesh manual when preparing per-node deployment artifacts

For mesh editing and apply workflows, start with
`public/console/docs/Bindu Studio Mesh User Manual.md`.

Bindu Studio `mesh node-deploy` builds per-node bundle directories from an existing root deployment. It is a reuse-only orchestration command: it reads the current root deployment and assembles node bundles deterministically without changing runtime behavior or mesh semantics.

## Activation Flow (Operator UX)

After importing update bundles, operators use explicit split activation:

1. `bindu-studio lifecycle activate-root`
2. `bindu-studio lifecycle activate-node`

Installed-root recipe (local CLI path + auth):

```bash
./bin/bindu-studio lifecycle import-update ~/binduUpdate-<version>.tar.gz --auth console:<password>
./bin/bindu-studio lifecycle activate-root --auth console:<password>
./bin/bindu-studio lifecycle activate-node --to active --auth console:<password>
```

Order requirement:
- activate root first, then activate nodes.
- if root runs as a service, root restart after `activate-root` is automatic.
- if root runs manually, relaunch root before `activate-node`.
- if you apply a plugin to root and root is running manually, relaunch root manually after the root-targeted plugin apply.

`activate-root` wizard shows imported versions and confirms the selected one.

`activate-node` defaults to root active version when `--to` is omitted, shows a summary, and enqueues intents asynchronously.

Operator boundary:

- `activate-node` is product release rollout only
- `activate-node` sends `NodeUpdate`
- `activate-node` does not send runtime config
- `activate-node` does not send plugins

## Production Note: Root Should Run As A Service

In production, root should run under a service manager.

Reasons:
- `activate-root` depends on reliable service restart for clean product activation.
- root-targeted plugin deployment is safer when the process is supervised and restartable.
- root is the mesh control plane, console host, artifact server, and runtime update API; those responsibilities should recover automatically after crash or reboot.
- service management gives one operator path for status, start, stop, and restart instead of ad hoc manual relaunch.

Canonical update send semantics:

- intent enqueue, not delivery confirmation
- no presence gating
- no skip

`bindu-studio lifecycle activate` is deprecated compatibility only and is not the recommended UI path.

## Update Bundle Wizard UX (Current)

`scripts/build-update-bundle.sh --wizard --from-source` now always builds CORE update payload:

- root runtime (`bindu-root`)
- console files (`public/console`)
- node bundles for selected/all targets

Studio tooling (`tools/studio/*`) is optional extra payload in wizard mode.
It is no longer presented as a standalone update path.

## Runtime Component Update Flow (Config + Plugins)

When the goal is to update node runtime components (config and assigned plugins), use the runtime wizard:

1. Ensure runtime desired state is up to date (plugins/config already managed from root).
2. Commit runtime desired snapshot:

```
bindu-studio runtime desired commit --note "config/plugins rollout"
```
3. Send runtime update intents:

```
bindu-studio runtime update wizard
```

Optional:

```
bindu-studio runtime update wizard --desired-id <desired-id> --yes
bindu-studio runtime update wizard --dry-run --json
```

Wizard behavior and flags (important):

- Interactive mode asks: `Type APPLY to confirm runtime update delivery`.
- `--yes` skips that confirmation prompt.
- `--desired-id` forces a specific runtime desired snapshot; without it, current runtime desired is used.
- `--dry-run` computes plan only (no enqueue).
- `--json` prints machine-readable output.
- Targets come from `--nodes` or interactive selection; root is always excluded.
- Without TTY, run with `--yes` (unless `--dry-run`) to avoid confirmation error.

Runtime update uses `NodeRuntimeUpdate` and the same async delivery model:

- enqueue intent first
- node applies when artifact is received
- delivery and application are asynchronous
- scope is runtime components only (config/plugins), not node binary rollout
- root is not a runtime-update target
- after a successful runtime apply, the node restarts to converge config/plugin state cleanly

Use binary OTA (`NodeUpdate`) only for node runtime binary/version updates.

## Operational Recipes

### 1. Import a product update and activate root

```bash
./bin/bindu-studio lifecycle import-update ~/binduUpdate-<version>.tar.gz --auth console:<password>
./bin/bindu-studio lifecycle activate-root --to <version> --yes --auth console:<password>
```

### 2. Activate all nodes to the current active root release

```bash
./bin/bindu-studio lifecycle activate-node --yes --auth console:<password>
```

Equivalent explicit form:

```bash
./bin/bindu-studio lifecycle activate-node --to active --yes --auth console:<password>
```

### 3. Activate selected nodes to a specific release

```bash
./bin/bindu-studio lifecycle activate-node --to <version> --nodes dell-debian1,bindu-pi3-uno --yes --auth console:<password>
```

### 4. Roll out runtime config only

```bash
./bin/bindu-studio runtime baseline accept --yes
./bin/bindu-studio runtime desired commit --note "config-only rollout"
./bin/bindu-studio runtime update wizard --nodes dell-debian1,bindu-pi3-uno --yes --auth console:<password>
```

### 5. Roll out plugins only

```bash
./bin/bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root --auth console:<password>
./bin/bindu-studio plugin apply --plugin my-plugin --target node --node bindu-pi3-uno --auth console:<password>
./bin/bindu-studio runtime desired commit --note "plugin-only rollout"
./bin/bindu-studio runtime update wizard --nodes bindu-pi3-uno --yes --auth console:<password>
```

### 6. Roll out config and plugins together

```bash
./bin/bindu-studio runtime baseline accept --yes
./bin/bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root --auth console:<password>
./bin/bindu-studio plugin apply --plugin my-plugin --target node --node bindu-pi3-uno --auth console:<password>
./bin/bindu-studio runtime desired commit --note "config+plugin rollout"
./bin/bindu-studio runtime update wizard --nodes bindu-pi3-uno --yes --auth console:<password>
```

Runtime wizard planning model:

1) Config:
- included only when runtime desired config baseline differs from the node baseline.
- materialized automatically by `runtime desired commit` from the full root `runtime/config` directory.
- this allows config-only rollout without separate manual staging.

2) Plugins:
- selected automatically from runtime desired snapshot + mesh assignment for each target node.
- no manual plugin picker in this command.

3) Nodes:
- selected by `--nodes` or interactive/all eligible mode.
- root is always excluded.

The same wizard supports:
- config-only rollout
- plugin-only rollout
- config + plugin rollout

Rule of thumb:
- `lifecycle activate-node` = release OTA only
- `runtime update wizard` = runtime OTA only

## Studio Self-Update (Studio-only)

For `bindu-studio` updates without root/node activation:

1. Build studio-only bundle:

```
./scripts/build-studio-update-bundle.sh --version <version>
```

2. Apply self-update on root host:

```
bindu-studio self-update import ./dist/updates/binduStudioUpdate-<version>.tar.gz
```

Semantics:

- validates current platform payload checksum
- atomically replaces `<BaseDir>/bin/bindu-studio`
- does not require `activate-root` or `activate-node`

## Deployment Contract (Runtime Layout)

Operator installs must follow the Single-Pointer launcher model:

- `<BaseDir>/bindu-root` or `<BaseDir>/bindu-node` is a stable launcher binary.
- `<BaseDir>/runtime/active.json` is the only active-version pointer.
- `<BaseDir>/releases/<version>/...` contains immutable runtime payload for that version.
- Launchers execute the active release binary from `releases/<active_version>/...`.
- Node releases must include version-coupled schemas under `public/console/schema/`.

Version output clarification (important):

- `./bindu-root --version` (or `./bindu-node --version`) reports the launcher binary identity.
- The launcher can keep the install-time build label while the active runtime version in `runtime/active.json` changes through lifecycle activation.
- To inspect the active runtime binary version, read `runtime/active.json` and run `./releases/<active_version>/bindu-root --version` (or node equivalent).

## Command Summary

```
bindu-studio mesh node-deploy [flags]
bindu-studio runtime update wizard [flags]
```

### Flags

- `--root` (string, default `~/.bindu/root`): Root deployment directory to read.
- `--nodes` (string): Comma-separated node_ids to include.
- `--all` (bool): Include all nodes (excluding root).
- `--out` (string, default `./dist/nodes`): Output directory.
- `--dry-run` (bool): Print plan only.

## Inputs

`mesh node-deploy` reads an existing root deployment. It expects:

- `config/mesh.json` for node inventory and assignments.
- `config/` for operational config (copied verbatim).
- `tools/node/<os-arch>/` for platform-specific node tools (Linux only for nodes).
- Root CA certificate under `config/tls/ca.crt`.

## Selection Rules

Nodes can be selected in three ways:

1. `--all`  
   Includes every node from the mesh, excluding root.
2. `--nodes gw-1,node-2`  
   Comma-separated list of node_ids.
3. Interactive selection  
   If no selection flags are given and a TTY is available, Studio prompts for a multi-select list.

Node platform (OS/arch) is resolved from the mesh entry for each node.

## Output Structure

Default output:

```
./dist/nodes/
  bindu-node-<node_id>-<timestamp>.tar.gz
```

Notes:

- Each archive expands to a **runtime-only** bundle (no toolchains).
- Deterministic content based on the root deployment and mesh.
- No runtime changes are performed; this is packaging only.

Bundle layout (after extraction):

```
bindu-node
bindu-install-node
bootstrap.json
config/
identity/
plugins.d/
public/console/schema/
data/
```

## TLS Handling & Node Enrollment Authentication

Each node bundle includes **only the CA certificate**:

- `ca.crt`

Node private keys are generated **locally** during installation. The node installer (`bindu-install-node`) requests a signed cert from the Root Coordinator.

### Installation Path & Systemd Sandbox Restrictions

> [!IMPORTANT]
> **Avoid `/home` paths for systemd services**: In modern Linux distributions (such as Ubuntu or Debian), security systems like AppArmor, SELinux, and systemd's sandbox settings restrict system-level services from executing binaries located in user home directories (`/home/`). Running a system service from a home directory will fail with a `status=203/EXEC` error.
> 
> However, this restriction only applies to the background daemon's executing code. The installer and the raw bundle files are temporary and can be extracted and executed safely inside a user directory (such as `/home/bindu/bindu-nodes/instalador/`). 
> 
> Only the final execution directory passed via the `--prefix` flag needs to be placed under a system path (e.g. `/opt/bindu-nodes/<node-id>`). Ensure the service user `bindu` owns the prefix directory prior to running the installer:
> ```bash
> sudo mkdir -p /opt/bindu-nodes/node-dell-1
> sudo chown bindu:bindu /opt/bindu-nodes/node-dell-1
> ```

### Authentication Options for Node Installation

When executing the installer command, the node must authenticate against the Root Coordinator to authorize the certificate signature. The operator can choose one of the following two routes:

#### Option A: Root Coordinator Credentials (Recommended for local test & Dev)
Uses the main admin credentials of the Root Coordinator. Highly robust as it doesn't require configuring tokens or restarting services.
```bash
sudo /home/bindu/bindu-nodes/instalador/bindu-install-node \
  --bundle /home/bindu/bindu-nodes/instalador \
  --prefix /opt/bindu-nodes/node-dell-1 \
  --node-id node-dell-1 \
  --root-user console \
  --root-pass supersecret \
  --service-name "bindu-node-dell-1" \
  --service-user "bindu" \
  --service-group "bindu" \
  --install-service \
  --yes
```
*   `--root-user`: Root Coordinator console administrator username (default `console`).
*   `--root-pass`: Root Coordinator console administrator password (default `supersecret`).
*   `--service-user` / `--service-group`: The system user and group executing the process (default `bindu`).
*   `--yes`: Bypasses confirmation prompts (auto-approves installer actions / non-interactive mode).

#### Option B: Enrollment Token (Recommended for mass production provisioning)
Uses a temporary, restricted Pre-Shared Key (token) specified in `fleet-deploy.json`.
```bash
sudo /home/bindu/bindu-nodes/instalador/bindu-install-node \
  --bundle /home/bindu/bindu-nodes/instalador \
  --prefix /opt/bindu-nodes/node-dell-1 \
  --node-id node-dell-1 \
  --enrollment-token "one-time-token-valid-for-24h-abc" \
  --service-name "bindu-node-dell-1" \
  --service-user "bindu" \
  --service-group "bindu" \
  --install-service \
  --yes
```
*   `--enrollment-token`: Pre-shared token key to authenticate node enrollment.
*   `--yes`: Bypasses confirmation prompts.

> [!IMPORTANT]
> If no enrollment tokens are configured in the Root Coordinator's `binduConfig.json` (located at `runtime/config/binduConfig.json`), the deployment wizard automatically writes a fallback token `"one-time-token-valid-for-24h-abc"` into the manifest.
> To use this fallback token, the operator must manually add it to the Root's `binduConfig.json` under the `"enrollment_tokens"` string array and restart the coordinator service (`sudo systemctl restart bindu-root`) before running the installer, otherwise enrollment will fail with `Unauthorized`.

No OS trust store changes are required on nodes for this step.

## Bootstrap & First Contact (Summary)

Each bundle includes `bootstrap.json`, which provides **first‑contact hints** for the node installer.
These hints are used only during the initial connection to root and are not written to `mesh.json`.

`mesh node-deploy` also injects a **signing URL** into `bootstrap.json`:

```
signing_url = "https://<reachable-root-host>:<rest_query.port>/api/v1/tls/sign-node"
```

`mesh node-deploy` selects a reachable root host from the same bootstrap candidates (prefer LAN IP, then `.local`, then node_id).
This value is required by the installer and removes any default port assumptions.

## Validation (Fail‑Fast)

Before creating any bundle, `mesh node-deploy` validates:

- Mesh topology (single parent, no cycles, unique IDs).
- Forbidden networking fields in `mesh.json` (no hosts/IPs/endpoints).
- Root deployability (root must include `rest_query` and `ws_integration`).
- Channel port collisions.

If any validation fails, no bundles are produced.

## Examples

Deploy bundles for all nodes:

```
bindu-studio mesh node-deploy --all
```

Deploy specific nodes into a custom output directory:

```
bindu-studio mesh node-deploy --nodes gw-1,node-2 --out ./dist/nodes
```

## Notes & Limitations

- Studio-only orchestration: no runtime activation or plugin lifecycle changes.
- Bundles are unpack-only; no automatic install is performed.
- If your mesh or root deployment is incomplete, `mesh node-deploy` will fail fast with a clear error.

## Generic Fleet Deployment Mode

For automated deployments or AI-agent-led rollouts, `bindu-studio mesh node-deploy` supports building **generic**, identity-free node bundles. This eliminates the need to generate separate, timestamped/customized archives for each individual node.

### Flags

*   `--generic` (bool): Activates generic, identity-free packaging.
*   `--platform` (string): Mandatory with `--generic`. Specifies the target OS/architecture (e.g. `linux/arm64` or `linux/amd64`).

### Behavior and Manifest Generation

1.  **Identity Omission**: When `--generic` is active, the generator does not package the `identity/` directory (holding client TLS certificates) inside the tarball. Nodes will dynamically generate their own private keys and request certificates from the Root during the installer run.
2.  **Generic Tarball Output**: A single generic bundle is compiled at `dist/nodes/bindu-node-generic-<os>-<arch>.tar.gz`.
3.  **Fleet Manifest (`fleet-deploy.json`)**: Alongside the tarball, a manifest file is generated in the output directory detailing the bootstrap configuration and enrollment credentials:

```json
{
  "bootstrap": {
    "parent_id": "root",
    "mqtt_candidates": ["192.168.1.10:1888"],
    "signing_url": "https://192.168.1.10:8448/api/v1/tls/sign-node",
    "root_ca_pem": "-----BEGIN CERTIFICATE-----\n..."
  },
  "enrollment": {
    "token": "one-time-token-valid-for-24h-abc"
  },
  "nodes": [
    {"node_id": "node-pi-1", "target_ip": ""},
    {"node_id": "node-pi-2", "target_ip": ""}
  ]
}
```

*   **`enrollment.token`**: Extracted from the active `enrollment_tokens` list in `binduConfig.json`.
*   **`target_ip`**: Maps each selected gateway node compatible with the target platform. If the node's `bind` address in `mesh.json` is a wildcard (`0.0.0.0`) or loopback, it is set to `""` so the deployment script or AI agent can map the target machine dynamically.

---

## Bulk Node Creation: `mesh bulk-add`

To accelerate mesh expansion, `bindu-studio mesh bulk-add` allows registering hundreds of nodes in a single non-interactive command using pattern range expansions.

### Usage

```bash
bindu-studio mesh bulk-add \
  --pattern "node-pi-{1..100}" \
  --platform "linux/arm64" \
  --parent "root" \
  --connectivity "gateway"
```

### Flags

*   `--pattern` (string, required): Brace range expansion pattern.
    *   **Numeric Ranges**: e.g., `node-pi-{1..100}`. If a leading zero is provided (e.g. `{01..05}`), the generated IDs are padded with zeros automatically (producing `01`, `02`, `03`...).
    *   **Alphabetic Ranges**: e.g., `node-{a..d}` or `node-{A..C}`.
*   `--parent` (string, required): Node ID of the parent node in the mesh hierarchy.
*   `--platform` (string): Target platform OS/Arch (required for `gateway` connectivity, e.g. `linux/arm64`).
*   `--connectivity` (string, default `gateway`): Connectivity kind, either `gateway` or `reference`.

### Mechanics
*   All generated nodes share the configured defaults (platform, parent connectivity).
*   For `gateway` nodes, channels `mesh_mqtt` (port 1883) and `mqtt_integration` (port 1885) are automatically created with default settings.
*   Nodes are checked for ID duplication and the document is validated against the active mesh JSON schema (`meshSchema.json`).
*   If validation succeeds, nodes are appended to `mesh.draft.json`. Run `bindu-studio mesh apply` to commit the changes.
