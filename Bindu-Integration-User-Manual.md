# Bindu Integration User Manual

Role of this document:

- Canonical public integration guide
- Best first stop for deciding between MQTT, REST Query, and BPR

## Purpose

This document describes **all supported ways to integrate with Bindu**, the technologies involved, and **when to use each one**.

It is intentionally **technology-agnostic and business-oriented**:
- No internal implementation details
- No plugin internals (covered in the Plugin User Manual)
- Focus on **integration contracts**, **security**, and **recommended usage patterns**

---

## Bindu Integration Model (Overview)

Bindu exposes **three integration layers**, each designed for a specific class of consumers:

1. **MQTT Integration Layer**  
2. **Read-Only REST Query API**  
3. **Plugin Runtime (BPR)**

Each layer serves a different purpose.  
They are **complementary**, not competing.

---

## 1. MQTT Integration Layer

### What it is

The MQTT Integration Layer is the **primary real-time integration interface** of Bindu.

It is used to:
- Publish domain messages
- Subscribe to streams of events
- Drive automation, dashboards, digital twins, and AI agents

### Transports & Channels

Bindu exposes application and mesh integration over four structured channels:

| Channel | Protocol | Default Port | Security / TLS | Intended Clients |
|:---|:---|:---|:---|:---|
| **Mesh (Inter-Node)** | MQTT over TCP | `8884` | **mTLS** (Certificates + AEAD) | Root and Node mesh routing only |
| **MQTT Integration** | MQTT over TCP | `8883` | **TLS (MQTTS)** + RBAC | Native backend services, sensors, IoT devices |
| **WS Integration** | MQTT over WS | `8084` | **TLS (WSS)** + RBAC | External browsers, external SPAs, web dashboards |
| **WS Console** | MQTT over WS | `8083` | **Loopback (127.0.0.1)** + Session UUID | Local Web Console UI only (SSH tunnel for remote) |

> **External integration transports run strictly over TLS with AEAD ciphers.**
> There is **no unencrypted external MQTT**.

---

### Security & Provenance
- All external MQTT integration runs over **TLS** with AEAD cipher suites.
- Authentication and authorization use **Access Control RBAC** (`admin` or `application` role).
- Ingress metadata ($user) is cryptographically stamped by the Dispatcher to prevent sender spoofing.
- Topic ACLs restrict clients to authorized domains and prevent direct mutation of internal queues (`bindu/*`).

---

### When to use MQTT

Use MQTT when you need:
- Real-time communication
- Streaming updates
- Event-driven systems
- Bidirectional integration

**Typical use cases**
- IoT telemetry
- Digital twin synchronization
- Real-time monitoring
- AI agents reacting to live events
- Web dashboards (via WSS)

---

## 2. Read-Only REST Query API

### What it is

The Read-Only REST Query API provides **historical and state-based access** to data stored in Bindu.

It is:
- **Read-only by contract**
- Designed for **queries**, not control
- Part of the **integration layer**, not the admin plane
- Intentionally minimal in public surface

### Transport & Deployment Context

- **HTTPS only** (TLS with AEAD ciphers; no plaintext HTTP mode).
- Exposed on the active `rest_query` listener defined in `mesh.json`.
- Available on **both Root and Edge Gateway Nodes**:
  - **On Root Gateway (`bindu-root`)**: Queries the consolidated cluster/mesh ledger (`v_messages`). Ideal for enterprise BI, centralized reporting, cross-gateway auditing, and central AI agents.
  - **On Edge Gateway Nodes (`bindu-node`)**: When `rest_query` is configured in `mesh.json`, the edge node launches its own secure TLS query listener on startup. Local edge applications (SCADA systems, shop-floor PLCs, tablets, POS terminals, and local Python/Go/Node microservices) can query the node's local SQLite ledger (`v_messages`) with **sub-millisecond latency and zero dependence on WAN connectivity**.
  - **Standalone Kiosk Viewer (`/viewer`)**: Both Root and Edge nodes with `rest_query` serve the embedded kiosk web application (`/viewer`), allowing wall-mounted displays and tablets to show real-time auto-refreshing operational dashboards powered by this same REST query endpoint.

### Public Contract

The public browser/backend integration contract is:

- `POST /api/v1/messages/sql`

That endpoint is the single public REST entrypoint for integration queries.

It is used for:
- historical queries
- aggregates and KPI totals
- real-time ledger inspection and auditing
- external SPA and local application integration

For browser-based third-party integrations, this endpoint must be exposed with an explicit CORS policy. No other REST integration endpoint should be made public for that purpose.

The public HTTPS base URL for that endpoint is resolved from the active node's `rest_query` listener:
- Root Query: `https://<root-ip>:<root_rest_query_port>/api/v1/messages/sql`
- Edge Node Query: `https://<node-ip>:<node_rest_query_port>/api/v1/messages/sql`

Those listener ports are defined by the active `mesh.json`, so examples such as `8444`, `38450`, or `18444` are runtime configurations, not fixed product constants.

The public WSS base URL for browser real-time integration is the active root `ws_integration` listener.

Note:
- the local Bindu Console does not use the public `ws_integration` listener
- Console live traffic uses the local `control_plane.console_ws` listener on loopback

#### Request Format & Authentication
- **Authentication**: HTTP Basic Auth (`Authorization: Basic <base64>`) or session cookie. Integrators only need a user with the **`application`** role (no `admin` privileges required).
- **HTTP Method**: `POST`
- **Headers**:
  - `Content-Type: application/json`
  - `Authorization: Basic YXBwOlN0cm9uZ1Bhc3M=` *(example)*
- **Body**:
  ```json
  {
    "sql": "SELECT \"$id\", \"$type\", \"$producer\", \"$when\", fields FROM v_messages WHERE \"$type\" = 'FreezerTelemetry' ORDER BY \"$when\" DESC LIMIT 10;"
  }
  ```

#### Response Format
```json
{
  "status": "ok",
  "details": {
    "columns": ["$id", "$type", "$producer", "$when", "fields"],
    "rows": [
      ["c4b8-...", "FreezerTelemetry", "freezer-01", "2026-11-14T10:00:00Z", "{\"temp_c\": -21.4}"]
    ]
  }
}
```

#### Strict Read-Only Enforcement
The endpoint strictly invokes `sqlstorage.ValidateReadOnlyQuery()`. Any statement containing keywords such as `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, or `ATTACH` is immediately rejected with HTTP `400 Bad Request`.

The canonical browser example in Console docs is:

- [hello-world-plugin-and-spa-tutorial.md](./hello-world-plugin-and-spa-tutorial.md)

It is **not** accompanied by public integration endpoints for:
- `/api/v1/mesh`
- `/api/v1/mqtt/credentials`
- any control-plane or admin route

Administrative tooling may still use non-public endpoints behind Studio auth. For example, `bindu-studio message-samples import` replays captured samples through `bindu/in` by using Studio credentials plus `/api/v1/mqtt/credentials`. That is an operator workflow, not part of the public browser/backend integration contract.

---

### Who uses it

- External integrators (ERP, SCADA, third-party backend microservices)
- Dashboards and reporting tools
- Local shop-floor tablets and wall-mounted Kiosks (`/viewer`)
- AI agents performing analysis and memory retrieval
- **Bindu Console itself**

> The Console and Kiosk Viewer consume the same public Query API as any external integrator.

---

### What it is NOT

The Read-Only Query API:
- Does **not** mutate state
- Does **not** ingest telemetry (all external ingestion must enter through MQTT `bindu/in`)
- Does **not** trigger maintenance
- Does **not** manage configuration
- Does **not** replace MQTT for real-time flows
- Does **not** expose mesh topology or MQTT credential discovery as public browser endpoints

---

### When to use REST Query

Use REST Query when you need:
- Historical data
- Aggregated views
- Search/filter queries
- Deterministic reads

**Typical use cases**
- Reporting
- Auditing
- Debugging
- AI analysis over stored data
- Dashboards with pagination and filters

---

## 3. Plugin Runtime (BPR)

### What it is

The Bindu Plugin Runtime (BPR) allows **server-side extensions** to run *inside* the Bindu execution environment.

Plugins:
- Subscribe to messages
- Execute logic
- Publish new messages
- Integrate external systems

Plugins are **not remote integrations** — they are **co-located runtime components**.

---

### When to use plugins

Use plugins when you need:
- Low-latency internal processing
- Access to internal routing
- Deterministic execution close to the data
- Complex automation pipelines

**Typical use cases**
- Protocol bridges
- Stateful processors
- Internal AI agents
- Control logic
- Edge processing

> Detailed plugin development and lifecycle is documented separately in the Plugin User Manual.

---

## Integration Layer Comparison

| Capability | MQTT | REST Query | Plugin |
|----------|------|------------|--------|
| Real-time | ✅ | ❌ | ✅ |
| Historical queries | ❌ | ✅ | ⚠️ |
| Write / publish | ✅ | ❌ | ✅ |
| Read-only | ❌ | ✅ | ❌ |
| Runs remotely | ✅ | ✅ | ❌ |
| Runs inside Bindu | ❌ | ❌ | ✅ |

---

## Recommended Integration Patterns

### Web Applications
- **WSS MQTT** for live updates
- **REST Query** via `POST /api/v1/messages/sql` for history and pagination

### Backend Services
- **MQTTS** for event-driven flows
- **REST Query** for audits and reports

### AI Agents
- **MQTT** for real-time perception
- **REST Query** for memory, analysis, and planning

### Edge Nodes & Local On-Premises Facilities (Plants / Stores / Vessels)
- **MQTTS Ingress (`bindu/in`)**: Sensors, readers (RFID, BLE, cameras, barcode), and PLCs stream raw telemetry and state events into the local edge node.
- **REST Query Egress (`POST /api/v1/messages/sql`)**: Local edge software (SCADA, shop-floor microservices, picking stations, local POS, and diagnostic tools) queries recent state and aggregates directly from the edge node with zero WAN dependency.
- **Standalone Kiosk Displays (`/viewer`)**: Dedicated wall-mounted screens, iPads, or floor tablets display auto-refreshing operational dashboards connected directly to the local node's query API.

---

## Security & Exposure Principles (LOCKED)

- Console is **local-only**
- Admin/control endpoints are **never public**
- Public REST surface is **read-only only**
- Public REST integration is limited to `POST /api/v1/messages/sql`
- Integration protocols are **TLS-only**
- If something is mandatory, it is **not configurable**

---

## Final Principle

> **Bindu exposes integrations by intent, not by technology.**  
> Choose the interface that matches your use case:
> - MQTT for flows
> - REST for queries
> - Plugins for execution
