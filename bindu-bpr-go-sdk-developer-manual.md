# Bindu BPR Go SDK Developer Manual

This manual is aligned with BREP (Bindu Runtime Extension Platform), the canonical plugin-extension model in Bindu.

## Table of Contents
1. [Introduction](#introduction)
2. [Quick Start (5 minutes)](#quick-start-5-minutes)
3. [Concepts Overview](#concepts-overview)
4. [What is BPR](#what-is-bpr)
5. [Plugin vs bindu-root responsibilities](#plugin-vs-bindu-root-responsibilities)
6. [Machine Requirements](#machine-requirements)
7. [Creating Your First Plugin (Step by Step)](#creating-your-first-plugin-step-by-step)
8. [BPR Runtime Interface (bindu-root ↔ plugin)](#bpr-runtime-interface-bindu-root--plugin)
9. [Go SDK Reference](#go-sdk-reference)
10. [Context](#context)
11. [Inbound Interfaces](#inbound-interfaces)
12. [Outbound Interfaces](#outbound-interfaces)
13. [Scheduling Interfaces](#scheduling-interfaces)
14. [SQL Read-Only Interface](#sql-read-only-interface)
15. [Studio plugin init Artifacts Explained](#studio-plugin-init-artifacts-explained)
16. [Development & Debugging](#development--debugging)
17. [Standalone mode](#standalone-mode)
18. [Managed by bindu-root](#managed-by-bindu-root)
19. [VS Code debugging](#vs-code-debugging)
20. [Certification & Artifacts](#certification--artifacts)
21. [Training an AI Agent to Build BPR Go Plugins](#training-an-ai-agent-to-build-bpr-go-plugins)
22. [Notes, Gaps, and Observations](#notes-gaps-and-observations)
23. [Appendix (if needed)](#appendix-if-needed)

## Introduction
This is the authoritative developer manual for building BPR plugins in Go using the Bindu SDK and Bindu Studio.
It is grounded in the current repository implementation, the Go SDK in `sdk/`, the BPR protocol in `bpr/`,
and the canonical example `plugins-bpr-demo-go/hello-world`.

## Quick Start (5 minutes)
Goal: create, run, and observe a working Bindu BPR Go plugin in under 5 minutes.

In this section you will create, run, and observe a working Bindu BPR Go plugin in under 5 minutes.

Steps:
1) Verify Go installation:
   - `go version` (repo `go.mod` uses Go 1.24)
2) Generate a plugin using Studio:
   - `bindu-studio plugin init --profile ai-safe` (canonical path)
3) Build the plugin:
   - `go build -o hello-world`
4) Run project doctor checks:
   - `bindu-studio plugin doctor --source ./hello-world --pre-release --strict`
5) Run it in standalone mode (handshake check):
   - `go run ./cmd/bpr-smoke-host --bin ./hello-world`
6) Observe output/logs:
   - Terminal prints: `bpr smoke handshake accepted`
7) Optional: run via bindu-root using "Run Now" from Console or via the plugin lifecycle steps below.

Example (conceptual):
```bash
bindu-studio plugin init --profile ai-safe
cd hello-world
go build
./hello-world
```

What just happened:
- Studio generated a Go plugin scaffold and a `plugin.json`.
- The SDK performed a BPR handshake over stdin/stdout.
- The smoke host validated the handshake and kept the process alive.

Where the plugin communicated:
- BPR uses length-prefixed JSON frames over stdin/stdout.

What bindu-root does differently:
- It spawns the plugin, injects session env vars, validates the handshake, and sends triggers/events.

## Concepts Overview
- BPR is the only supported runtime contract for plugins.
- Plugins are external processes; they never run inside bindu-root.
- Plugin behavior is declared in manifests and validated by Studio before import.
- Plugins consume and emit messages; routing and storage are owned by bindu-root.

## What is BPR
BPR (Bindu Plugin Runtime) is a semantic runtime contract for plugins:
- Transport: framed JSON over stdin/stdout.
- Handshake: plugin declares inputs, outputs, capabilities, and contract version.
- Execution: events and control triggers are delivered by bindu-root.
- Security: session token binds the process to its spawn.
- Sandboxing: strict environment variable whitelisting (`PATH`, `TMPDIR`, `LANG`, `BINDU_PLUGIN_*`) prevents host secret leaks (`BINDU_MASTER_KEY`, cloud credentials).
- Process Containment: runs in an isolated process group (`Setpgid`) with kernel-level lifecycle termination (`Pdeathsig: SIGKILL`) if the host terminates.

## Plugin vs bindu-root responsibilities
Plugin responsibilities:
- Implement the BPR handshake and handlers.
- Validate and transform input payloads.
- Emit new messages declared in `plugin.json` and handshake.
- Log errors to stderr and fail fast on misconfiguration.

bindu-root responsibilities:
- Spawn and supervise plugin processes under strict environment isolation.
- Provide session identity via environment variables.
- Validate and route emitted messages.
- Enforce capabilities, schedules, and lifecycle policies.
- Automatically terminate orphaned plugin child processes on shutdown.

## Machine Requirements
This section applies to the machine running bindu-root during Go plugin development.

Go toolchain:
- Install Go 1.24 (matches `go.mod`).
- Verify: `go version`.

Bindu Studio / bindu-root:
- `bindu-studio` must be available for `bindu-studio plugin init`, `plugin doctor`, `plugin certify`, and lifecycle commands.
- `bindu-root` runs locally during development (use `./dev.sh` for dev mode).
- `bindu-studio plugin init` pulls Message Type samples from the running system, so bindu-root must be reachable.

Canonical Message Type JSON location (SDK docs contract):
- Repository source (development):
  - `export-message-types/export-message-types.json`
- Installed root bundle (operator/runtime):
  - `public/console/docs/hello-world-tutorial/hello-world-message-types.json`

Import examples:
```bash
# repo/development path
bindu-studio message-types import --in-path ./export-message-types/export-message-types.json

# installed root base-dir path (run from <BaseDir>)
bindu-studio message-types import --in-path ./releases/<active-version>/public/console/docs/hello-world-tutorial/hello-world-message-types.json
```

This keeps tutorial message contracts available both in source and in shipped runtime docs.

Filesystem expectations:
- Plugin source lives anywhere (developer workspace).
- Certified bundles are `.tar.gz` artifacts (your chosen output path).
- Staged candidates live under `artifacts/plugins/<id>/<version>/<candidate>/`.
- Active plugins live under `plugins.d/bpr/<id>/<version>/`.

Environment variables (provided by bindu-root at runtime):
- `BINDU_PLUGIN_TOKEN` (required; session token)
- `BINDU_PLUGIN_NAME`, `BINDU_PLUGIN_VERSION` (required identity)
- `BINDU_NODE_ID`, `BINDU_NODE_ROLE` (optional context)
- `BINDU_SESSION_ID` (optional context)

Certification requirements:
- Bindu Studio must generate deterministic bundles with manifest + checksums.
- Bundles must include `bindu.env` with a valid `BINDU_PLUGIN_TOKEN`.

## Creating Your First Plugin (Step by Step)
This walkthrough uses the canonical example `plugins-bpr-demo-go/hello-world` as reference.

If Go is not installed yet:
```bash
go version
```
Install Go 1.24 and re-run `go version` until it reports 1.24.x.

### 1) Initialize a new plugin via Studio
Run:
```bash
bindu-studio plugin init
```
Expected output (example structure, from `plugins-bpr-demo-go/hello-world`):
```
hello-world/
  go.mod
  main.go
  plugin.json
  README.md
  examples/
    out/
      HelloWorldState.json
```

Key observations from the generated example:
- `plugin.json` declares `runtime: "bpr"`, outputs, and schedules.
- `main.go` implements the BPR SDK runtime and emits a sample payload.
- The template includes message samples under `examples/out/`.

The generated plugin imports `bindunet/sdk`.
Its `go.mod` points at a local Bindu module root via `require bindunet` + `replace bindunet => ...`.
Do not copy `sdk/` or `bpr/` into the plugin source tree.


### 2) Inspect and edit `plugin.json`
From `plugins-bpr-demo-go/hello-world/plugin.json`:
```json
{
  "id": "hello-world-id",
  "name": "hello world example",
  "version": "0.1.0",
  "runtime": "bpr",
  "out": [
    "HelloWorldState"
  ],
  "schedules": [
    { "cron": "0 */5 * * * *" }
  ]
}
```
Guidelines:
- Keep `id` and `version` stable per release.
- Declare outputs and schedules explicitly.
- Keep this in sync with your Go `Handshake()` values.

### 3) Implement plugin logic in `main.go`
The hello-world template shows:
- `Handshake()` declares outputs and capabilities.
- `OnTrigger()` emits messages using `sdk.Message`.
- `OnEvent()` is a no-op until you add inputs.

### 4) Build the plugin
```bash
go build -o hello-world
```

### 5) Run in standalone development mode (hosted handshake)
Use the built-in smoke host:
```bash
go run ./cmd/bpr-smoke-host --bin ./hello-world
```
Expected output:
```
bpr smoke handshake accepted
```
This confirms the plugin can handshake and stay alive under a host.

### 6) Certify, import, activate, and run under bindu-root
```bash
bindu-studio plugin doctor --source ./hello-world --pre-release --strict
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz
bindu-studio plugin apply --bundle ./hello-world.tar.gz --target root
```
Then use Console "Run Now" or wait for cron triggers.

Target flags (important):
- `--target` expects `os/arch` (for example `linux/amd64`), and is repeatable.
- `--targets-from-mesh` includes only platform targets present in current mesh.
- `--targets-from-mesh` is not “all Linux targets”.
- For Raspberry Pi 32-bit OS, use canonical `linux/armv7` (legacy `linux/arm` is accepted as compatibility alias).

Example for root on mac + Linux nodes:

```bash
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH)
```

Certification contract notes:
- `plugin certify` hard-fails when `plugin.json` and `Handshake()` diverge (`in/out`).
- **Semantic Closure & Schema Resolution**: All message types emitted (`out`) or accepted (`in`) in `plugin.json` must exist in the message types registry. When developing a plugin with newly introduced domain schemas not yet merged into the runtime (`runtime/config/messageTypesConfiguration.json`), supply `--config path/to/poc_message_types.json` so the tool validates semantic closure against the local schema file.
- If schedules are declared, `Handshake().Capabilities` must include `timer:cron`.
- Use `bindu-studio plugin certify --json` for machine-readable issues.

## BPR Runtime Interface (bindu-root ↔ plugin)
Transport:
- stdin (host -> plugin), stdout (plugin -> host), stderr (logs only).
- Framed JSON with 4-byte big-endian length prefix.
- Max frame size: 16 MiB (`bpr.MaxFrameSize`).

Logging safety rule:
- Use `ctx.Logger().Printf(...)` or `log.Printf(...)`.
- Never write runtime logs with `fmt.Printf`/`fmt.Println` (stdout is protocol channel).

Handshake:
- Plugin sends `handshake` envelope with token, plugin identity, and capabilities.
- bindu-root replies with `handshake` accepted/rejected.
- The SDK enforces token presence and exits on rejection.
- `schema_set_hash` is optional; include it if you pin a schema set for validation.

Session token:
- Provided via `BINDU_PLUGIN_TOKEN`.
- Must be included in the handshake.
- Must never be logged.

Envelope types (BPR v1):
- `handshake`
- `event` (messages)
- `control` (shutdown, trigger)
- `emit` (plugin -> host)
- `error` (host -> plugin)

Context and identity:
- `BINDU_PLUGIN_TOKEN` is required and must match the host session token.
- `BINDU_PLUGIN_NAME` and `BINDU_PLUGIN_VERSION` must match the activated bundle.
- `BINDU_NODE_ID`, `BINDU_NODE_ROLE`, and `BINDU_SESSION_ID` may be set by the host.
- Token + identity are guaranteed for managed runs; node/session fields are optional.

Lifecycle:
1) Process spawn
2) Handshake (token/identity)
3) Event/control processing loop
4) Shutdown request or host termination

Errors:
- Handler errors are logged by the SDK and processing continues.
- Fatal errors should exit the process (non-zero).
- Capabilities are free-form strings (examples: `timer:cron`, `state:kv`, `fs:readonly`).

## Go SDK Reference
The Go SDK lives under `sdk/` and is intentionally small.
It implements the BPR handshake, loop, and stdio framing.

Public types and helpers:
- `HandshakeInfo`, `Trigger`, `Event`, `Message`
- `Plugin`, `EventHandler`, `Shutdowner`
- `ContextInfo`, `Context`
- `Logger`
- `Transport`, `NewTransport`
- `Run(plugin Plugin) error`
- `LoadConfig(path string) (map[string]any, error)`

Runtime defaults applied by `Run`:
- `Language`: "go" if empty
- `SDKVersion`: `bpr-sdk-go/0.1.0` if empty
- `ContractVersion`: `bpr/v1` if empty

HandshakeInfo fields:
```go
type HandshakeInfo struct {
    Inputs          []string
    Outputs         []string
    Capabilities    []string
    Language        string
    SDKVersion      string
    ContractVersion string
    SchemaSetHash   string
}
```

### Context
`ContextInfo` provides runtime identity:
```go
type ContextInfo struct {
    PluginID      string
    PluginVersion string
    NodeID        string
    Role          string
    SessionID     string
}
```

`Context` methods:
- `Info() ContextInfo`
- `Config() map[string]any` (empty by default; use `LoadConfig`)
- `Emit(messageType string, payload map[string]any) error`
- `Logger() Logger` (stderr logger)
- `Now() time.Time`

Logger behavior:
- Uses the standard library `log.Logger` to stderr with timestamps.
- Do not log the session token.

Config helper:
```go
cfg, err := sdk.LoadConfig("./config.json")
```
Missing file returns an empty map, not an error.

### Inbound Interfaces
Primary interface:
```go
type Plugin interface {
    Handshake() sdk.HandshakeInfo
    OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error)
}
```

Optional interfaces:
```go
type EventHandler interface {
    OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error)
}

type Shutdowner interface {
    Shutdown(ctx *sdk.Context) error
}
```

Event payload structure (from `bpr.EventPayload`):
- `message_type`
- `payload` (JSON object)
- `meta` (optional)

Notes:
- If `EventHandler` is not implemented, events are ignored.
- `OnTrigger` is called for control triggers (cron or manual).
- Returning an error from handlers logs the error and continues the loop.

### Outbound Interfaces
Emit messages:
```go
return []sdk.Message{
    {
        Type: "HelloWorldState",
        Payload: map[string]any{
            "greeting": "hello",
        },
    },
}, nil
```

You may also emit directly from within any handler or asynchronous worker:
```go
_ = ctx.Emit("HelloWorldState", payload)
```

#### Envelope Stamping & Spatial Context ($where) Contract
When `ctx.Emit(messageType, payload)` is called:
- **`fields` Mapping**: The `payload` map (`map[string]any`) is serialized as the `fields` object in the resulting Bindu message envelope.
- **Automatic Supervisor Metadata**: The supervising Bindu node process automatically injects envelope metadata:
  - `"$type"`: set to `messageType`.
  - `"$when"`: current UTC ISO8601 timestamp.
  - `"$producer"`: set to the local node's identifier (`node_id`).
  - `"$group"`: assigned according to the message type contract.

> [!TIP]
> **Spatial Context & `$where` Best Practice**:
> Because the external BPR plugin runs as an isolated OS process communicating over standard I/O, domain spatial data (indoor Cartesian coordinates $X/Y$, reference plane slugs, zone identifiers, antenna angles) should be declared directly inside the Message Type schema's `properties` and passed in `payload`:
> ```go
> payload := map[string]any{
>     "beacon_id":      "beacon-fda50693-100-1",
>     "status":         "SEEN",
>     "zone_id":        "operation-room",
>     "where_locagram": "node-dell-2/operation-room",
>     "coordinates_x":  6.0,
>     "coordinates_y":  4.0,
>     "rssi_dbm":       -62.5,
> }
> _ = ctx.Emit("BLEBeaconSeen", payload)

#### Strongly-Typed Messages & Zero `SCHEMA_INVALID` Guarantee (`types_gen.go`)
Manually writing `map[string]any` dictionaries carries the risk of typos in field names or type mismatches that trigger runtime `SCHEMA_INVALID` rejection events.
To eliminate this risk, `bindu-studio` generates strongly-typed Go structs directly from your message schemas:

```bash
bindu-studio message-types gen-go --config ./poc_message_types.json --out ./types_gen.go --package main
```

**What it generates:**
1. **Type-Safe Fields Structs**: Named `<TypeName>Fields` (e.g., `EpcSgtinReadFields`) with exact JSON tags (`json:"..."`), automated Go initialisms (`EPCHex`, `CanonicalURN`, `AntennaID`, `FirstSeenUTC`), and field comments.
2. **Generic Pointer Helper**: `Ptr[T any](v T) *T` to cleanly assign optional primitive properties.
3. **Safe Serialization Helpers**:
   - `fields.ToMap()`: Converts the struct to `map[string]any` ready for `ctx.Emit()`.
   - `fields.ToMessage(producer, whenRFC3339)`: Creates a full message envelope for testing/MQTT simulation.
   - `Parse<TypeName>Fields(map[string]any)`: Parses inbound event payloads safely.

**Example Usage in Plugin Handlers:**
```go
fields := EpcSgtinReadFields{
    EPCHex:        item.EPCHex,
    CanonicalURN:  result.CanonicalURN,
    Standard:      "SGTIN-96",
    CompanyPrefix: result.CompanyPrefix,
    ItemReference: result.ItemReference,
    SerialNumber:  result.SerialNumber,
    AntennaID:     int64(item.AntennaID),
    ReaderID:      p.readerID,
    PeakRSSIDBm:   item.PeakRSSI,
    FilterValue:   Ptr(int64(result.Filter)), // Ptr helper for optional integer
    FilterDesc:    Ptr(result.FilterDesc),    // Ptr helper for optional string
}

payload, err := fields.ToMap()
if err != nil {
    ctx.Logger().Printf("Marshal error: %v", err)
    return
}
_ = ctx.Emit(MessageTypeEpcSgtinRead, payload)
```
Any schema discrepancy is caught immediately at compile-time by `go build` rather than failing in production.
> ```
> Having spatial coordinates and zone IDs as typed properties inside `fields`:
> 1. Allows pure Zero-RAG SQL queries in Dashboards (`json_extract(raw_json, '$.fields.coordinates_x')`).
> 2. Enables 2D Operational Maps (`view.type: "map"`) to plot positions dynamically.
> 3. Allows CEL kinetics to trigger boundary and dwell rules without external lookups.

Constraints:
- `messageType` must be non-empty and declared in `plugin.json` (`out`).
- Payload is JSON-encoded by the SDK and must satisfy the registered Message Type schema.

### Transport (low-level)
If you need raw BPR envelopes:
```go
t := sdk.NewTransport(os.Stdin, os.Stdout)
env, _ := t.ReadEnvelope()
_ = t.WriteEnvelope(bpr.Envelope{Type: "emit", Payload: payload})
```
Transport methods:
- `ReadEnvelope() (*bpr.Envelope, error)`
- `WriteEnvelope(env bpr.Envelope) error`

Use this only for tooling; `sdk.Run` is the default.

### Scheduling Interfaces
Scheduling uses control triggers:
```go
type Trigger struct {
    ID     string
    Reason string
    When   time.Time
}
```

Bindu sends:
```json
{
  "type": "control",
  "payload": {
    "action": "trigger",
    "trigger_id": "...",
    "reason": "cron|manual",
    "when": "2026-01-10T16:32:23Z"
  }
}
```

SDK behavior:
- Parses `when` as RFC3339.
- If missing/invalid, uses current UTC time.
- Cron uses 6-field format (seconds included).

### SQL Read-Only Interface
Current state (Go SDK):
- The SDK exposes read-only query helpers in `sdk/query_rest.go`.
- These helpers are REST wrappers over `POST /api/v1/messages/sql`.
- Plugins must not access SQLite directly and must not rely on DB paths.

Available methods:
- `ctx.QueryReadOnly(ctx, sql, args...)`:
  - Executes one read-only SQL statement through the existing API.
  - Uses env vars (mandatory):
    - `BINDU_QUERY_API_URL`
    - `BINDU_QUERY_AUTH`
  - Optional env var for HTTPS trust:
    - `BINDU_QUERY_CA_CERT` (PEM CA file path)
  - Runtime injects query URL/auth automatically, and injects CA path when HTTPS query endpoint is used.
- `ctx.QueryPage(ctx, sdk.QueryPageRequest{...})`:
  - Applies client-side pagination (`LIMIT/OFFSET`) when missing.
  - Optional `IncludeTotal` computes `COUNT(*)` via SQL wrapper query.
  - Returns rows + pagination metadata (`Page`, `PageSize`, `TotalRows`, `TotalPages`, `HasNext`, `HasPrev`).

Notes:
- Backend read-only validation is still enforced server-side.
- REST endpoint does not expose native page/cursor fields; paging is encapsulated in SDK wrapper.
- Keep SQL scoped to `v_messages` per backend policy.
- In `v_messages`, message payload is exposed as column `fields` (not `payload`).
- For JSON extraction over message body, use `json_extract(fields, '$.path')`.

Example SDK usage:
```go
rows, err := ctx.QueryReadOnly(ctx2, `
  SELECT "$type", "$when", "$producer"
  FROM v_messages
  WHERE "$type" = ?
  ORDER BY "$when" DESC
  LIMIT 20
`, "HelloWorldState")
```

```go
page, err := ctx.QueryPage(ctx2, sdk.QueryPageRequest{
  SQL: `SELECT "$type", "$when" FROM v_messages WHERE "$group" = ? ORDER BY "$when" DESC`,
  Args: []any{"defaultGroup"},
  Page: 1,
  PageSize: 100,
  IncludeTotal: true,
})
_ = page.Rows
_ = page.TotalPages
```

Example SQL statements:

Example 1: simple SELECT
```sql
SELECT $type, $when, $producer
FROM v_messages
WHERE $type = 'HelloWorldState'
ORDER BY $when DESC
LIMIT 20;
```

Example 2: JSON payload filter
```sql
SELECT json_extract(fields, '$.greeting') AS greeting, $when
FROM v_messages
WHERE $type = 'HelloWorldState'
  AND json_extract(fields, '$.greeting') IS NOT NULL
ORDER BY $when DESC
LIMIT 10;
```

Example 3: time-bounded query over the common view
```sql
SELECT $type, $when, $group
FROM v_messages
WHERE $group = 'defaultGroup'
  AND $when BETWEEN '2026-01-10T00:00:00Z' AND '2026-01-11T00:00:00Z'
ORDER BY $when ASC;
```

Expected results:
- Rows are JSON objects with column names matching the SELECT list.
- Empty results are valid and must be handled gracefully.
- SQL args are supported by API payload (`args`) and SDK methods.
```
{
  "success": true,
  "details": { "rows": [ { "col": "value" } ] }
}
```

### Schema Query Interface
BPR plugins and semantic reactors running on *any* node (including remote gateways) can fetch message type definitions and JSON schemas dynamically.

This is supported by the SDK context methods:
- `ctx.GetMessageTypeSchema(ctx, name)`: Returns a `map[string]any` with the message type configuration, including its properties list (`properties`).
- `ctx.GetMessageTypeJSONSchema(ctx, name)`: Returns the raw JSON Schema (`[]byte`) of the message type envelope.

**How it works under the hood**:
- The runtime automatically starts a **local query server** for the plugin process and registers the `/api/v1/message-types/` and `/api/v1/message-types/{name}/schema` handlers.
- Both BRP plugins and AI reactors can invoke these methods on child/remote gateway nodes without hitting the central coordinator's REST port.
- This allows plugins and AI reactors to perform dynamic type checking, payload coercion, and validation directly at the edge.

## Studio plugin init Artifacts Explained
Observed from `plugins-bpr-demo-go/hello-world`:

1) `plugin.json`
   - Purpose: source-of-truth manifest for the plugin.
   - Used by: Studio certify, import, lifecycle, and developer edits.
   - Editable: yes, but keep in sync with `Handshake()`.

2) `main.go`
   - Purpose: Go entrypoint implementing the BPR SDK.
   - Used by: developer and `go build`.
   - Editable: yes.

3) `go.mod`
   - Purpose: Go module definition for the plugin.
   - Used by: `go build` and tooling.
   - Notes: template is self-contained; no `bindunet` module dependency is required.

4) `README.md`
   - Purpose: next steps and usage notes for the plugin.
   - Editable: yes.

5) `examples/out/<MessageType>.json`
   - Purpose: message sample generated from live registry.
   - Used by: developer reference and payload scaffolding.
   - Editable: yes, but keep consistent with schema.
   - Note: the Go template extracts `fields` from the sample and emits fields-only payloads.
   - Note: samples are full message envelopes (including `$type`, `$when`, etc.).

6) `examples/in/<MessageType>.json` (only if inputs selected)
   - Purpose: sample incoming payloads.
   - Used by: testing and handler logic.
   - Editable: yes.

## Development & Debugging

### Standalone mode
Standalone mode validates the handshake and keeps the process running:
```bash
go build -o hello-world
go run ./cmd/bpr-smoke-host --bin ./hello-world
```
This uses `executioneng.StartBPRSession` to spawn the plugin and validate:
- token binding
- plugin identity
- contract version

### Managed by bindu-root
Use the lifecycle pipeline:
```bash
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz
bindu-studio plugin apply --bundle ./hello-world.tar.gz --target root
```

Then:
- Use Console "Run Now" to trigger a manual run.
- Or wait for cron schedules in `plugin.json`.

Quick iteration tips:
- Rebuild: `go build -o hello-world`
- Re-certify after changes: `bindu-studio plugin certify ...`
- Re-import only when the bundle changes.
- Check bindu-root logs for handshake errors and schedule triggers.

### VS Code debugging
Recommended workflow:
1) Build the plugin with debug symbols:
   - `go build -gcflags "all=-N -l" -o hello-world`
2) Run the smoke host to spawn the plugin:
   - `go run ./cmd/bpr-smoke-host --bin ./hello-world`
3) Attach VS Code Go debugger to the running plugin process.

For bindu-root managed runs:
- Start bindu-root with `./dev.sh`.
- Trigger the plugin via Console "Run Now".
- Attach debugger to the spawned plugin process.

## Certification & Artifacts
Certification is required for import.

Command:
```bash
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz
```

Multi-target examples:
```bash
# explicit targets (slash syntax)
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz \
  --target linux/amd64 \
  --target linux/arm64 \
  --target linux/armv7

# targets from mesh only
bindu-studio plugin certify --source ./hello-world --out ./hello-world.tar.gz \
  --targets-from-mesh
```

Validation performed (from Studio implementation specs and runtime checks):
- `plugin.json` correctness (id, version, runtime, inputs/outputs)
- Runtime compatibility (BPR)
- Cron validity (when schedules are present)
- Forbidden files (secrets, unexpected artifacts)
- Deterministic bundle layout and checksums
- Example fixtures validation: automatically validates all JSON samples in `examples/out/*.json` against message schemas

### Automatic Example Fixture Generation (`message-types gen-examples`)
To ensure your plugin's example fixtures are 100% compliant with the schema registry and pass Level 1 certification without warnings, generate fixtures directly:
```bash
bindu-studio message-types gen-examples --config ./poc_message_types.json --out ./my-plugin/examples/out
```
This generates formatted `<MessageType>.json` files in `./examples/out` conforming strictly to both the message schema and generic envelope. During `plugin certify`, these files are validated automatically.

Expected outputs in the certified bundle:
- `manifest.json` (bundle manifest with checksums)
- `sha256sums.txt` (content digests)
- `bindu.env` (contains `BINDU_PLUGIN_TOKEN`)
- Plugin runtime root (Go bundles use `manifest.json` + `runtime.json` plus binaries)
- `attestations/preflight.json` (when certification is required)

Additional certification output:
- `certify-report.json` alongside the bundle (warnings/errors + checksums)

Metadata embedded in Go bundles:
- Bundle manifest (`manifest.json` at bundle root):
  - `bundleType`, `id`, `version`, `timestamp`, `targets`, `contents`
- Plugin manifest (`manifest.json` inside plugin root):
  - `id`, `version`, `entrypoint`, `messages.consumes/produces`, `capabilities`
- Runtime spec (`runtime.json`):
  - `runtime: go-binary`, `protocol: bpr`, `targets`, `triggers.cron`, timeouts, limits

Import checks (bindu-root):
- Bundle manifest `bundleType` must be `bpr-plugin`.
- `bindu.env` must include a token.
- Bundle contents must match checksums and manifest.

## Training an AI Agent to Build BPR Go Plugins
Minimum context the agent must know:
- BPR is stdin/stdout framed JSON with a handshake.
- `plugin.json` is the source-of-truth manifest at dev time.
- Go SDK entrypoint is `sdk.Run(plugin)`.
- Messages are emitted by returning `[]sdk.Message`.

Expected file layout:
```
my-plugin/
  go.mod
  main.go
  plugin.json
  README.md
  examples/
    out/
```

Typical pattern:
1) Initialize plugin via Studio.
2) Implement `Handshake()` with declared inputs/outputs.
3) Implement `OnTrigger()` and optional `OnEvent()`.
4) Emit messages using `sdk.Message` or `Context.Emit`.
5) Certify and import.

Validation checklist:
- Handshake inputs/outputs match `plugin.json`.
- No undeclared message types emitted.
- Plugin exits on handshake rejection.
- Logs go to stderr, not stdout.

Common mistakes to avoid:
- Logging protocol data to stdout (breaks framing).
- Missing `BINDU_PLUGIN_TOKEN`.
- Emitting messages that are not declared in `plugin.json`.
- Forgetting to re-certify after changes.

## Notes, Gaps, and Observations
- The Go SDK does not expose handshake response context (`bpr.HandshakeResponse.Context`), even though the protocol provides it.
- The Go SDK does not implement a query request API; read-only SQL is only documented for the REST query console.
- The Go template example does not include `bindu.env` in the source tree, but import requires it in certified bundles.
- `plugin.json` field names in the Go example (`out`, `schedules`) differ from older docs; treat the generated artifact as authoritative.

## Appendix (if needed)
Standalone trigger harness (optional):
```go
sess, _ := executioneng.StartBPRSession(ctx, executioneng.BPRSessionOptions{
    BinaryPath: "./hello-world",
    PluginID: "hello-world-id",
    PluginVersion: "0.1.0",
})
_ = sess.SendControl(bpr.ControlPayload{
    Action: "trigger",
    TriggerID: "manual",
    Reason: "manual",
    When: time.Now().UTC().Format(time.RFC3339),
})
```
