# 📡 Publish Message — User Guide (Updated)

## Overview
**Publish Message** lets you send fully validated messages into Bindu over MQTT. It’s the developer/integrator workbench for simulation, live testing, and end‑to‑end verification—**no Trigger Rule is required**.

What you can do here
- Emulate devices/clients and publish messages to the platform
- Exercise rules and pipelines end‑to‑end
- Verify Message Type schemas and error handling
- Capture **canonical samples** for docs and regression tests
- Troubleshoot by watching **ErrorEvent** feedback in real time

---

## Where to find it
**Menu → Messages → Publish Message**

---

## Quick start
1. **Select a Message Type.** The console loads a valid **sample** and the type’s **JSON Schema**.
2. **Edit the JSON** in the editor. Validation runs in real time (see the banner).
3. When the banner shows **No validation errors.** and **Send** is enabled, click **Send Message**.  
   The message is published to `bindu/in`.
4. Watch the **Errors pane**: the console is subscribed to `ErrorEvent/#`. Any server‑side error appears here.

---

## The editor (superpowers)
- **Schema‑aware validation (Draft‑07)** applied 1:1 to the open document.  
  Banner: ✅ **green** (OK) · ⚠️ **yellow** (syntax‑only) · ❌ **red** (errors).
- **Clickable error list** in the banner focuses the relevant field in the editor.
- **Send button guards**: enabled only when the JSON **parses**, the **schema is applied**, and there are **zero markers**.
- **RFC‑3339 date‑time** guidance (e.g., `2025-01-01T00:00:00Z`). The editor warns about “smart dashes” or quotes; some builds include a **Normalize dates** action to auto‑fix common copy/paste issues.
- **Fast feedback**: brief **Validating…** state while Monaco recalculates markers after edits or type changes.
- *(Optional)* A **Generate $cid** action may be available to insert a UUID in the envelope for app‑level tracing (see “Semantic correlation”).

---

## Bindu message envelope
Every published message uses the Bindu envelope plus your payload in `fields`:

```json
{
  "$type": "MyMessageType",
  "$producer": "exampleProducer",
  "$user": "exampleUser",
  "$group": "defaultGroup",
  "$when": "2025-01-01T00:00:00Z",
  "$where": "exampleLocation",
  "fields": { /* type-specific data */ }
}
```

> The dynamic schema enforces `$type` to match the selected Message Type.

---

## Developer workflow (recommended)
1. **Pick the type** → get **sample + schema** automatically.  
2. **Tweak the sample** until the banner is green.  
3. **Send** and observe **ErrorEvent**. Iterate until no errors.  
4. **Capture the message** (copy the exact JSON you used) as your team’s **canonical sample**.  
5. Repeat for success and failure cases to build a “golden corpus” for regression.

---

## ErrorEvent — structure, when it fires, and codes

**What it is**  
`ErrorEvent` is a system message the platform publishes on MQTT whenever a published payload can’t be accepted (auth/ACL, topic misuse, schema problems, DB issues, etc.). The console is subscribed to `ErrorEvent/#`. The type is registered as a **system** Message Type with fixed fields.

**Envelope**  
(standard Bindu envelope; the server may also set `$arriveTime` on accepted events)
```json
{
  "$type": "ErrorEvent",
  "$producer": "bindu/system",
  "$user": "system",
  "$group": "system",
  "$when": "2025-01-01T00:00:02Z",
  "$where": "core/validator",
  "fields": { ... }
}
```

**Fields (current implementation)**  
All fields are strings unless noted; `timestamp` is `string` with `format: "date-time"`:
- `group_id` — group that triggered the error  
- `client_id` — MQTT Client ID  
- `username` — MQTT username (if any)  
- `origin_producer` — original `$producer` of the failing message  
- `topic` — the topic involved  
- `reason` — human‑readable error detail  
- `payload` — original JSON payload (stringified)  
- `error_type` — programmatic error code  
- `timestamp` — RFC‑3339 timestamp

**When the server emits ErrorEvent**

- **During publish handling (`bindu/in`):**
  - Invalid JSON → `INVALID_JSON`  
  - Missing or empty `$type` → `MISSING_TYPE`  
  - Unknown `$type` → `UNKNOWN_TYPE`  
  - Schema failure (envelope/fields) → `SCHEMA_INVALID`  
  - Serialization failure → `SERIALIZATION_ERROR`  
  - Badger spool enqueue conflict → `SPOOL_CONFLICT`
  - other DB/storage failure → `DB_ERROR`  
  Each case emits an ErrorEvent with `reason`, `error_type`, and the original `payload`.

- **During ACL checks (publish/subscribe):**
  - Publish to a topic other than `bindu/in` → `TOPIC_INVALID`  
  - Subscription topic malformed (levels) → `WRONG_SUB_TOPIC`  
  - Unknown message type in subscription → `UNKNOWN_TYPE`  
  - ACL denied (no permission) → `ACL_DENIED`

**Cycle protection**  
If a message arrives with `$rebroadcast: true`, the dispatcher skips re‑emitting it to avoid loops. Accepted messages get `$arriveTime` added server‑side before the platform re‑publishes to `$type/$group/$producer`.

**Suggested schema sketch (informative)**  
Adjust as your deployment evolves:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ErrorEvent",
  "type": "object",
  "required": ["$type", "$producer", "$user", "$group", "$when", "fields"],
  "properties": {
    "$type": { "const": "ErrorEvent" },
    "$producer": { "type": "string" },
    "$user": { "type": "string" },
    "$group": { "type": "string" },
    "$where": { "type": "string" },
    "$when": { "type": "string", "format": "date-time" },
    "$arriveTime": { "type": "string", "format": "date-time" },
    "fields": {
      "type": "object",
      "required": ["group_id","client_id","origin_producer","topic","reason","error_type","payload","timestamp"],
      "additionalProperties": false,
      "properties": {
        "group_id": { "type": "string" },
        "client_id": { "type": "string" },
        "username": { "type": "string" },
        "origin_producer": { "type": "string" },
        "topic": { "type": "string" },
        "reason": { "type": "string" },
        "payload": { "type": "string" },
        "error_type": { "type": "string" },
        "timestamp": { "type": "string", "format": "date-time" }
      }
    }
  },
  "additionalProperties": false
}
```

---

## Semantic correlation in Bindu (why we don’t need transport tricks)

Raw MQTT is a great wire, but it’s intentionally “mute”: topics carry no schema, payloads are opaque, and conventions live in meetings and tribal knowledge. Bindu is different: it brings a **semantic envelope** and **typed messages** (with JSON Schema) so you can correlate, reason, and automate with business meaning—no protocol hacks required.

What makes Bindu “alive”:
- `$type`: the event’s identity (e.g., `InitRFIDReader`, `ConfigRFIDReader`, `OrderEntryMsg`). Types are validated and versioned.
- `$producer` and `$group`: who/what emits the event and the access/tenant context it belongs to.
- `$where` and `$when`: spatiotemporal anchors (where it happened, when it happened).
- `fields`: strongly validated, declared by the Message Type schema.

With just these, developers and integrators can correlate messages **by intent** instead of by transport‑level properties.

### Practical correlation patterns
1) **By lifecycle (message pairs/flows)**  
   Example: send `InitRFIDReader` → expect `ConfigRFIDReader`.  
   Correlate by (`$producer`, `$group`) and a short time window around `$when`.  
   Add a stable domain ID in `fields` when it matters (`order_id`, `device_id`, etc.)—that ID travels across types.

2) **By actor/context**  
   (`$producer`, `$group`) define the actor and tenancy.  
   Query “all events from reader X in group Y” and refine by `$type` or time.

3) **By location and time**  
   `$where` (site/zone/location) + `$when` give robust spatiotemporal views (timelines, dashboards, SLAs).

4) **By business intent (types as contracts)**  
   Message Types are contracts, not just strings: they validate shape and meaning.  
   A chain like `PickStarted` → `PickCompleted` → `InventoryAdjusted` is naturally correlated by `$producer`/`$group` and domain IDs in `fields`.

5) **Error correlation without extra IDs**  
   `ErrorEvent` includes context like origin producer/group/topic and a clear `error_type`.  
   You can correlate “what I just sent” with “what failed” by matching time window, `$producer`/`$group`, and the intended `$type` (from your sent payload), without requiring any extra correlation field.

### Design tips for Message Types (to keep correlation strong)
- Choose meaningful `$type` names that reflect intent (`Init…`, `Config…`, `Ack…`, `Error…`, `StateChanged…`).
- Keep field names stable and in `snake_case`; include domain identifiers where needed (`order_id`, `device_id`).
- Use `version` to evolve schemas without breaking flows; rules can branch by version if needed.
- Keep `$where`/`$when` accurate; they are the backbone of timelines and SLAs.
- Prevent loops with `$rebroadcast` (already supported by the platform).

### Optional note on `$cid` and MQTT 5 properties
- You don’t need any transport‑level property for correlation in Bindu.  
- If a publisher wants extra, app‑level tracing, it may include an **optional** `$cid` in the envelope; the server can echo it in `ErrorEvent.fields.correlation_id`. This is strictly optional and adds convenience for certain workflows, not capability.  
- MQTT 5 features (`Response Topic`, `Correlation Data`) remain available at protocol level if a client chooses to use them; they are **not required** for Bindu’s semantic correlation.

---

## MQTT topics
- **Publish**: `bindu/in` (the console sends here)  
- **Errors**: `ErrorEvent/#` (the console listens here)  
- Your application may additionally publish acks or domain events—subscribe as needed.

---

## Tips & best practices
- Prefer **snake_case** fields for portability across tooling.
- For date‑times, use **`type: "string"` + `format: "date-time"`** with real RFC‑3339 values; avoid empty defaults.
- Keep `ErrorEvent.fields.error_type` **stable** for tooling and dashboards.
- Include a **correlation_id** only if you already use `$cid` (optional); otherwise rely on semantic correlation.
- Save the final, successful JSON as the team’s **canonical sample**.

---

## Troubleshooting
- **Send disabled**: schema not applied yet, JSON syntax error, or markers present. Fix until the banner is green.
- **Invalid date‑time**: replace smart dashes (`–`, `—`) with normal `-`; ensure `YYYY-MM-DDThh:mm:ssZ`.
- **Additional properties**: your `fields` contains a name not defined in the Message Type.
- **Banner stuck in “Validating…”**: a small edit or reselecting the type forces a re‑run of validation.

---

## Appendix — Developer workflow recap
1. Select a Message Type → review the **sample** and **schema**.  
2. Edit, validate (green banner), and **Send**.  
3. If an **ErrorEvent** appears, inspect `error_type`/`reason`, fix, and retry.  
4. Capture both **successful** and **expected‑error** payloads as reference cases.  
5. Automate regression tests using those canonical samples.
