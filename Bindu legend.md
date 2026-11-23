# Bindu — Console & Studio Navigation Legend (Reality-Aligned)

This legend answers a simple user question:

> **“Where do I go to do X?”**

Bindu follows one core rule:

> **Studio defines. Console observes.**

---

## 1) Bindu Console (UI · Observe · Interact)

**What the Console is**
- A **local UI** for interacting with a *running* Bindu system
- Focused on **visibility, interaction, and safe operations**
- Not the authority for configuration contracts

**What the Console is NOT**
- Not where topology is defined
- Not where mesh.json is edited
- Not where schemas are changed

---

### Console Menu (As Seen Today)

```

Bindu Console
├─ Home
│  └─ High-level system overview
│
├─ Message Types
│  ├─ Browse message type definitions
│  ├─ Inspect schemas & semantics
│  └─ Understand routing / retention meaning
│
├─ Publish Message
│  └─ Manually publish application messages for testing (combobox search)
│
├─ Queries
│  └─ Run read-only queries over application & system messages
│
├─ Dashboards
│  ├─ Select or create custom dashboards
│  ├─ Use dynamic layout and drag-and-drop tiles
│  └─ View multiple chart types (Bar, Line, Area, Pie, Polar, Radar, Scatter, Bubble)
│
├─ Mesh Config
│  ├─ View mesh topology (read-only)
│  ├─ See nodes and hierarchy
│  └─ Observe resolved channels and bindings
│
├─ Geo-Locagram
│  ├─ Operational live map
│  ├─ Search/filter nodes by slug and path
│  └─ Track node geometry and presence status
│
├─ Plugins
│  ├─ View installed plugins
│  ├─ See plugin status (running / stopped)
│  └─ Observe execution state
│
├─ Semantic Reactor
│  ├─ Cognitive orchestration hub
│  └─ Inspect AI agent plans and semantic rules
│
├─ Media Storage
│  └─ View local/S3 camera capture and storage metadata
│
├─ License
│  └─ View license status and limits
│
└─ Help
   └─ Documentation and guidance

```

**Key rules (Console)**
- ❌ No mesh editing
- ❌ No topology changes
- ❌ No inference or “blank means off”
- ✅ Shows **what is running right now**

If you can *click it* in the Console, it is **observational or interactive**, not contractual.

---

## 2) Bindu Studio (CLI · Define · Validate · Apply)

**What Studio is**
- The **authoritative tool** for defining contracts
- Works on **files**, not live runtime state
- Deterministic, schema-driven, explicit

**Mental model**
- Console = *window*
- Studio = *pen*

---

### Studio Command Space (What Exists Today)

```

Bindu Studio
├─ access-control
│  └─ Inspect and validate access config
│
├─ completion
│  └─ Shell autocompletion scripts
│
├─ config-state
│  └─ Show if config changes require restart
│
├─ doctor
│  └─ Diagnostics and health commands
│
├─ hall
│  └─ Hierarchical command/flag help tree
│
├─ help
│  └─ Print command help
│
├─ license
│  ├─ status
│  └─ update
│
├─ lifecycle
│  └─ Product import, update and activation
│
├─ mesh
│  ├─ tree (ASCII tree view of mesh hierarchy)
│  ├─ gateways (list gateway nodes)
│  ├─ channels (compare bindings)
│  ├─ platforms (OS/ARCH metadata summary)
│  └─ validate (validate mesh.json against schema)
│
├─ message-samples
│  └─ Export and replay SQL-stored message samples
│
├─ message-types
│  └─ Inspect defined message types (read-only)
│
├─ plugin
│  ├─ init
│  ├─ certify
│  ├─ import
│  ├─ activate
│  └─ lifecycle inspection
│
├─ reset-bindu
│  └─ Restart bindu-root service
│
├─ root-service
│  └─ Manage systemd root service lifecycle
│
├─ runtime
│  ├─ baseline (status / diff / accept)
│  ├─ desired (show / commit)
│  ├─ update (wizard)
│  └─ config-backup / config-restore
│
├─ self-update
│  └─ Update bindu-studio binary
│
└─ system-config
   ├─ get / set
   └─ validate / plan

```

**Key rules (Studio)**
- ✅ Operates on full documents
- ✅ Schema validation before write
- ✅ Atomic writes
- ❌ No partial edits
- ❌ No runtime guessing

---

## 3) How Mesh Fits (Very Important)

### Mesh today

- **Defined in Studio**
- **Observed in Console**
- Stored as **one file**: `config/mesh.json`

```

Define mesh      →  bindu-studio mesh …
Validate mesh    →  bindu-studio mesh validate
Apply mesh       →  bindu-studio mesh apply (when implemented)
Observe mesh     →  Console → Mesh Config

```

Console’s *Mesh Config* page is a **viewer**, not an editor.

---

## 4) One-Line Orientation Map

```

Design & contracts → Studio → validate → apply
Runtime & insight  → Console → observe → interact

```

If you want to:

- **Understand what’s running** → Console  
- **Change definitions** → Studio  
- **Commit a contract** → Studio (apply)  

---

## 5) Why This Legend Exists

This separation prevents:

- UI-driven configuration drift
- Hidden defaults
- “Blank means off” logic
- Accidental topology changes

It is not about limitation —  
it is about **trust**.

---

## Final Note (Human)

If at any moment you feel:
> “Why is this so strict?”

That’s the point.

Bindu is designed so:
- humans don’t guess
- UIs don’t lie
- AI doesn’t hallucinate

This legend is the map that keeps everyone oriented.
```

---
