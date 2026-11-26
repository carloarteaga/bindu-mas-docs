# AI Agent Fleet Deployment & Bulk Add Tutorial

This guide is designed for AI Coding Agents and DevOps automation scripts to understand, execute, and program against the headless fleet deployment and bulk node addition subcommands in **Bindu Studio**.

---

## 1. Node Addition in Bulk: `mesh bulk-add`

Before generating deployment packages, the nodes must exist in the mesh topology. Instead of interactive prompts, use `mesh bulk-add` to register nodes programmatically.

### Command Syntax

```bash
bindu-studio mesh bulk-add \
  --pattern "<id-pattern>" \
  --platform "<os>/<arch>" \
  --parent "<parent-node-id>" \
  --connectivity "<gateway|reference>" \
  [--mesh "<path-to-mesh.json>"] \
  [--base "<base-dir>"]
```

### Pattern Expansion Rules for AI Agents

*   **Numeric Ranges**: Use curly braces with double dots, e.g. `{1..100}`.
    *   **Zero-Padding**: If the start or end boundary has leading zeros (e.g., `{01..10}`), the command automatically formats all expanded IDs with the same width (e.g., `01`, `02`, ..., `10`).
*   **Alphabetic Ranges**: Use lowercase range `{a..z}` or uppercase `{A..Z}` (e.g., `{a..e}`).
*   **No Range**: If no range is specified (e.g., `node-pi-5`), it behaves as a single add operation.

### Step-by-Step Registration Example

1.  **Bulk Add Nodes**:
    ```bash
    bindu-studio mesh bulk-add --pattern "node-pi-{01..12}" --platform "linux/arm64" --parent "root" --connectivity "gateway"
    ```
2.  **Commit Draft Changes**:
    The bulk-add subcommand validates the draft schema and writes changes to `mesh.draft.json`. To apply these changes to the active mesh, run:
    ```bash
    bindu-studio mesh apply
    ```

---

## 2. Packaging the Fleet: `mesh node-deploy`

Once the nodes are registered, package the node software and configuration in a generic form.

### Command Syntax

```bash
bindu-studio mesh node-deploy \
  --generic \
  --platform "<os>/<arch>" \
  [--out "<output-directory>"]
```

### Outputs Produced

1.  **`bindu-node-generic-<os>-<arch>.tar.gz`**:
    *   Contains launcher, installer, and root configuration files.
    *   **Crucial for Automation**: It does **not** contain any node identity files (the `identity/` directory is omitted).
2.  **`fleet-deploy.json`**:
    A manifest file detailing the bootstrap configurations, CA certs, and enrollment token:

```json
{
  "bootstrap": {
    "parent_id": "root",
    "mqtt_candidates": ["192.168.1.10:1888"],
    "signing_url": "https://192.168.1.10:8448/api/v1/tls/sign-node",
    "root_ca_pem": "-----BEGIN CERTIFICATE-----\n..."
  },
  "enrollment": {
    "token": "active-enrolment-token-xyz"
  },
  "nodes": [
    {"node_id": "node-pi-01", "target_ip": ""},
    {"node_id": "node-pi-02", "target_ip": ""}
  ]
}
```

---

## 3. Native Fleet Deployment Orchestration: `bindu-studio fleet deploy`

Instead of writing fragile custom Python or Bash scripts with external `scp` and `ssh` subprocess loops, Bindu provides a native, highly concurrent Go orchestrator built into the CLI: **`bindu-studio fleet deploy`**.

### Architecture & Key Capabilities

1. **Zero-Disk Streaming (CGO-Free Go SSH)**:
   Connects using `golang.org/x/crypto/ssh` and pipes the compressed generic payload directly to the remote machine's `tar -xzf -` over the SSH session's stdin. No temporary `.tar.gz` files on remote disks, no multiple SCP round-trips.
2. **Semantic Template Cloning**:
   Clones an existing, fully configured node (`--template <node_id>`) with its assigned MAS plugins (`bindu-cv`, `hello-world`), MQTT channels, ports, and CEL reactors across all target units.
3. **Dynamic Hardware Discovery & Naming**:
   Supports patterns such as:
   - `bn-{ip}`: e.g. `bn-10-0-1-42`
   - `drone-{mac}`: inspects remote network interface (`/sys/class/net/*/address`) and formats hardware MAC.
   - `drone-{index:03d}`: zero-padded numeric indices (`drone-001`, `drone-002`).
   - `node-{machine-id}`: unique Linux system machine-id (`/etc/machine-id`).
4. **Concurrent Worker Pool**:
   Deploys to hundreds of machines in parallel with configurable goroutine pools (e.g. `--concurrency 30`), achieving $<3$ minutes rollout across 400 drones.

### Command Examples

```bash
# 1. Deploy 50 drones via IP range with automatic naming by IP
bindu-studio fleet deploy \
  --template "drone-template" \
  --ips "10.0.1.1-10.0.1.50" \
  --ssh-user "pi" \
  --ssh-password "raspberry" \
  --naming "bn-{ip}" \
  --concurrency 25

# 2. Deploy using a JSON inventory with dynamic MAC address discovery
bindu-studio fleet deploy \
  --template "drone-template" \
  --inventory ./drones-fleet.json \
  --naming "drone-{mac}" \
  --concurrency 30

# 3. Simulate and preview node ID resolution without modifying the mesh or executing installations
bindu-studio fleet deploy \
  --template "drone-template" \
  --ips "192.168.10.0/28" \
  --naming "drone-{index:03d}" \
  --dry-run --json
```

---

## 4. Legacy / External Script Orchestration (Python / Shell Reference)

If integrating with external CI/CD or legacy orchestrators, you can still consume `fleet-deploy.json` directly:

### Pseudocode for Custom External Scripts

```python
import json
import subprocess

# 1. Load the fleet manifest
with open("dist/nodes/fleet-deploy.json") as f:
    manifest = json.load(f)

token = manifest["enrollment"]["token"]
generic_bundle = "dist/nodes/bindu-node-generic-linux-arm64.tar.gz"

# 2. Iterate and deploy
for node in manifest["nodes"]:
    node_id = node["node_id"]
    
    # Resolve target IP (if blank in manifest, lookup from agent's inventory)
    target_ip = node["target_ip"] or my_inventory.lookup_ip(node_id)
    
    # Copy the generic bundle to the target machine
    subprocess.run(["scp", generic_bundle, f"bindu@{target_ip}:/tmp/bindu-node.tar.gz"], check=True)
    
    # Extract and execute the installer using enrollment token overrides
    install_command = (
        f"tar -xzf /tmp/bindu-node.tar.gz -C /opt/bindu && "
        f"/opt/bindu/bindu-install-node "
        f"--node-id {node_id} "
        f"--enrollment-token {token} "
        f"--yes"
    )
    subprocess.run(["ssh", f"bindu@{target_ip}", install_command], check=True)
    
    print(f"Node {node_id} successfully enrolled and started!")
```

### Command Run on Node Target:

#### Option A: Authenticating with Enrollment Token (Recommended for head-less / mass deploy)
```bash
./bindu-install-node --node-id <node_id> --enrollment-token <token> --yes
```
*   **`--node-id`**: Forces the node identity.
*   **`--enrollment-token`**: Authenticates the preflight request with the Root using the header `X-Enrollment-Token`, bypassing interactive Basic Auth prompts.
*   **`--yes`**: Auto-confirms all installer options (non-interactive mode).

#### Option B: Authenticating with Root Admin Credentials (Alternative/Dev)
If no active enrollment tokens are configured in the Root Coordinator's `binduConfig.json` (located at `runtime/config/binduConfig.json`), the deployment CLI will fallback to using the default token `"one-time-token-valid-for-24h-abc"`. 
To enroll using this fallback token, the operator must manually add the token to the Root Coordinator's `binduConfig.json` inside the `"enrollment_tokens"` array and restart the coordinator service (`sudo systemctl restart bindu-root`).

Alternatively, to avoid modifying config files, the operator can pass the Root Coordinator's console credentials directly:
```bash
./bindu-install-node --node-id <node_id> --root-user console --root-pass <your_console_password> --yes
```
*   **`--root-user`**: Root Coordinator console administrator username.
*   **`--root-pass`**: Root Coordinator console administrator password.
*   **`--yes`**: Auto-confirms all installer options (non-interactive mode).

Once completed, `bindu-node` is launched under systemd. It will automatically announce its presence and ports back to the Root Coordinator.
