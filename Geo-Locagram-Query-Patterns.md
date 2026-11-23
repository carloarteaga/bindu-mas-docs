# Geo-Locagram Query Patterns

## Purpose

This guide shows practical read-only SQL patterns for consuming Geo-Locagram data from:

- Console Queries
- REST `POST /api/v1/messages/sql`
- plugins and integration tooling

These queries are intended for:

- site bootstrap
- local-plane inspection
- CV plugin context loading
- operator diagnostics

## REST query contract

The read-only query API accepts:

- `SELECT ...`
- `WITH ... SELECT ...`

Constraints:

- exactly one statement
- no write/DDL tokens
- query must use `v_messages`

That means CTE queries are valid as long as they remain read-only.

## 1. Latest world geometry per node

```sql
WITH ranked AS (
  SELECT
    "$producer" AS node_id,
    "$when",
    "$id",
    "$where" AS where_json,
    json_extract(fields, '$.color') AS color,
    ROW_NUMBER() OVER (
      PARTITION BY "$producer"
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeGeoLocation'
)
SELECT
  node_id,
  json_extract(where_json, '$.geojson') AS geojson,
  color,
  "$when"
FROM ranked
WHERE rn = 1;
```

## 2. Which nodes are local planes

```sql
WITH ranked AS (
  SELECT
    json_extract(fields, '$.node_id') AS node_id,
    json_extract(fields, '$.plane_node_id') AS plane_node_id,
    json_extract(fields, '$.local_role') AS local_role,
    ROW_NUMBER() OVER (
      PARTITION BY json_extract(fields, '$.node_id')
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeLocalGeometry'
)
SELECT node_id
FROM ranked
WHERE rn = 1
  AND local_role = 'plane_root'
  AND node_id = plane_node_id;
```

## 3. Current local snapshot for one site

This is a very good site bootstrap query.

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
    json_extract(fields, '$.color') AS color,
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
  geometry_json,
  color
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1'
ORDER BY node_id;
```

## 4. Local plane readiness for CV

This query answers:

- is there a `plane_root`?
- are there `asset` geometries?
- are there `observer` geometries yet?
- are there `zone` geometries?

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

Example interpretation:

- `plane_root = 1`, `asset > 0`, `observer = 0`
  - site has local context but not yet camera arbitration context
- `plane_root = 1`, `asset > 0`, `observer > 0`
  - site is ready for first multi-camera spatial reasoning

## 5. Camera observers for arbitration

This is the query a CV plugin will use once cameras have been authored as `observer`.

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
  locagram,
  geometry_json
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1'
  AND local_role = 'observer'
  AND geometry_json IS NOT NULL
ORDER BY node_id;
```

If this returns `0 rows`, the most common reason is:

- the site plane exists
- but camera nodes have not yet been authored as `local_role = 'observer'`

## 6. Join world root with local context

This is a good conceptual query for integrations because it combines:

- world footprint
- local descendants

```sql
WITH latest_world AS (
  SELECT *
  FROM (
    SELECT
      "$producer" AS node_id,
      json_extract("$where", '$.geojson') AS world_geojson,
      ROW_NUMBER() OVER (
        PARTITION BY "$producer"
        ORDER BY datetime("$when") DESC, "$id" DESC
      ) AS rn
    FROM v_messages
    WHERE "$type" = 'NodeGeoLocation'
  )
  WHERE rn = 1
),
latest_local AS (
  SELECT *
  FROM (
    SELECT
      json_extract(fields, '$.node_id') AS node_id,
      json_extract(fields, '$.plane_node_id') AS plane_node_id,
      json_extract(fields, '$.local_role') AS local_role,
      json_extract(fields, '$.geometry') AS local_geometry,
      ROW_NUMBER() OVER (
        PARTITION BY json_extract(fields, '$.node_id')
        ORDER BY datetime("$when") DESC, "$id" DESC
      ) AS rn
    FROM v_messages
    WHERE "$type" = 'NodeLocalGeometry'
  )
  WHERE rn = 1
)
SELECT
  w.node_id AS world_node_id,
  w.world_geojson,
  l.node_id AS local_node_id,
  l.local_role,
  l.local_geometry
FROM latest_world w
LEFT JOIN latest_local l
  ON l.plane_node_id = w.node_id
WHERE w.node_id = 'gas-station-1';
```

## 7. Why this matters

These queries show that Geo-Locagram is not only a drawing tool.

It already provides a structured spatial context that can be consumed by:

- CV plugins
- operational dashboards
- integration apps

## 8. Rehydrate latest spatial state to a late-joining node

The Queries page can republish the results of a read-only query to selected destination nodes.

For rehydration, the query must return message-shaped rows:

- `"$type"`
- `"$group"`
- `"$producer"`
- `"$when"`
- `"$where"`
- `fields`

### Rehydrate latest `NodeGeoLocation` for one site

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

### Rehydrate latest `NodeLocalGeometry` snapshot for one plane

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

This lets an operator republish the current spatial state to one or more child nodes without adding any special-case routing logic for Geo-Locagram.
- diagnostics and QA workflows

The key progression is:

- `NodeGeoLocation`
  - world footprint
- `NodeLocalGeometry`
  - site-local operational geometry
- read-only query layer
  - contextual snapshots for runtime consumers
