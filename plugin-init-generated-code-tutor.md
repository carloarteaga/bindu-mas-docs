# Bindu Studio Plugin Init - Generated Code Tutor

This page explains, in one pass, what `bindu-studio plugin init` generates and where each concern should live.

## 1) Generate a baseline plugin

```bash
bindu-studio plugin init --profile ai-safe --id my-plugin
```

### 1.1 Interactive wizard sequence

When run interactively, `plugin init` asks in this order:
1. `Plugin ID`
2. Optional `Plugin name`
3. `Plugin version` (default `1.0.0`)
4. Output directory
5. Input message types
6. Output message types (required)
7. Optional README query examples
8. Optional read-only query scaffold
9. Optional schedules (preset/manual, repeatable)

The generated project is a **starter scaffold**:
- contract and files are valid
- compile should pass
- author zones may contain `TODO` markers intentionally

This means immediate strict pre-release doctor can fail until TODOs are implemented.

Recommended immediate sequence after interactive init:

```bash
bindu-studio plugin doctor --source ./my-plugin
# edit main.go author zones, replace TODOs
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json
```

For the dual-cron prototype:

```bash
bindu-studio plugin init \
  --no-interactive \
  --profile ai-safe \
  --template hello-world-dual-cron \
  --id hello-world-plugin \
  --outdir codex-plugins/hello-world-plugin
```

## 2) Message Contract (Canonical for this tutor)

Source of truth in repo:
- `export-message-types/export-message-types.json`
- Installed/docs bundle copy:
  - `public/console/docs/hello-world-tutorial/hello-world-message-types.json`

Before running this tutor end-to-end, these types must exist in the runtime registry:

- `HelloWorld` (v1.0.0, retention `count:1`, routing `local,parent`, persist `true`)
- `HelloWorldConfig` (v1.0.0, retention `count:1`, routing `local,child`, persist `true`)
- `HelloWorldStats` (v1.0.0, retention `count:1`, routing `local,parent`, persist `true`)

Recommended validation/import sequence:

```bash
bindu-studio message-types list
bindu-studio message-types import --in-path ./export-message-types/export-message-types.json
bindu-studio message-types list
```

If you are using an installed root (outside repo), import from shipped docs JSON:

```bash
# run from <BaseDir> (for example ~/.bindu/root)
bindu-studio message-types import --in-path ./releases/<active-version>/public/console/docs/hello-world-tutorial/hello-world-message-types.json
```

Notes:
- The file `./export-message-types/export-message-types.json` is produced by `bindu-studio message-types export`.
- The file `hello-world-message-types.json` is shipped with root console docs so operators can import without repo source.
- Re-import is safe for this tutor workflow and keeps local/runtime contracts aligned.

This tutor expects these message types:

1. `HelloWorldConfig`
- Purpose: runtime configuration for greeting text.
- Required field:
  - `greeting_text` (string)
- Retention: `count:1`
- Routing: `local,child`
- Persist: `true`

2. `HelloWorld`
- Purpose: periodic output message every 5 minutes.
- Required field:
  - `greeting` (string)
- Retention: `count:1`
- Routing: `local,parent`
- Persist: `true`

3. `HelloWorldStats`
- Purpose: periodic stats output every 15 minutes.
- Required fields:
  - `count` (integer)
  - `since` (date-time string)
  - `until` (date-time string)
- Retention: `count:1`
- Routing: `local,parent`
- Persist: `true`

### Manual creation in Console (`Message Types > New`) - Copy/Paste JSON

If you prefer not to import, create each type manually with these JSON definitions.

`HelloWorld`:

```json
{
  "name": "HelloWorld",
  "emoji": "👍🏼",
  "version": "1.0.0",
  "properties": [
    {
      "name": "greeting",
      "type": "string",
      "required": true,
      "description": "Greeting field"
    }
  ],
  "retentionMode": "count",
  "retentionValue": 1,
  "routing": [
    "local",
    "parent"
  ],
  "persist": true
}
```

`HelloWorldConfig`:

```json
{
  "name": "HelloWorldConfig",
  "emoji": "👍🏼",
  "version": "1.0.0",
  "properties": [
    {
      "name": "greeting_text",
      "type": "string",
      "required": true,
      "description": "Gretting text for hello world msg"
    }
  ],
  "retentionMode": "count",
  "retentionValue": 1,
  "routing": [
    "local",
    "child"
  ],
  "persist": true
}
```

`HelloWorldStats`:

```json
{
  "name": "HelloWorldStats",
  "emoji": "👍🏼",
  "version": "1.0.0",
  "properties": [
    {
      "name": "count",
      "type": "integer",
      "required": true,
      "description": "Number of HelloWord received."
    },
    {
      "name": "since",
      "type": "string",
      "required": true,
      "description": "Start of  the window time.",
      "format": "date-time"
    },
    {
      "name": "until",
      "type": "string",
      "required": true,
      "description": "End of the window time.",
      "format": "date-time"
    }
  ],
  "retentionMode": "count",
  "retentionValue": 1,
  "routing": [
    "local",
    "parent"
  ],
  "persist": true
}
```

## 3) Runtime Sequence (What happens in order)

1. Plugin process starts.
2. First `cron` tick (5 min):
- Runs config bootstrap query for latest `HelloWorldConfig`.
- If found, uses `greeting_text`.
- If not found, keeps default `Hello World`.
- Emits `HelloWorld`.
3. `cron.2` tick (15 min):
- Queries previous `HelloWorldStats.until` (continuity anchor).
- Counts `HelloWorld` in `[since, until)`.
- Emits `HelloWorldStats`.
4. Any inbound `HelloWorldConfig` event:
- `OnEvent` updates greeting in memory immediately.
- Next `HelloWorld` uses new value.

### Runtime flow (ASCII)

```text
HelloWorldConfig (persisted, count:1) ---> OnEvent updates greeting
                     |
                     +--> first cron run bootstrap query (latest config)
                                |
                                +--> found? yes: greeting_text
                                |            no: default "Hello World"
                                v
                         emit HelloWorld (5m)
                                |
                                v
             count HelloWorld in window [since, until)
                                |
                                v
                      emit HelloWorldStats (15m)
```

## 3.1) Logging Rule (Critical)

BPR plugins use framed JSON over stdio. Keep this invariant:

- `stdout`: protocol frames only (managed by SDK transport)
- `stderr`: human logs

Use these for logs:

- `ctx.Logger().Printf(...)` inside handlers
- `log.Printf(...)` in `main()` lifecycle paths

Do not use `fmt.Printf` / `fmt.Println` for runtime logs, because that writes to `stdout` and can corrupt plugin protocol framing.

## 4) File-by-file map (Detailed)

- `plugin.json`
  - Declarative plugin contract.
  - Defines identity (`id`, `version`, `name`), runtime mode, inputs/outputs, and schedules.
  - In dual-cron template:
    - input: `HelloWorldConfig`
    - outputs: `HelloWorld`, `HelloWorldStats`
    - schedules carry stable trigger IDs (`cron`, `cron.2`).
  - What to edit here:
    - Message contract (`in`, `out`)
    - Cron cadence (`schedules`)
  - What not to edit casually:
    - `runtime` and contract shape that must match `Handshake()`

- `main.go`
  - Executable plugin logic.
  - `Handshake()` must match the contract declared in `plugin.json`.
  - `OnTrigger()` branches by trigger ID and emits outputs.
  - Bootstraps `HelloWorldConfig` via read-only query on first run.
  - Uses default greeting when no config exists yet.
  - `OnEvent()` updates greeting when `HelloWorldConfig` messages arrive.
  - Read-only SQL must use SDK wrappers (`QueryReadOnly`, `QueryPage`) only.
  - Key functions in this tutor:
    - `ensureGreetingConfig`: one-time bootstrap query
    - `buildHelloWorldStats`: windowed count query
    - `extractGreetingText`: tolerant payload parsing (`greeting_text` / `fields.greeting_text`)

- `go.mod`
  - Standalone Go module for plugin build/certification.
  - Keeps plugin build isolated from host repo structure.
  - Should stay minimal and deterministic.

- `README.md`
  - Operator runbook for this plugin.
  - Should explain behavior, required env vars, and certify/deploy commands.
  - This is the first document operators should read.

- `examples/out/*.json`
  - Output message envelope examples for humans and AI agents.
  - Useful as copy-ready test payload references.
- `examples/in/*.json`
  - Input envelope examples used by event handlers.
  - Includes `HelloWorldConfig.json` in dual-cron template.
  - Useful for console/manual publish tests.

## 5) Where to define each concern

- Message schema contract: `plugin.json` + `Handshake()` in `main.go`
- Scheduling and trigger IDs: `plugin.json` `schedules[*].trigger`
- Business behavior: `OnTrigger()` and helper functions in `main.go`
- Read-only data access: SDK query wrappers in `main.go` (never direct SQLite access)

## 6) Read-only query model

- Preferred path is REST query API through SDK wrappers.
- Keep SQL read-only and scoped to stable read views (for example `v_messages`).
- Support pagination where needed with `QueryPage`.
- In `v_messages`, use column `fields` for message body JSON (`json_extract(fields, '$.x')`), not `payload`.

## 7) Validation pipeline

```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --json
```

Certify target notes:
- `--target` uses `os/arch` (slash syntax), one per flag.
- `--targets-from-mesh` includes only mesh platforms, not all Linux combinations.
- For Raspberry Pi 32-bit OS, use canonical `linux/armv7` (legacy `linux/arm` remains a compatibility alias).

## 8) Optional scaffold controls

- `--template hello-world-dual-cron`: generates two-cron example and trigger-based routing.
- `--with-readonly-query`: force add read-only query scaffold.
- `--no-readonly-query`: disable read-only scaffold even under `ai-safe`.

## 9) Practical rule of thumb

If a human or AI asks "where do I change X?":

- "Contract?" -> `plugin.json`
- "Runtime logic?" -> `main.go`
- "Sample output?" -> `examples/out/*.json`
- "Build/package?" -> `go.mod` and `bindu-studio plugin certify`
