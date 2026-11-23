# Bindu Studio — Geo-Locagram & Spatial Semantics User Guide

## Executive Overview

**Bindu Geo-Locagram** is the spatial semantic subsystem of the Bindu Event Network. It provides multi-tiered spatial awareness across two coupled coordinate spaces:

1. **Global Macro Space (WGS84)**: Standard geographic coordinates (`Point`, `Polygon`) anchored to the planet via the `NodeGeoLocation` system message type.
2. **Branch-Local Micro Space (Cartesian Planes)**: Metric, millimetric, or pixel local planes anchored to physical facilities (e.g. stores, warehouses, factory floors, charging stations) via the `NodeLocalGeometry` system message type.

The `bindu-studio geo-locagram` command suite exposes the complete operational surface of Geo-Locagrams to DevOps operators and autonomous AI agents, supporting spatial topology discovery, address geocoding, local plane calibration, declarative batch export/import, offline schema validation, and low-latency bounding-box querying.

---

## Architectural Principles & Message Contracts

### 1. Global Geography: `NodeGeoLocation`
When global coordinates are assigned to a mesh node, Bindu publishes a `NodeGeoLocation` system message to `bindu/in`:
- **Topic**: `bindu/in`
- **`$type`**: `NodeGeoLocation`
- **`$group`**: `mesh`
- **`$producer`**: Target `node_id`
- **`$where`**:
  ```json
  {
    "locagram": "/sites/station-1/pump-1",
    "geojson": {
      "type": "Point",
      "coordinates": [-73.985130, 40.748817]
    }
  }
  ```
- **`fields`**: Optional metadata such as `address`, `color`, `accuracy_meters`, `source` (`"manual"`, `"gps"`, `"geocode"`, `"sim"`).
- **Tombstone**: When deleted, a `NodeGeoLocation` is published with `$where.geojson: null` to purge the spatial index.

### 2. Local Facility Planes: `NodeLocalGeometry`
When local micro-coordinates are assigned, Bindu publishes a `NodeLocalGeometry` system message:
- **Topic**: `bindu/in`
- **`$type`**: `NodeLocalGeometry`
- **`$producer`**: Target `node_id`
- **`fields`**:
  ```json
  {
    "node_id": "pump-1",
    "plane_node_id": "station-1",
    "coordinate_space": "local",
    "local_role": "pos_terminal",
    "color": "#3388FF",
    "geometry": {
      "type": "Point",
      "coordinates": [12.5, 8.3]
    },
    "plane": {
      "units": "meters",
      "scale": 1.0,
      "rotation_deg": 45.0,
      "origin_x": 0.0,
      "origin_y": 0.0
    }
  }
  ```
- **Tombstone**: When deleted, `fields.geometry: null` is published.

---

## Command Reference

### 1. Spatial Topology Discovery (`list` & `show`)

#### List all mesh nodes with spatial status
```bash
bindu-studio geo-locagram list
```
*Output Table:*
```
NODE ID   LOCAGRAM             GEOMETRY   COORDINATES            ADDRESS                     LOCAL PLANE             COLOR
-------   --------             --------   -----------            -------                     -----------             -----
root      /root                Point      -73.985130,40.748817   Empire State Building, NY   -                       #ff0000
node-1    /root/store-1        Polygon    -                      350 5th Ave, NY             plane_root              #00ff00
node-2    /root/store-1/pos-1  Point      -                      -                           pos_terminal (node-1)   #3388ff
```

Output as machine-readable JSON for AI scripting:
```bash
bindu-studio geo-locagram list --json
```

#### Inspect deep spatial profile of a single node
```bash
bindu-studio geo-locagram show --node node-1
```
*Output:*
```
🗺️ Spatial Profile for Node: node-1
• Locagram Path: /root/store-1
• Global GeoLocation (WGS84):
  - Address: 350 5th Ave, NY
  - Color:   #00ff00
  - GeoJSON: {"type":"Point","coordinates":[-73.986000,40.749000]}
• Local Geometry:
  - Plane Root: node-1 (Self Plane Root)
  - Local Role: plane_root
  - Plane Def:  units=meters, scale=1.00, rotation=0.0 deg
  - Geometry:   {"coordinates":[[[0,0],[20,0],[20,10],[0,10],[0,0]]],"type":"Polygon"}
```

---

### 2. Global WGS84 Coordinates (`set`, `delete`, `geocode`)

#### Set node coordinates (Point)
```bash
bindu-studio geo-locagram set \
  --node node-1 \
  --lat 40.748817 \
  --lon -73.985130 \
  --address "Empire State Building, NY" \
  --color "#ff0000"
```

#### Set complex polygon boundary (Geofence / Facility footprint)
```bash
bindu-studio geo-locagram set \
  --node store-1 \
  --geojson '{"type":"Polygon","coordinates":[[[-73.98,40.74],[-73.98,40.75],[-73.97,40.75],[-73.97,40.74],[-73.98,40.74]]]}' \
  --address "Store 101 Boundary"
```

#### Remove/delete global coordinates (Publish Tombstone)
```bash
bindu-studio geo-locagram delete --node node-1
```

#### Geocode human-readable address
```bash
bindu-studio geo-locagram geocode "Gran Via 1, Madrid, Spain"
```
*Output:*
```
📍 Address Resolved: Gran Via 1, Madrid, Spain
• Best Match: Calle Gran Vía, 1, 28013 Madrid, Spain
• Latitude:   40.419612
• Longitude:  -3.698751
```

---

### 3. Indoor Cartesian Local Planes (`local-plane`, `local-set`, `local-delete`)

#### Configure a node as a Local Plane Root
```bash
bindu-studio geo-locagram local-plane \
  --node store-1 \
  --units meters \
  --scale 1.0 \
  --rotation 35.5 \
  --origin-x 0.0 \
  --origin-y 0.0
```

#### Assign local coordinates to a child node / asset
```bash
bindu-studio geo-locagram local-set \
  --node pump-1 \
  --plane store-1 \
  --role dispenser \
  --geometry '{"type":"Point","coordinates":[12.4, 4.8]}' \
  --color "#00ffcc"
```

#### Define an exclusion or detection polygon in local coordinates
```bash
bindu-studio geo-locagram local-set \
  --node zone-unloading \
  --plane store-1 \
  --role safety_zone \
  --geometry '{"type":"Polygon","coordinates":[[[10,0],[15,0],[15,5],[10,5],[10,0]]]}'
```

#### Delete local geometry (Publish Local Tombstone)
```bash
bindu-studio geo-locagram local-delete --node pump-1
```

---

### 4. Declarative CI/CD Batch Import & Export

#### Export mesh spatial state (Native Bindu bundle)
```bash
bindu-studio geo-locagram export --out ./mesh-geometries.json
```

#### Export as standard GeoJSON FeatureCollection (for QGIS / Mapbox / Leaflet / Kepler.gl)
```bash
bindu-studio geo-locagram export --format geojson --out ./mesh-features.geojson
```

#### Import and batch publish spatial configurations
```bash
# Supports both native Bindu JSON and standard GeoJSON FeatureCollections
bindu-studio geo-locagram import --file ./mesh-geometries.json
```

---

### 5. Offline Schema & Geometry Validation

Validate any spatial geometry file before deploying to production:
```bash
bindu-studio geo-locagram validate --file ./facility-boundary.json
```
*Checks performed:*
- Valid GeoJSON Point or Polygon structure.
- Coordinate validity (Latitude `[-90, 90]`, Longitude `[-180, 180]`).
- Closed Polygon linear rings (minimum 4 coordinates, first point matching last point).
- FeatureCollection property schemas and `node_id` references.

---

### 6. Spatial Query Engine

#### Query events within geographic bounding box
```bash
bindu-studio geo-locagram query within-bbox \
  --min-lat 40.70 --min-lon -74.00 \
  --max-lat 40.80 --max-lon -73.90 \
  --type NodeGeoLocation \
  --limit 50
```

#### Snapshot all spatial nodes in a branch hierarchy
```bash
bindu-studio geo-locagram query branch --root root --include-root
```

---

## AI Agent & Automation Protocol

Autonomous AI Agents orchestrating Multi-Agent Systems (MAS) can execute batch spatial provisioning with deterministic consistency:

```bash
#!/usr/bin/env bash
set -euo pipefail

export BINDU_STUDIO_AUTH="console:SuperSecret1"

# 1. Resolve geographic center
GEO_JSON=$(bindu-studio geo-locagram geocode "100 Main St, Austin, TX" --json)
LAT=$(echo "$GEO_JSON" | jq -r '.data.best.lat // .details.best.lat')
LON=$(echo "$GEO_JSON" | jq -r '.data.best.lng // .details.best.lng')

# 2. Establish Global Facility Anchor
bindu-studio geo-locagram set --node warehouse-1 --lat "$LAT" --lon "$LON" --address "Austin Hub"

# 3. Establish Metric Local Plane
bindu-studio geo-locagram local-plane --node warehouse-1 --units meters --scale 1.0

# 4. Position Indoor Edge Assets
bindu-studio geo-locagram local-set --node agv-1 --plane warehouse-1 --role robot --geometry '{"type":"Point","coordinates":[2.5,1.0]}'
bindu-studio geo-locagram local-set --node camera-dock-1 --plane warehouse-1 --role sensor --geometry '{"type":"Point","coordinates":[0.0,15.0]}'
```
