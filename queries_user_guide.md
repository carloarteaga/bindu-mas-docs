# Queries User Guide

## Overview

The Query module provides read-only SQL over stored messages via:

- Console Query page (`/console/queries`)
- CLI command: `bindu-studio query rehydrate`
- REST endpoints: `POST /api/v1/messages/sql` and `POST /console/queries/rehydrate`

Target dataset is the `v_messages` view.

The Queries module supports:

- web console and CLI query-based state rehydration to selected gateway nodes
- retention-aware query generation from the selected message type

That means you can run a read-only query that returns message-shaped rows and then republish that persisted state to one or more destination gateway nodes using the normal `bindu/in` routing path.

## Safety Contract

1. Only read-only `SELECT` or `WITH ... SELECT ...` statements targeting `v_messages` are allowed.
2. Exactly one statement per request. Semicolons inside string literals or a single trailing semicolon are accepted, but multi-statement executions (e.g. `SELECT 1; DROP TABLE messages;`) are strictly rejected.
3. No write/DDL/transaction commands (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `ATTACH`, `PRAGMA`, etc.) outside of string literals.
4. Keywords inside string literals (e.g., `WHERE fields LIKE '%delete%'` or `WHERE "$type" = 'StatusUpdate'`) are parsed by the lexical scanner and permitted safely.

If you send multiple statements, Query returns:
`multiple statements are not allowed`.

## `v_messages` Columns (common)

The `v_messages` dataset is a virtual SQL view defined over the physical database `messages` table. For performance and clarity, the view projects only the most common envelope parameters as top-level columns:

*   `"$id"` (Message ID)
*   `"$type"` (Message Type)
*   `"$group"` (Mesh Group)
*   `"$producer"` (Generating Entity ID)
*   `"$user"` (User ID)
*   `"$when"` (Message Timestamp)
*   `"$arriveTime"` (Database Ingress Timestamp)
*   `"$where"` (Geo-Locagram JSON or string)
*   `fields` (Core JSON payload fields)
*   `raw_json` (A fully reconstructed raw JSON string of the complete message envelope)

> [!NOTE]
> Quote all columns starting with `$` in your SQL statements (e.g. `"$type"` or `"$when"`).

---

## 🛰️ Querying Advanced Envelope Keys (Routing & Exec Metadata)

Some system and routing envelope parameters—such as `"$node-origin"`, `"$node-dest"`, `"$node-exec"`, or `"$rebroadcast"`—are **not** projected as top-level SQL columns in the `v_messages` view. 

However, they are fully preserved inside the `raw_json` column. You can query or filter by these fields dynamically using SQLite's JSON extraction capabilities:

### A. Modern Arrow Operator (`->>`)
Use the `->>` operator to extract a text value directly from `raw_json`:
*   `raw_json ->> '$.$node-origin'`
*   `raw_json ->> '$.$node-dest'`
*   `raw_json ->> '$.$node-exec'`

### B. Compatibility Function (`json_extract`)
Use `json_extract(raw_json, '$.path')` if maximum compatibility across older SQLite binaries is required:
*   `json_extract(raw_json, '$.$node-origin')`

---

## Useful Query Examples for Routing Metadata

### 1. Filter Messages Destined to a Specific Node
This query filters only the `PatioMonitorMedia` telemetry events that were routed and destined to `"node-dell-1"`:

```sql
SELECT *
FROM v_messages
WHERE "$type" = 'PatioMonitorMedia'
  AND raw_json ->> '$.$node-dest' = 'node-dell-1'
  AND "$when" >= '2026-07-26T04:00:00.000Z'
  AND "$when" <= '2026-07-27T03:59:00.000Z'
ORDER BY "$arriveTime" ASC
LIMIT 100;
```

### 2. Filter Messages Originating from a Specific Node
This query filters only `CountFlowersByColor` results that originated on the gateway `"node-dell-1"`:

```sql
SELECT *
FROM v_messages
WHERE "$type" = 'CountFlowersByColor'
  AND raw_json ->> '$.$node-origin' = 'node-dell-1'
  AND "$when" >= '2026-07-26T04:00:00.000Z'
  AND "$when" <= '2026-07-27T03:59:00.000Z'
ORDER BY "$arriveTime" ASC
LIMIT 100;
```

### 3. Filter Messages Executed on a Specific Node
This query returns events that were processed by a BPR plugin engine executing on `"node-1"`:

```sql
SELECT *
FROM v_messages
WHERE raw_json ->> '$.$node-exec' = 'node-1'
ORDER BY "$arriveTime" DESC
LIMIT 50;
```

## Quick Start

```sql
SELECT *
FROM v_messages
WHERE "$type" = 'HelloWorld'
ORDER BY datetime("$when") DESC
LIMIT 100
```

## ♻ Apply Retention (Retention-Aware Query Generation)

The Queries page features an **`♻ Apply Retention`** button situated in the query builder toolbar. This feature automates the generation of SQL queries that are mathematically aligned with the retention policies of your registered message types.

### How to Use It:
1. Select a **Message Type** from the dropdown list. The status summary adjacent to the button will immediately display the active retention configuration (e.g., *7 days* or *100 messages*).
2. Optionally specify a **From (UTC)** and/or **To (UTC)** date-time range.
3. Click the **`♻ Apply Retention`** button.
4. The Monaco SQL Editor will be populated automatically with a query tailored to that specific type and its active retention policy, which you can run immediately.

---

### Dynamic Generation Behaviors:

#### 1. When retention mode is `count`
If the message type is configured to retain a maximum number of records per source (e.g. `count: 5`), clicking **`♻ Apply Retention`** generates a SQLite window function query to partition by group and producer, keeping only the latest `N` messages:

```sql
WITH ranked AS (
  SELECT
    "$type",
    "$group",
    "$producer",
    "$user",
    "$when",
    "$where",
    fields,
    ROW_NUMBER() OVER (
      PARTITION BY "$group", "$producer"
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'HelloWorld'
)
SELECT
  "$type",
  "$group",
  "$producer",
  "$user",
  "$when",
  "$where",
  fields
FROM ranked
WHERE rn <= 5;
```

#### 2. When retention mode is `days`
If the message type is configured to retain messages for a relative duration (e.g. `days: 7`), clicking the button automatically appends the date-based retention filter relative to the current time:

```sql
SELECT *
FROM v_messages
WHERE "$type" = 'HelloWorld'
  AND datetime("$when") >= datetime('now', '-7 days')
ORDER BY "$group" ASC, "$producer" ASC, datetime("$when") DESC;
```

#### 3. When retention mode is `none`
If the message type does not enforce a retention limit, the builder generates a standard structured query for the selected type, sorting results by group, producer, and descending timestamp:

```sql
SELECT *
FROM v_messages
WHERE "$type" = 'HelloWorld'
ORDER BY "$group" ASC, "$producer" ASC, datetime("$when") DESC;
```


## Query-Based Rehydration

After executing a successful query, the Queries page allows you to republish query results to edge nodes:

1. Look at the **Query Results** section at the bottom of the page.
2. Click the **`↻ Rehydrate to Nodes`** button in the Results header to display the target node selector.
3. Select one or more destination gateway nodes from the target list.
4. Click **`↻ Rehydrate Results`** to prepare and republish the resulting rows as semantic messages through the shared console connection to the selected nodes via `bindu/in`.

This is useful for:

- late-joining nodes
- plugin bootstrap
- restoring current reference state
- reasserting Geo-Locagram state like `NodeGeoLocation` and `NodeLocalGeometry`

### Rehydration contract

The query must return rows shaped like messages. In practice that means the result set should expose:

- `"$type"`
- `"$group"`
- `"$producer"`
- `"$when"`
- `"$where"`
- `fields`

The rehydration step then:

- keeps `"$type"`, `"$group"`, `"$producer"`, `"$where"`, and `fields`
- sets a fresh `"$when"` at publish time
- uses the current console user as `"$user"`
- sends one publish per `message x destination node`
- adds `"$node-dest"` for the selected node

Notes:

- root is intentionally excluded as a destination
- queries are capped to a small batch size for safety
- if your query returns computed summaries instead of message rows, rehydration will reject it

### Rehydrate latest world geometry to one node

```sql
WITH ranked AS (
  SELECT
    "$type",
    "$group",
    "$producer",
    "$user",
    "$when",
    "$where",
    fields,
    ROW_NUMBER() OVER (
      PARTITION BY "$producer"
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeGeoLocation'
)
SELECT
  "$type",
  "$group",
  "$producer",
  "$user",
  "$when",
  "$where",
  fields
FROM ranked
WHERE rn = 1
  AND "$producer" = 'gas-station-1';
```

### Rehydrate latest local plane state to one node

```sql
WITH ranked AS (
  SELECT
    "$type",
    "$group",
    "$producer",
    "$user",
    "$when",
    "$where",
    fields,
    json_extract(fields, '$.node_id') AS node_id,
    ROW_NUMBER() OVER (
      PARTITION BY json_extract(fields, '$.node_id')
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeLocalGeometry'
)
SELECT
  "$type",
  "$group",
  "$producer",
  "$user",
  "$when",
  "$where",
  fields
FROM ranked
WHERE rn = 1
  AND json_extract(fields, '$.plane_node_id') = 'gas-station-1';
```

### CLI Rehydration (`bindu-studio query rehydrate`)

For automated pipelines, background reconciliation scripts, or AI agent toolchains, `bindu-studio` exposes the `query rehydrate` command to rehydrate state without opening the Web Console UI.

#### Command Signature & Flags

```bash
bindu-studio query rehydrate [flags]
```

Flags:
- `--sql` *(required)*: The read-only SQL query selecting rows to rehydrate.
- `--targets` *(required)*: Comma-separated list of target gateway node IDs (e.g., `node-1,dell-debian1`).
- `--auth`: Basic auth credentials in `user:pass` format (or via `BINDU_STUDIO_AUTH` environment variable).
- `--root-url`: Base URL for `bindu-root` (default: from `binduConfig.json`).
- `--json`: Output execution summary in structured JSON format.

#### How It Works Under the Hood

1. **Validation & Preparation**: Calls `/console/queries/rehydrate` to validate read-only semantics and prepare clean message envelopes (stripping `$id`, `$arriveTime`, setting current user and UTC `$when`).
2. **MQTT Credentials**: Queries `/api/v1/mqtt/credentials` to retrieve ephemeral WebSocket broker credentials.
3. **Targeted Delivery**: For every target in `--targets` and every prepared message, sets `"$node-dest": target` and publishes to topic `bindu/in` with QoS 0.

#### CLI Examples

**1. Rehydrate single latest state to one gateway node:**

```bash
bindu-studio query rehydrate \
  --sql "SELECT * FROM v_messages WHERE \"$type\" = 'NodeLocalGeometry' ORDER BY \"$when\" DESC LIMIT 1" \
  --targets node-1 \
  --auth console:Pass123
```

**2. Rehydrate declarative MAS Solution Sets across multiple fleet nodes:**

```bash
bindu-studio query rehydrate \
  --sql "SELECT * FROM v_messages WHERE \"$type\" = 'MASSolutionSet'" \
  --targets node-1,dell-debian1 \
  --auth console:Pass123 \
  --json
```

**JSON Output Format:**

```json
{
  "success": true,
  "message_count": 1,
  "target_count": 2,
  "publish_count": 2,
  "targets": [
    "node-1",
    "dell-debian1"
  ],
  "topic": "bindu/in",
  "sql": "SELECT * FROM v_messages WHERE \"$type\" = 'MASSolutionSet'",
  "prepared_at_utc": "2026-09-26T15:00:00.123456789Z"
}
```

## Useful Patterns

### Count by type

```sql
SELECT "$type", COUNT(*) AS count
FROM v_messages
GROUP BY "$type"
ORDER BY count DESC
```

### Filter by time window

```sql
SELECT "$when", "$id", "$type"
FROM v_messages
WHERE datetime("$when") >= datetime('now','-30 minutes')
ORDER BY datetime("$when") DESC
LIMIT 200
```

### Extract payload fields (JSON)

```sql
SELECT
  "$when",
  json_extract(fields, '$.count') AS count,
  json_extract(fields, '$.since') AS since,
  json_extract(fields, '$.until') AS until
FROM v_messages
WHERE "$type" = 'HelloWorldStats'
ORDER BY datetime("$when") DESC
LIMIT 100
```

### Inspect errors

```sql
SELECT "$when", "$id", json_extract(fields,'$.reason') AS reason
FROM v_messages
WHERE "$type" = 'ErrorEvent'
ORDER BY datetime("$when") DESC
LIMIT 100
```

### Geo-Locagram local plane readiness

This is a good first example of a more advanced `WITH` query.

It helps answer:

- does this site already have a plane root?
- how many assets exist?
- are camera observers already authored?

```sql
WITH ranked AS (
  SELECT
    json_extract(fields, '$.node_id') AS node_id,
    json_extract(fields, '$.plane_node_id') AS plane_node_id,
    json_extract(fields, '$.local_role') AS local_role,
    CASE
      WHEN json_type(json_extract(fields, '$.geometry'), '$.geojson') = 'null' THEN NULL
      ELSE json_extract(fields, '$.geometry')
    END AS geometry_json,
    ROW_NUMBER() OVER (
      PARTITION BY json_extract(fields, '$.node_id')
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeLocalGeometry'
)
SELECT
  local_role,
  COUNT(*) AS count
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1'
  AND geometry_json IS NOT NULL
GROUP BY local_role
ORDER BY local_role;
```

### Geo-Locagram local snapshot for one site

This is useful for plugin bootstrap and operator inspection.

```sql
WITH ranked AS (
  SELECT
    json_extract(fields, '$.node_id') AS node_id,
    json_extract(fields, '$.plane_node_id') AS plane_node_id,
    json_extract(fields, '$.local_role') AS local_role,
    json_extract(fields, '$.locagram') AS locagram,
    CASE
      WHEN json_type(json_extract(fields, '$.geometry'), '$.geojson') = 'null' THEN NULL
      ELSE json_extract(fields, '$.geometry')
    END AS geometry_json,
    ROW_NUMBER() OVER (
      PARTITION BY json_extract(fields, '$.node_id')
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeLocalGeometry'
)
SELECT
  node_id,
  plane_node_id,
  local_role,
  locagram,
  geometry_json
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1'
ORDER BY node_id;
```

## REST Example

```bash
curl -X POST http://127.0.0.1:8080/api/v1/messages/sql \
  -u "console:SuperSecret1" \
  -H "Content-Type: application/json" \
  -d '{"sql":"SELECT COUNT(*) AS c FROM v_messages"}'
```

## Troubleshooting

- `multiple statements are not allowed`
  - send one read-only query only
  - remove trailing second statements (semicolons inside string literals `'...'` are permitted)
- `only SELECT statements are allowed`
  - start the query with `SELECT` or `WITH`
  - CTE queries are supported as long as they remain read-only and use `v_messages`
- `forbidden SQL token in query`
  - verify the query does not contain DDL, DML, or transaction keywords outside of string literals
  - keywords inside literal values must be wrapped in standard single quotes (e.g. `WHERE fields LIKE '%delete%'` or `WHERE status = 'update'`)
- `query must select from v_messages`
  - all queries must select from the virtual view `v_messages` rather than underlying physical tables
- `Row N is not rehydratable`
  - make sure the query returns message-shaped rows, not summaries
  - include the canonical message columns like `"$type"` and `fields`
- empty results
  - verify type name and time window
- bad JSON extraction
  - check `json_extract(fields, '$.path')` or `raw_json ->> '$.$node-dest'`

## Related Docs

- `publish_message_user_guide.md`
- `message_types_user_guide.md`
- `bindu-studio-plugin-user-guide.md`
- `Geo-Locagram-Query-Patterns.md`
