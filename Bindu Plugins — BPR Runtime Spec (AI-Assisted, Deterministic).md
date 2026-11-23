Bindu Plugins — BPR Runtime Spec (AI‑Assisted, Deterministic)

Platform positioning: **BREP (Bindu Runtime Extension Platform)**.
BREP is the canonical runtime-extension model for Bindu.

Version: v1.0
Status: Active (Current Model)
Replaces: Bindu BPR Plugins — Go Runtime Spec

1. Purpose

This document defines the current and only supported plugin model in Bindu.

It exists to ensure that:

Plugins are deterministic, portable, and governable

Plugin behavior is semantically declared, not implicitly coded

AI agents can safely generate and reason about plugins

Tooling, packaging, and lifecycle are uniform and enforceable

This specification eliminates BPR entirely.

All plugins run via BPR (Bindu Plugin Runtime) and are managed exclusively through Bindu Studio.

2. Core Principles

Semantics before execution
Plugins exist to react to meaning, not to expose services.

Isolation by default
Plugins are external OS processes. They never run inside Bindu.

Determinism over convenience
Every input, output, and side effect is declared.

AI‑assisted, not AI‑controlled
AI may generate logic, but the system enforces boundaries.

One runtime, one lifecycle
BPR + Bindu Studio are the only supported path.

3. What a Bindu Plugin Is

A Bindu plugin is:

A reactive computational actor that consumes messages and emits new messages.

A plugin:

reacts to Message Types

performs computation

emits new Message Types

A plugin does not:

manage routing

own persistence

control topology

call other plugins

mutate system state

Bindu remains authoritative at all times.

4. Execution Boundary & Sandboxing (BPR)

Plugins run as separate OS processes and communicate with Bindu exclusively through BPR.

BPR Characteristics:
- JSON messages over stdin/stdout
- strict framing and ordering
- explicit handshake and session token verification
- explicit error signaling
- no shared memory, no direct DB access, no RPC framework
- **Environment Whitelisting:** Child plugin processes receive only sanitized system variables (`PATH`, `TMPDIR`, `LANG`, `BINDU_PLUGIN_*`). Sensitive host credentials and master keys (`BINDU_MASTER_KEY`) are stripped.
- **Process Containment:** Plugins run in an isolated process group (`Setpgid: true`) and are bound to the parent daemon lifecycle via `Pdeathsig: syscall.SIGKILL`. If Bindu exits, the kernel immediately terminates all running plugin processes.

5. Communication Model
5.1 Handshake

On startup, the plugin:

reads runtime environment variables

initializes internal state

signals readiness via BPR handshake

If the handshake fails, the plugin is rejected.

5.2 Message Flow

Messages arrive via BPR as validated JSON

Messages always conform to a Message Type schema

Plugins never mutate incoming messages

Plugins emit new messages as consequences

6. Authentication & Identity

Plugins authenticate using a root‑issued plugin token.

Rules:

Tokens are provided via environment variables

Tokens define identity and permissions

Plugins must never generate or store tokens

Authentication is mandatory.

7. Message‑Centric Contract

Plugins must declare:

Message Types they consume (IN)

Message Types they produce (OUT)

Rules:

A Message Type may not appear in both IN and OUT

Produced Message Types must exist in the system

Routing, retention, and permissions are defined by Message Types — not plugins

Plugins adapt to Message Types, never the reverse.

8. Cron Semantics

Cron is declared, not implemented.

Cron schedules live in plugin.json

Bindu triggers cron events

Cron events arrive as messages

The plugin reacts like any other message.

9. Queries (Read‑Only)

Plugins may execute SELECT‑only queries.

Constraints:

Queries are executed by Bindu

No side effects allowed

Plugins never access databases directly

10. Plugin Manifest (plugin.json)

plugin.json is the single source of truth.

Example:

{
  "id": "example-plugin",
  "version": "0.1.0",
  "description": "Example Bindu plugin",
  "entrypoint": "python3 main.py",
  "runtime": "bpr",
  "messageTypesIn": ["InputEvent"],
  "messageTypesOut": ["OutputEvent"],
  "cron": "0 */5 * * * *"
}

Rules:

Exactly one plugin.json

No runtime.json

No transport‑specific config

11. Source Tree (Author View)
my-plugin/
  plugin.json
  main.<language>
  README.md
  (optional dependency files)

Canonical authoring path uses `bindu-studio plugin init` (wizard or non-interactive).

12. Packaging Contract (BPR Bundle)

Plugins are distributed as deterministic .tar.gz bundles.

Bundles are created only via Bindu Studio:

bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz

The bundle contains:

normalized plugin.json

plugin source files

bindu.env

certification metadata

Manual packaging is not supported.

13. Lifecycle (Bindu Studio)

Bindu Studio is the only supported lifecycle tool.

Typical flow:

bindu-studio plugin init
bindu-studio plugin doctor
bindu-studio plugin certify
bindu-studio plugin apply

All state transitions are explicit and auditable.

14. Failure Model

Plugins are allowed to:

crash

restart

be slow

Bindu guarantees:

isolation

supervision

observability

System stability is never compromised by plugin failure.

15. AI‑Assisted Development Model

AI agents may:

generate plugin logic

reason about Message Types

propose pipelines

generate sample messages

AI agents must not:

invent semantics

bypass schema validation

emit undeclared Message Types

assume storage or routing

The system enforces correctness.

16. Non‑Goals (Explicit)

No BPR

No in‑process plugins

No dynamic code loading

No runtime compilation

No hidden side effects

Final Principle

Bindu plugins are not extensions of the core.
They are governed actors reacting to declared meaning.

This is what makes Bindu safe for humans — and for AI.
