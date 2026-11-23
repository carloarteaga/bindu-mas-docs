# Hello World BPR Plugin Lifecycle Tutorial 🌐🔌

> **Audience**: Solution Creators, Plugin Developers, and Autonomous AI System Agents.  
> **Prerequisites**: A running `bindu-root` instance (or `.e2e-root/current` workspace), with `bindu-studio` available in your `PATH` or executable via `go run ./cmd/bindu-studio`.

---

## Overview & Architecture

This tutorial walks you through the **complete, end-to-end development and operational lifecycle** of a native **Binary Plugin Runtime (BPR)** plugin in Bindu.

Unlike simple scripting plugins, Bindu BPR plugins are external operating system processes running under strict security sandboxing. They communicate with the host engine exclusively via length-prefixed JSON frames over `stdin` and `stdout` using the `bpr/v1` protocol.

By completing this tutorial, you will master the **6 Standard Stages** of plugin development:
1. **Schema-First Engineering**: Declaring schemas and generating Go structs (`gen-go`) and envelope fixtures (`gen-examples`).
2. **Project Scaffolding**: Initializing a production-ready dual-cron project using `plugin init`.
3. **In-Engine Simulation (Dry-Run)**: Dynamically verifying handshake, trigger handling, schema adherence, and clean shutdown before building distribution packages.
4. **Static Audit & Packaging**: Validating project consistency (`plugin doctor`) and cross-compiling bundles (`plugin certify`).
5. **Cluster Deployment & Telemetry**: Deploying to root or edge gateways and consuming telemetry in a live Single Page Application (`hello-world-spa`) via WebSockets (WSS) and REST SQL queries.
6. **Maintenance & Teardown**: Performing rolling version updates, node unassignment (`plugin remove`), cluster undeployment (`plugin undeploy`), and physical cache pruning (`plugin purge`).

---

## Environment Setup & Authentication

Export your studio credentials so CLI commands communicate with the Bindu cluster seamlessly:

```bash
export BINDU_STUDIO_AUTH="console:StrongPass123"
export BINDU_BASE_DIR="./.e2e-root/current"
```

*(Alternatively, you can provide `--auth console:<password>` and `--base-dir <path>` on any command).*

---

## Stage 0: Schema Definition & Code Generation (Level 0)

Every plugin in Bindu is strictly contract-bound. Before writing any plugin logic, define the message contracts in [`hello-world-message-types.json`](./hello-world-message-types.json):

* **`HelloWorldConfig`**: Top-down configuration envelope (`["local", "child"]`) containing `greeting_text`.
* **`HelloWorld`**: Bottom-up telemetry envelope (`["local", "parent"]`) containing the greeting message.
* **`HelloWorldStats`**: Periodic operational statistics (`["local", "parent"]`) containing execution counts and time windows.

### 0.1 Merge Schemas into Cluster Registry
Merge the message definitions into your cluster:

```bash
bindu-studio message-types merge \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --all \
  --base-dir .
```

### 0.2 Generate Strongly-Typed Go Structs (`gen-go`)
Avoid typing errors and runtime payload mismatches by compiling JSON schemas into native Go structs:

```bash
bindu-studio message-types gen-go \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --out ./public/console/docs/hello-world-tutorial/hello-world-plugin/types_gen.go \
  --package main
```

This generates [`types_gen.go`](./hello-world-plugin/types_gen.go) containing:
* `HelloWorldFields`, `HelloWorldConfigFields`, and `HelloWorldStatsFields`.
* Automatic `ToMap()` serializers for emitting payloads via the BPR SDK.
* Safe `Parse...Fields()` deserializers for handling incoming event payloads.

### 0.3 Generate Validated Sample Envelopes (`gen-examples`)
Generate compliant mock payloads for the plugin's `examples/` fixtures directly from the schema constraints:

```bash
bindu-studio message-types gen-examples \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --out ./public/console/docs/hello-world-tutorial/hello-world-plugin/examples/out
```

Copy the generated `HelloWorldConfig.json` to `examples/in/` to serve as the input fixture:

```bash
cp ./public/console/docs/hello-world-tutorial/hello-world-plugin/examples/out/HelloWorldConfig.json \
   ./public/console/docs/hello-world-tutorial/hello-world-plugin/examples/in/
```

---

## Stage 1: Project Scaffolding & Authorship (Level 1)

### 1.1 Scaffolding with `plugin init`
Generate a new BPR Go plugin project using the standard dual-cron template:

```bash
bindu-studio plugin init \
  --profile ai-safe \
  --template hello-world-dual-cron \
  --id hello-world-plugin \
  --outdir ./my-hello-plugin
```

### 1.2 Inspecting `plugin.json`
Review the declarative manifest ([`hello-world-plugin/plugin.json`](./hello-world-plugin/plugin.json)):

```json
{
  "id": "hello-world-plugin",
  "name": "Hello World Plugin",
  "version": "1.0.0",
  "runtime": "bpr",
  "in": [
    "HelloWorldConfig"
  ],
  "out": [
    "HelloWorld",
    "HelloWorldStats"
  ],
  "schedules": [
    {
      "cron": "0 */5 * * * *",
      "trigger": "cron"
    },
    {
      "cron": "0 */15 * * * *",
      "trigger": "cron.2"
    }
  ]
}
```

* **Multi-Cron Triggers**:
  * `cron`: Fires every 5 minutes (`0 */5 * * * *`), causing the plugin to emit a `HelloWorld` greeting.
  * `cron.2`: Fires every 15 minutes (`0 */15 * * * *`), computing cumulative metrics and emitting `HelloWorldStats`.

### 1.3 Implementing Business Logic (`main.go`)
In [`hello-world-plugin/main.go`](./hello-world-plugin/main.go), implement the `sdk.Plugin` interface:

```go
func (p *Plugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
    switch strings.TrimSpace(trigger.ID) {
    case "cron":
        payload := HelloWorldFields{
            Greeting: p.currentGreeting(),
        }
        raw, _ := payload.ToMap()
        return []sdk.Message{
            {Type: MessageTypeHelloWorld, Payload: raw},
        }, nil

    case "cron.2":
        stats, err := buildHelloWorldStats(ctx)
        if err != nil {
            return nil, err
        }
        return []sdk.Message{stats}, nil

    default:
        return nil, nil
    }
}
```

---

## Stage 2: In-Engine Dynamic Simulation (Level 2 - Dry-Run)

Before building release packages or distributing code across the cluster, execute **`plugin dry-run`**.

This command runs the **exact production BPR supervisor engine** (`executioneng.StartBPRSession`) in-process, tests process spawning, verifies token handshakes, delivers real stimulus, checks emitted schema validity, and verifies clean shutdown.

### 2.1 Simulating the Primary Cron Trigger
Simulate the 5-minute `cron` schedule activation:

```bash
bindu-studio plugin dry-run \
  --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --base-dir . \
  --trigger cron \
  --timeout 4s
```

**Terminal Verification Output**:
```text
=================================================================
🔬 BPR Plugin Dry-Run Simulation: "hello-world-plugin" (v1.0.0)
=================================================================
Platform:            linux/amd64
Handshake:           Accepted ✓ (4 ms)
Stimulus Injected:   Type: trigger | ID: cron
Outputs Emitted:     1 message(s)
-----------------------------------------------------------------
[1] Message Type: HelloWorld [✓ PASS]
    Emit Latency: 2 ms | Declared in Manifest: true
    Payload:
{
  "greeting": "Hello World"
}
-----------------------------------------------------------------
Shutdown:            Clean: true | Exit Code: 0 (1 ms)
=================================================================
✅ DRY-RUN RESULT: PASSED (Handshake, Stimulus, Schemas & Shutdown Valid)
=================================================================
```

### 2.2 Simulating Configuration Events
Inject a mock `HelloWorldConfig` event to verify runtime state reconfiguration:

```bash
bindu-studio plugin dry-run \
  --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --base-dir . \
  --event HelloWorldConfig \
  --payload '{"greeting_text": "Greetings from Bindu Mesh!"}' \
  --timeout 4s
```

### 2.3 Automated Agent Integration (`--json`)
For autonomous CI/CD pipelines or AI agents, pass `--json` to receive structured assertions:

```bash
bindu-studio plugin dry-run \
  --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --base-dir . \
  --trigger cron \
  --json
```

---

## Stage 3: Static Audit & Multi-Architecture Certification (Level 3)

### 3.1 Run Static Project Audit (`doctor`)
Audit code structure, SDK import rules, and manifest constraints:

```bash
bindu-studio plugin doctor \
  --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --base-dir . \
  --strict
```

### 3.2 Package Multi-Platform Bundle (`certify`)
Cross-compile and bundle the plugin into an immutable `.tar.gz` archive:

```bash
bindu-studio plugin certify \
  --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
  --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
  --base-dir . \
  --target linux/amd64 \
  --out ./public/console/docs/hello-world-tutorial/hello-world-plugin.tar.gz
```

---

## Stage 4: Cluster Rollout & Application Consumption (Level 4)

### 4.1 Option A: Direct Central Root Deployment
Deploy the plugin directly onto the central root orchestrator:

```bash
bindu-studio plugin apply \
  --bundle ./public/console/docs/hello-world-tutorial/hello-world-plugin.tar.gz \
  --target root \
  --auth console:StrongPass123
```

### 4.2 Option B: Remote Edge Gateway Rollout (Runtime OTA Triptych)
To distribute the plugin to an edge gateway node (e.g. `node-1` or `gateway-minis`):
1. Assign the plugin bundle to the node in `mesh.json`.
2. Execute the 3-step Runtime OTA Triptych:

```bash
# Step 1: Accept the new baseline
bindu-studio runtime baseline accept --yes

# Step 2: Commit the desired target state
bindu-studio runtime desired commit --note "Deploy hello-world-plugin v1.0.0"

# Step 3: Run the interactive/automated rollout wizard
bindu-studio runtime update wizard --yes --auth console:StrongPass123
```

### 4.3 Consuming Live Telemetry in the Standalone SPA
Open [`hello-world-spa/index.html`](./hello-world-spa/index.html) in your browser:
* **Live WebSockets (WSS)**: The SPA connects to the WSS broker port (e.g., `8443`) and subscribes to `HelloWorldStats/defaultGroup/+`. Every time `cron.2` triggers, metrics are rendered instantly in the log table.
* **Read-Only SQL Queries (REST)**: The SPA queries `/api/v1/messages/sql` using queries in [`hello-world-spa/queries/`](./hello-world-spa/queries/) to aggregate historical greeting totals by gateway.
* **Geographic Locagram Maps**: Leaflet maps visualize node polygons dynamically derived from `NodeGeoLocation` metadata.

---

## Stage 5: Advanced Lifecycle: Upgrade, Decommission & Deep Purge (Level 5)

A complete plugin lifecycle includes in-place updates, clean node unassignment, and complete cache purging.

### 5.1 Inspect Active Cluster State
Inspect where the plugin is running:

```bash
# Check all assigned nodes
bindu-studio plugin who-uses --plugin hello-world-plugin

# List installed plugins and versions
bindu-studio plugin list
```

### 5.2 Seamless In-Place Version Upgrade (`1.0.0` $\rightarrow$ `1.1.0`)
1. Increment the version in `plugin.json` to `"version": "1.1.0"`.
2. Re-certify the bundle:
   ```bash
   bindu-studio plugin certify \
     --source ./public/console/docs/hello-world-tutorial/hello-world-plugin \
     --config ./public/console/docs/hello-world-tutorial/hello-world-message-types.json \
     --base-dir . \
     --out ./public/console/docs/hello-world-tutorial/hello-world-plugin-1.1.0.tar.gz
   ```
3. Apply the updated bundle:
   ```bash
   bindu-studio plugin apply \
     --bundle ./public/console/docs/hello-world-tutorial/hello-world-plugin-1.1.0.tar.gz \
     --target root \
     --auth console:StrongPass123
   ```
   The supervisor gracefully terminates the `1.0.0` process and boots `1.1.0` with zero message drops.

### 5.3 Unassign from Nodes (`remove`)
To stop the plugin from running on a specific node without deleting its package archive:

```bash
bindu-studio plugin remove \
  --plugin hello-world-plugin \
  --target root \
  --yes \
  --auth console:StrongPass123
```

### 5.4 Undeploy from Cluster (`undeploy`)
Unregister the plugin bundle from the active cluster inventory:

```bash
bindu-studio plugin undeploy \
  --plugin hello-world-plugin \
  --yes \
  --auth console:StrongPass123
```

### 5.5 Deep Purge of All Artifacts & Caches (`purge`)
Wipe all staged candidate files, unpacked binaries, and cached bundles completely from the cluster disks:

```bash
bindu-studio plugin purge \
  --plugin hello-world-plugin \
  --all \
  --yes \
  --auth console:StrongPass123
```

Confirm that the cluster is completely clean:

```bash
bindu-studio plugin list
```
The output confirms `hello-world-plugin` is no longer active, staged, or cached anywhere on the mesh.

---

## Conclusion & Best Practices

1. **Always run `message-types gen-go`** when modifying JSON schemas to maintain 100% type safety in Go plugin code.
2. **Never deploy without running `plugin dry-run`**. In-process simulation guarantees that process spawning, protocol framing, and schema adherence are 100% sound.
3. **Use the complete teardown sequence** (`remove` $\rightarrow$ `undeploy` $\rightarrow$ `purge`) when decommissioning plugins to maintain pristine cluster health.
