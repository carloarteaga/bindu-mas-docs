# Geo-Locagram User Guide

## 1. Purpose

Geo-Locagram is Bindu's spatial authoring surface.

It is used to:

- author world geometry for referential mesh nodes
- define a local plane rooted at a world polygon
- author local geometry for descendants inside that plane
- publish schema-valid semantic messages

Geo-Locagram is not:

- a spatial analytics tool
- a history browser
- a CV plugin
- a query engine by itself

It authors the context that other systems query and consume.

Related guides in this same docs directory:

- `queries_user_guide.md`
- `Geo-Locagram-Query-Patterns.md`

## 2. Core mental model

There are two editors inside the same page:

1. `World`
2. `Local plane`

`World` is for geographic geometry.

`Local plane` is for local X/Y geometry relative to one parent plane node.

The same mesh node can participate in both spaces:

- a gas station can have a `NodeGeoLocation` in world space
- that same gas station can also be the `plane_root` of a local plane
- children under that gas station can publish `NodeLocalGeometry`

The hierarchy always comes from mesh.
Geo-Locagram does not invent parent-child relationships.

## 3. Page layout

Geo-Locagram has three areas:

1. Toolbar
2. Nodes tree
3. Map/editor surface

### 3.1 Toolbar

The toolbar controls the current authoring mode.

Controls:

- `Space`
  - `World`
  - `Local plane`
- `Use active polygon as plane` or `Open local plane`
- `Zoom`
  - only active in local plane mode
  - changes editor display scale, not the stored coordinates
- `Units`
  - only active in local plane mode
  - sets the real-world unit family for the plane, such as `meters`
- `Scale`
  - only active in local plane mode
  - sets real units per one local coordinate unit
  - example: `units=meters` and `scale=0.5` means `1.0` local unit equals `0.5` meters
  - when you change `Units`, `Scale`, or `Zoom`, Geo-Locagram targets the plane root because that metadata belongs to the plane itself
  - while drawing in local mode, Geo-Locagram uses this value to show live segment or radius distance in the single top banner
- `Size`
  - only active in local plane mode
  - sets the exact width and height of the plane-root geometry in the current `units`
  - meant for the plane root itself, before authoring child areas or observers
- `Draw`
  - `None`
  - `Point`
  - `Polygon`
  - `Circle (UX)`
  - `Square (UX)`
- `Color`
  - applies to the active node
  - supported by both `NodeGeoLocation` and `NodeLocalGeometry`
- `Local Role`
  - only active in local plane mode
  - sets the spatial role stored in `NodeLocalGeometry`
  - use `plane_root` only for the active plane node itself
  - use `asset` for local zones/areas/polygons that can be covered or occupied
  - use `observer` for point-like observers such as cameras or sensors
- `Reset color`
- `Clear`
- `Edit`
- `Move`
- `Delete`
- `Finish`
- `Here`
  - only available in world mode
- `Go to address`
  - only available in world mode
  - resolves a street address and centers the world map on the best match
- `Save current geometry`

### 3.2 Nodes tree

Each row represents a referential node from mesh.

Each row has:

- filter box with autocomplete by slug/locagram
- expand/collapse arrow
- visibility checkbox
- node label and path
- optional `Local plane` badge
- optional count of inner nodes already authored in that plane

Meaning:

- filter box:
  - type any part of a node slug, path, locagram, name, or id
  - matching branches stay visible and auto-expand
  - non-matching branches are hidden from the tree view
  - browser autocomplete suggestions are provided from current mesh nodes
- row click: selects the active node
- checkbox in world mode: show/hide that node world geometry
- checkbox in local mode:
  - local geometries already inside the active plane are always shown
  - their checkbox is informational and disabled

### 3.3 Map/editor surface

In `World`:

- the editor uses geographic coordinates
- background is the real web map
- geometries are `Point` or `Polygon`
- `Go to address` is available in the editor bar after `Finish`

In `Local plane`:

- the editor uses local coordinates
- the plane root polygon is fixed
- a frozen reference snapshot from world is shown behind the local plane
- descendant nodes with local geometry are shown automatically

## 4. World mode

Use world mode when you want to define the geographic footprint of a referential node.

Typical flow:

1. select a node in the tree
2. optionally use `Go to address` to center the map on the site
2. choose `Polygon` or `Point`
3. draw on the map
4. optionally pick a color
5. save

What gets published:

- one `NodeGeoLocation` message
- producer = selected node id

### 4.1 Draw tools in world mode

- `Point` stores a GeoJSON `Point`
- `Polygon` stores a GeoJSON `Polygon`
- `Circle (UX)` is converted to a polygon before save
- `Square (UX)` is converted to a polygon before save

### 4.2 Editing in world mode

Right click a geometry to:

- edit
- move
- replace with another geometry type
- delete
- exit action mode

The same actions are also available in the toolbar:

- `Edit`
- `Move`
- `Delete`
- `Finish`

Geo-Locagram now uses one top banner for both:

- status and publish feedback
- current editing or drawing guidance

That banner can be hidden with its close button and will show again automatically when a new status or mode message is written.

## 5. Entering local plane mode

Local plane mode starts from a world polygon.

Requirements:

- the active node must already have a world `Polygon`

Flow:

1. select a node with a saved world polygon
2. click `Use active polygon as plane`

If the node already has local-plane data, the button changes to `Open local plane`.

What happens:

- the active node becomes the plane root
- the editor switches from geographic to local coordinates
- the world map is frozen into a reference background
- the local plane root is loaded or derived
- all saved local geometries for descendants of that plane are loaded
- the world polygon acts as the spatial seed for the plane
- if you want exact geometry dimensions, use `Set size` after the plane opens

## 6. Local plane mode

Use local plane mode to author local geometry inside one branch of mesh.

The active plane has exactly one root node.

That root node:

- belongs to mesh
- already has a world polygon
- also acts as `plane_root` in local space

Descendants in that branch can then receive local geometry.

### 6.1 What stays fixed

The plane root geometry is fixed in local mode.

That means:

- you do not redraw the plane root in local mode as a normal child geometry
- you set the exact geometry dimensions with `Set size`
- zoom changes only the display scale of the editor
- scale defines the real-world interpretation of local distances
- the stored local coordinates remain canonical

### 6.1.1 Distance contract

Geo-Locagram now distinguishes between two different ideas:

- `plane.display_scale`
  - editor-only zoom for authoring convenience
- `plane.scale`
  - real-world scale of the plane in `units` per one local coordinate unit

This matters for CV, observer arbitration, and any future reasoning that depends on real distance.

Examples:

- `units=meters` and `scale=1`
  - local coordinates are already authored directly in meters
- `units=meters` and `scale=0.25`
  - every local unit equals `0.25` meters
- `units=unitless` and `scale=1`
  - the plane has shape but no trustworthy real-world metric interpretation yet

### 6.1.2 Setting an exact geometry size

When the plane root comes from a world polygon, that polygon is only the spatial seed.

If you want the plane root geometry to become exact:

1. open `Local plane` on `room-1`
2. set `Units`, usually `meters`
3. enter `Size` as `width × height`
4. click `Set size`

This rescales the local `plane_root` polygon so the geometry has the exact dimensions you entered.

Recommended usage:

- do this before authoring `area-1`, `area-2`, observers, or other child local geometries
- after `Set size`, keep `scale=1` when you want local coordinates to directly mean meters

### 6.2 What is automatically shown

When a local plane opens, Geo-Locagram shows all nodes in that plane that already have saved local geometry, even if they were not checked in the tree before.

This is intentional.

In local mode, the important thing is the authored plane context, not the world visibility toggles.

### 6.3 Drawing in local mode

Select a descendant node and then choose a draw mode.

Supported:

- `Point`
- `Polygon`
- `Circle (UX)`
- `Square (UX)`

All are stored as local geometry.

`Circle (UX)` and `Square (UX)` are authoring helpers only.
Saved geometry remains standard point/polygon geometry in local coordinates.

While drawing polygons in local mode, the single top banner shows:

- `path`
  - the cumulative length of the sides already confirmed
- `edge`
  - the live distance from the last point to the cursor

### 6.4 Colors in local mode

`NodeLocalGeometry` supports `fields.color`.

You can assign color to:

- the plane root
- any local child geometry

Color affects UI only.
It does not affect routing, semantics, or coordinate transforms.

### 6.5 Leaving local mode

Use:

- `Back to world map`
or
- switch `Space` back to `World`

When you return to world:

- world geometry is shown again
- plane roots keep a visual distinction
- the tree continues to indicate which nodes already have local planes

## 7. Save behavior

Geo-Locagram does not publish continuously.

Edits remain local in the browser until you press `Save`.

Save publishes only the active node in the current editor space.

That means:

- `World` save emits `NodeGeoLocation`
- `Local plane` save emits `NodeLocalGeometry`
- delete is published as a semantic tombstone using `fields.status = "deleted"`
- Geo-Locagram still emits the legacy null-geometry tombstone shape for compatibility with existing read models
- during maintenance, if the latest retained message for a geometry scope is marked `deleted`, Bindu purges that whole `type + group + producer` scope from storage

## 8. Messages emitted by Geo-Locagram

## 8.1 NodeGeoLocation

Purpose:

- define the current world geometry of a node

Typical envelope:

```json
{
  "$type": "NodeGeoLocation",
  "$group": "mesh",
  "$producer": "gas-station-1",
  "$user": "console",
  "$when": "2026-03-26T22:00:00Z",
  "$where": {
    "geojson": {
      "type": "Polygon",
      "coordinates": [[[...]]]
    }
  },
  "fields": {
    "color": "#3388FF"
  }
}
```

Important fields:

- `$producer`
  - the node whose world geometry is being authored
- `$where.geojson`
  - geographic GeoJSON
- `fields.color`
  - optional UI color

## 8.2 NodeLocalGeometry

Purpose:

- define local geometry for a node inside one local plane

Typical envelope:

```json
{
  "$type": "NodeLocalGeometry",
  "$group": "mesh",
  "$producer": "gs1-fuel-pump-3",
  "$user": "console",
  "$when": "2026-03-26T22:10:00Z",
  "$where": {
    "locagram": "/headquarters/gas-station-1/fuel-pump-3"
  },
  "fields": {
    "node_id": "gs1-fuel-pump-3",
    "plane_node_id": "gas-station-1",
    "coordinate_space": "local",
    "local_role": "asset",
    "geometry": {
      "type": "Point",
      "coordinates": [14.2, 31.8]
    },
    "color": "#F97316"
  }
}
```

Plane root example:

```json
{
  "$type": "NodeLocalGeometry",
  "$group": "mesh",
  "$producer": "gas-station-1",
  "$user": "console",
  "$when": "2026-03-26T22:05:00Z",
  "$where": {
    "locagram": "/headquarters/gas-station-1"
  },
  "fields": {
    "node_id": "gas-station-1",
    "plane_node_id": "gas-station-1",
    "coordinate_space": "local",
    "local_role": "plane_root",
    "geometry": {
      "type": "Polygon",
      "coordinates": [[[...]]]
    },
    "plane": {
      "units": "meters",
      "scale": 1,
      "display_scale": 1.25
    },
    "color": "#3388FF"
  }
}
```

Important fields:

- `$producer`
  - for `NodeLocalGeometry`, Geo-Locagram uses the same locagram path stored in `$where.locagram`
  - this keeps retention keyed by branch-local spatial identity instead of by bare node id
  - Geo-Locagram should not publish `NodeLocalGeometry` if the active mesh node has no locagram path
- `fields.node_id`
  - node represented by the local geometry
- `fields.plane_node_id`
  - plane root that owns this local space
- `fields.coordinate_space`
  - currently `local`
- `fields.local_role`
  - for example `plane_root` or `asset`
- `fields.geometry`
  - local point or polygon
- `fields.plane`
  - only meaningful for the plane root
  - includes:
    - `units`
    - optional `scale` for real-world distance interpretation
    - optional `display_scale` for editor-only zoom
- `fields.color`
  - optional UI color

## 9. How messages are organized

The organization is:

- mesh defines hierarchy
- `NodeGeoLocation` defines world geometry
- `NodeLocalGeometry` defines local geometry inside one plane

One branch can have:

- one world polygon on the plane root
- one local plane rooted at that same mesh node
- many descendant local geometries under that plane

Association is explicit through:

- mesh hierarchy
- `fields.plane_node_id`
- `fields.node_id`
- `$where.locagram`

## 10. Query patterns

The queries below are the right starting point for CV and integration work.

They assume access to `v_messages`.

### 10.1 Latest world geometry per node

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

### 10.2 Which nodes are local planes

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

### 10.3 Latest local geometries for one plane

```sql
WITH ranked AS (
  SELECT
    json_extract(fields, '$.node_id') AS node_id,
    json_extract(fields, '$.plane_node_id') AS plane_node_id,
    json_extract(fields, '$.local_role') AS local_role,
    json_extract(fields, '$.geometry') AS geometry_json,
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
  local_role,
  geometry_json,
  color
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1'
ORDER BY node_id;
```

### 10.4 Latest plane root plus all children

```sql
WITH ranked AS (
  SELECT
    json_extract(fields, '$.node_id') AS node_id,
    json_extract(fields, '$.plane_node_id') AS plane_node_id,
    json_extract(fields, '$.local_role') AS local_role,
    json_extract(fields, '$.geometry') AS geometry_json,
    json_extract(fields, '$.plane') AS plane_json,
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
  geometry_json,
  plane_json
FROM ranked
WHERE rn = 1
  AND plane_node_id = 'gas-station-1';
```

### 10.5 Join world root with local children

This is the most useful conceptual query for CV bootstrap.

It gives:

- world footprint of the site
- local plane root
- local children authored inside it

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

### 10.6 CV bootstrap: local plane readiness

This query is a very practical first check for CV bootstrap.

It tells you whether one site already has:

- one `plane_root`
- operational `asset` geometry
- `observer` geometry for cameras
- `zone` geometry if authored

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

- `plane_root = 1` and `asset > 0`
  - the site is already useful for local spatial reasoning
- `observer = 0`
  - there are still no camera observers authored in that plane
- `zone = 0`
  - no explicit semantic zones have been authored yet

### 10.7 CV bootstrap: all current local geometry in one site

This query returns the current local snapshot for one plane.

It is a good bootstrap query for a site-centric CV plugin because it gives:

- the plane root
- all assets
- all future observers
- all future zones

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

If the current site contains only:

- `plane_root`
- `asset`

that means the local plane is already valid, but camera arbitration is not ready yet.

### 10.8 CV bootstrap: observers for arbitration

This query is the one a CV plugin will care about when it starts doing multi-camera arbitration.

It returns only `observer` geometry inside one site plane.

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

If this query returns `0 rows`, that does not mean the SQL is wrong.

It usually means:

- the site plane exists
- but no camera nodes have been authored yet as `local_role = 'observer'`

## 11. Why this matters for CV

Geo-Locagram is already enough to produce a first spatial context for a site-centric CV plugin.

For example, a plugin configured for `gas-station-1` can:

1. load the world footprint with `NodeGeoLocation`
2. load the local plane snapshot with `NodeLocalGeometry`
3. separate the results by `local_role`
4. keep `asset` geometry in memory for pump matching
5. later use `observer` geometry for camera arbitration

This means Bindu can evolve from:

- drawing geometry

to:

- building operational context for plugins
- associating detections with pumps or zones
- deciding when a site is ready for multi-camera reasoning

## 12. Recommended operator workflow

For a site like a gas station:

1. use the node filter to jump quickly to the target site by slug or locagram
2. author the site polygon in `World`
3. save `NodeGeoLocation`
4. open `Local plane`
5. verify the plane root and reference image
6. select descendant nodes one by one
7. draw local geometry
8. optionally color them
9. save `NodeLocalGeometry`
10. validate with SQL queries

## 13. What Geo-Locagram does not guarantee

Geo-Locagram does not by itself:

- infer business semantics from geometry
- define plugin logic
- perform CV calibration
- create analytics indexes
- replace mesh hierarchy

Those come later from plugins, policies, and read models built on top of these messages.
