# 🧾 Understanding Messages in Bindu

Messages in Bindu are structured objects that flow through the system. They are validated at runtime and composed of two main layers:

1. ✅ **Generic Envelope** → defined in `genericMessageSchema.json`
2. 🧩 **Message Type Fields** → defined in `messageTypesConfiguration.json`

Together, they form a **Message Sample**, which can be published to a topic like:

```
InventoryUpdate/store1/device01
```

---

## 📦 What is a Message Sample?

A **Message Sample** is the combination of:

- Generic metadata (`$type`, `$producer`, `$when`, etc.)
- Dynamic fields from the message type definition (e.g. `temperature`, `sku`, `level`, etc.)

> The generic part defines **what, when, who, where** — the semantic context.
>
> The message type defines **what's being measured or sent**.

---

## ✅ Generic Envelope Fields (Required by All Messages)

These fields are **required** in every message, regardless of type, and are validated by the Enforcer:

| Field       | Type      | Description & Ingestion Behavior                                  |
| ----------- | --------- | ----------------------------------------------------------------- |
| `$type`     | string    | Unique PascalCase name of the message type (e.g. `BatteryLevel`) |
| `$producer` | string    | **Logical Producer of the payload**: <br>- **BPR Plugins**: set to the plugin name (e.g. `hello-world-plugin`).<br>- **AI Semantic Reactors**: set to the specific `ReactorID` (e.g. `backyard-flower-counter`).<br>- **System Messages**: stamped automatically with the `LocalNodeID` by the Enforcer.<br>- *If missing, defaults to `"bindu_auth"` or `"bindu-system"`.* |
| `$user`     | string    | Associated user for the event. Defaults to `"bindu-system"`.      |
| `$group`    | string    | Group ID for access control. Defaults to `"defaultGroup"`.        |
| `$when`     | date-time | Timestamp when the event occurred. Stamped automatically if missing. |
| `$where`    | string    | Hierarchical location path in Locagram/LocaGrama (e.g. `store1.zoneA.rack3`). |
| `fields`    | object    | Dynamic payload, must match the structure of the Message Type.    |

Other metadata and optional fields include:

| Field            | Type      | Description & Ingestion Behavior                                  |
| ---------------- | --------- | ----------------------------------------------------------------- |
| `$node-origin`   | string    | **Mesh Entry Point**: Stamped by the Enforcer with the ID of the node where the message first entered the mesh. Cannot be forged by plugins. |
| `$node-exec`     | string    | **Execution Stamping**: Stamped by the Enforcer with the ID of the host node currently executing the runtime that processed/ingested the message. If a plugin attempts to set this, the host runtime ignores it, logs a warning (`msg.node_exec.ignored`), and stamps the true local node ID. |
| `$src`           | string    | **Network Origin**: Used for routing control and connectivity diagnostics (e.g., `MessagePing` and `MessagePong`). Represents the node ID that initiated the request or is sending the reply. |
| `$arriveTime`    | date-time | Time the dispatcher received the message.                         |
| `$rebroadcast`   | boolean   | Internal flag to prevent re-broadcast loops (for system use only). |

---

## 🧩 The `fields` Property

The `fields` object is validated using the **message type schema** defined in `messageTypesConfiguration.json`. It contains the real business data:

Example:

```json
"fields": {
  "temperature": 22.5,
  "unit": "Celsius"
}
```

The allowed fields, types, and validation rules (like required, enum, etc.) come directly from the message type config.

---

## 🔍 Sample Message in Full

Here's a complete message:

```json
{
  "$type": "BatteryLevel",
  "$producer": "reader001",
  "$user": "employee_store1",
  "$group": "store1-employee",
  "$when": "2025-07-04T08:34:55.000Z",
  "$where": "store1.zoneA.rack3",
  "fields": {
    "level": 87
  }
}
```

This message:

- Has metadata to track where/when/who
- Uses type `BatteryLevel`
- Has a single field `level` defined in that type

---

## 🗺️ What is `$where`?

The `$where` field encodes **semantic location** in a hierarchical structure we call **LocaGrama**:

- Example: `store1.zoneA.rack3`
- It's **not physical coordinates**, but logical/symbolic placement
- Helps route, query, or visualize messages

---

## 🔐 How is it Validated?

Every message goes through two layers of validation:

1. **Generic Envelope**: checked against `genericMessageSchema.json`
2. **Payload Fields**: validated against the type schema selected by `$type`

If any part is invalid:

- The message is rejected
- An error message is published to an error topic

---

## 🧭 Summary

A valid Bindu message = **Envelope** + **Fields**:

- `$type` → selects the schema
- `fields` → validated against that schema
- `$where` → places the event in semantic space

This approach separates concerns:

- **Message Types** define content
- **Generic Schema** defines behavior

Everything is validated and traceable, enabling a clean, semantic IoT model. 🌐

