## 🤖 Message Types for AI Agents

This section explains how **AI agents** should reason about Message Types in Bindu.

Bindu is **AI-driven by design**: Message Types are meant to be *understood, authored, and extended* by AI systems — safely and deterministically.

---

## The AI Contract

> **Message Types are not code.**  
> **They are semantic contracts.**

An AI agent must treat a Message Type as a **declaration of intent**, never as an imperative program.

The platform guarantees execution, safety, and determinism.

---

## How an AI Should Read a Message Type

An AI agent should always analyze a Message Type along **five independent axes**:

```text
1. Identity     → What does this message represent?
2. Storage      → Should it be remembered?
3. Retention    → How long does it matter historically?
4. Routing      → Who should know about it?
5. Behavior     → What happens because of it?
```

Each axis is explicit in the schema and must not be inferred implicitly.

---

## Axis 1 — Identity (Semantic Meaning)

**Fields to read:**

* `name`
* `description`
* `properties`
* `version`

**AI guidance:**

* Treat the name as a **domain concept**, not a technical identifier.
* Use properties to understand what facts are being expressed.
* Never infer hidden meaning from property names alone.

**Correct reasoning:**

> “InventoryUpdate expresses a business fact about stock levels.”

**Incorrect reasoning:**

> “This looks like a database row, so I’ll store it forever.”

---

## Axis 2 — Storage Semantics

**Fields to read:**

* `persist`

**AI guidance:**

* `persist: false` means **ephemeral signal**
* `persist: true` means **stateful fact**
* Never override persistence by emitting storage-heavy pipelines unless explicitly required.

**Correct reasoning:**

> “This is a transient signal; do not assume historical availability.”

---

## Axis 3 — Retention Semantics (MAS vs BI Philosophy)

**Fields to read:**

* `retentionMode` (`count`, `days`, `none`)
* `retentionValue` (integer)

**Architectural Law: Real-Time MAS vs Business Intelligence (BI)**:

* **Bindu is a Multi-Agent System (MAS) for real-time edge intelligence and autonomous decision-making — NOT a Business Intelligence (BI) data warehouse, time-series lake, or cold-storage historian.**
* Retention defines a **short-term local sliding memory window** in SQLite to support real-time edge evaluation (e.g., sliding window aggregations, reactor condition evaluation, anomaly deltas).
* Hard validation bounds enforced by the runtime:
  * `count`: strictly between `1` and `100` messages.
  * `days`: strictly between `1` and `30` days.
  * `none`: `retentionValue` must be `0`.

**AI guidance:**

* **Triggers and Command Events** (e.g., `MeshAuditTrigger`, `InitReader`): Must use `retentionMode: "count", retentionValue: 1` (or `1..3`). Triggers represent actionable intent; once processed by reactors, they do not require historical backlog accumulation.
* **Telemetry and Periodic Signals** (e.g., `FreezerTelemetry`, `SensorPing`): Use small windows such as `count: 1..20` or `days: 1..7` for sliding state context.
* **Never treat Bindu as a BI data lake**: Do not attempt to store months/years of historical raw events inside edge nodes.
* **External BI as a Message Producer**: If deep historical analytics or machine learning predictions are required, an external BI/analytics warehouse processes the history and publishes distilled actionable facts **into Bindu as a Message Producer** (e.g., `RiskForecast`, `OptimalOperatingSchedule`) for edge agents to act upon in real time.

**Correct reasoning:**

> “This message is an audit trigger. Its retention should be count: 1 because edge agents only need the latest trigger for real-time evaluation.”

**Incorrect reasoning:**

> “I will set retentionValue: 500 to keep a full historical archive for BI analytics in SQLite.” *(Fails schema validation and violates MAS architecture).*

---

## Axis 4 — Routing Semantics

**Fields to read:**

* `routing`
* `longevity`

**AI guidance:**

* Routing expresses **semantic relevance**, not transport.
* Current routing is an explicit list of directions using:
  * `local`
  * `parent`
  * `child`
* `local` ≠ private forever — only locally relevant.
* `parent` means upstream visibility.
* `child` means downstream visibility, optionally narrowed by `$node-dest`.
* `longevity` defines delivery relevance during outages.

**Correct reasoning:**

> “This event is important upstream, but becomes meaningless after 5 minutes.”

**Incorrect reasoning:**

> “I can resend this anytime.”

---

## Axis 5 — Lifecycle & Behavior Semantics

**Fields to read:**

* `execution`
* `x-pipeline`
* `x-*`

**AI guidance:**

* `x-pipeline` is **declarative logic**, not code.
* Each step should represent one semantic transformation or emission.
* Avoid circular emissions unless explicitly guarded.
* Never mutate message intent; only derive or emit new messages.

**Correct reasoning:**

> “This message triggers a derived event when conditions are met.”

**Incorrect reasoning:**

> “I’ll modify the original message fields arbitrarily.”

---

## What AI Agents Must **NEVER** Do

An AI agent must never:

* invent new top-level fields not in the schema
* bypass `additionalProperties: false`
* emit messages of undefined Message Types
* ignore routing or longevity semantics
* assume storage when `persist: false`
* create infinite emission loops
* treat Message Types as mutable at runtime
* set `retentionValue > 100` for `count` or `> 30` for `days` (violates MAS validation bounds)
* configure large retention buffers attempting to turn edge nodes into BI data warehouses or cold storage historians

---

## What AI Agents **ARE Allowed** to Do

An AI agent may:

* create new Message Types (via editor or API)
* extend Message Types using `x-*` namespaced fields
* generate `x-pipeline` steps declaratively
* reason about behavior compositionally
* suggest schema evolution via version bumps
* generate sample messages for testing

---

## AI-Friendly Design Guarantees in Bindu

Bindu guarantees that:

* Message Types are fully schema-validated
* Runtime behavior is deterministic
* Semantics are explicit (no hidden side effects)
* Invalid messages are rejected early
* Pipelines are sandboxed and declarative

This allows AI agents to operate safely without full system context.

---

## Mental Model for AI Agents

A Message Type is a **law of the system**, not a suggestion.

* Messages are **facts**
* Pipelines are **consequences**
* Routing is **relevance**
* Retention is **memory**

---

## Example — AI Reasoning Summary

**Given:**

```json
{
  "name": "TemperatureReading",
  "persist": false,
  "routing": ["local", "parent"],
  "longevity": "PT5M"
}
```

**Correct AI interpretation:**

* This is a signal, not a record
* It should be forwarded upstream
* It is only relevant for 5 minutes
* It should not be relied on for historical queries
* Any derived state must be stored separately

---

## 🏛️ System Message Types vs Application Message Types

An AI agent must strictly distinguish between **System Message Types** and **Application Message Types**:

| Characteristic | ⚙️ System Message Types (Tier 1) | 📦 Application Message Types (Tier 2) |
| :--- | :--- | :--- |
| **Origin & Purpose** | Built-in infrastructure contracts (`MessageState`, `NodePresence`, `SystemAlert`, `SystemScheduledEvent`, `SystemCronJob`, `SystemNodeReset`, etc.) | Domain-specific business contracts (`ColdChamberTelemetry`, `PotatoDefrostAlert`, etc.) |
| **Location** | Compiled in Go code (`messagetypes/messagetypes.go` $\rightarrow$ `SystemMessageTypes`) | Declared in JSON (`messageTypesConfiguration.json`, `MASSolutionSet`) |
| **Distribution / Updates** | Distributed via **Binary Release Updates** (`binduUpdate`, `binduNodeLauncherUpdate`). AI agents do not modify them via JSON. | Hot-reloaded in memory via `message-types merge` and rolled out via the **Runtime OTA Triptych** (`baseline accept` $\rightarrow$ `desired commit` $\rightarrow$ `update wizard`). |
| **AI Modification Rule** | ❌ **Immutable**: AI agents must never attempt to create or override system message types via solution bundles. | ✅ **Authorable**: AI agents actively scaffold, validate, and evolve these domain contracts. |

---

## Final Rule

> **If an AI cannot explain why a field exists, it must not invent behavior for it.**

Message Types are the **shared language** between humans, AI, and the Bindu runtime.

Use them with precision.

> **Bindu is not programmed — it is declared.**

---
