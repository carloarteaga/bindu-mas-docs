# Bindu Dashboards — Advanced User Manual

Welcome to the Dashboards module. This tool allows you to transform raw semantic messages into actionable real-time visualizations. By leveraging standard SQL over the Bindu message fabric, you can create high-fidelity monitors for any operational metric.

---

## 🏛️ Core Architecture

Dashboards in Bindu are **message-driven**. Every dashboard you create is saved as a `DashboardDefinition` message.
- **Persistence**: Definitions are stored in the same fabric they observe.
- **Portability**: Dashboards can be rendered by any node in the mesh (if permitted by access control).
- **Semantics**: Dashboards target the `v_messages` view, providing a unified SQL interface to distributed message logs.
- **Type icon**: The visible dashboard emoji comes from the `DashboardDefinition` message type metadata, not from each individual message sample.
- **Retention identity**: Each dashboard uses group `dashboard` and a unique `$producer` derived from its name, so retention keeps the latest definition per dashboard.
- **Lifecycle**: Definitions may include `fields.status`. `deleted` acts as a semantic tombstone while preserving append-only history.
- **Maintenance materialization**: During maintenance, if the latest `DashboardDefinition` for a given `type + group + producer` has `fields.status = "deleted"`, Bindu purges that whole scope from storage.

---

## 🛠️ The Administration Wizard

Creating a dashboard follows a structured 3-step synchronization process within a dual-panel editor.

### Step 1: SQL Data Acquisition
The heart of your dashboard is a SQL query. 
- **The Source**: Query from `v_messages`.
- **Filtering**: Use `WHERE "$type" = 'YourMessageType'` to target specific schemas.
- **Time Aggregation**: Use `strftime()` for time-series grouping.
- **Validation**: You must click **Validate & Next** to ensure your query results are valid for mapping.

### Step 2: Semantic Identifiers
Map technical database columns to human-readable labels.
- These labels appear in **Chart Legends**, **Axis Titles**, and **Table Headers**.
- These labels are persisted inside `fields.labels` of the `DashboardDefinition`.

### Step 3: Visualization Widgets
Select the visualization type that best represents your data.
- The widget title is persisted inside `fields.view.title`.
- **Series Selection (Y-Axis):** For Bar, Line, Area, and Radar charts, the configuration panel exposes a multi-select list for the Y-Axis / Series. You can select multiple numeric columns (by holding `Ctrl` or `Cmd` keys) to plot them simultaneously on the same chart. The engine automatically renders each series with a distinct color from a premium palette. Distribution charts (Pie, Doughnut, Polar Area) and Correlation charts (Scatter, Bubble) remain as single-select inputs.

#### 1. Stat (Big Number) 🔢
Used for primary KPIs. Displays a single large numeric value.
- *Best for*: "Active Nodes", "Critical Alarms", "Daily Total".

#### 2. Charts (Visual Trends & Distributions) 📊
Bindu supports a full suite of high-performance charts powered by Chart.js:
- **Trend Charts**:
  - **Line Chart**: Tracks continuous variables over time.
  - **Area Chart**: Displays variables over time as a smooth curve with translucent colored fill.
- **Comparison**:
  - **Bar Chart**: Compares discrete categories side-by-side or stacked.
- **Composition**:
  - **Pie Chart** & **Doughnut Chart**: Shows shares of a total. Cartesian gridlines are hidden automatically.
- **Distribution & Multivariate**:
  - **Polar Area Chart**: Compares categories with equal-angled segments and variable radii.
  - **Radar Chart**: Maps multiple variables on a spiderweb layout.
- **Correlation**:
  - **Scatter Chart**: Plots discrete coordinate points ($X$, $Y$).
  - **Bubble Chart**: Plots coordinate points adding a third dimension ($R$) to control marker size.

#### 3. Data Table 📋
Displays raw query results in a structured grid. Ideal for audit trails or log exploration.

#### 4. Operational Map (map) 🗺️
Transforms semantic geometries, coordinates, and geo-locations into an interactive Leaflet map.

> [!IMPORTANT]
> **Operational Map Schema Contract (`dashboardDefinitionSchema.json`)**:
> The dashboard schema validator enforces strict JSON Schema rules with `additionalProperties: false`:
> - **Visualization Type**: `"type": "map"` (legacy or non-standard identifiers such as `"spatial_ops"` are rejected).
> - **Chart Renderer**: `"chart_type": "map"`.
> - **Coordinate Mapping**:
>   - `"x"`: name of the SQL column containing the X coordinate (or Longitude / Easting).
>   - `"y"`: array containing the SQL column name for the Y coordinate (or Latitude / Northing), e.g. `["pos_y"]`.
> - **Strict Properties**: Extra fields such as `mesh_anchor` are **strictly forbidden** and will cause `bindu-studio dashboard validate` to fail with schema validation errors.
>
> **Example Valid View Configuration**:
> ```json
> "view": {
>   "type": "map",
>   "title": "Operation Room Real-Time 2D BLE Spatial Tracker",
>   "chart_type": "map",
>   "x": "pos_x",
>   "y": ["pos_y"]
> }
> ```

- **Mesh Context:** Reference Planes and Zones native to the Bindu topology automatically render in the background.
- **Data Polling:** Dynamic assets (e.g. `BLEBeaconSeen`, `ClassSeen` events) are overlaid on the map and automatically move whenever the dashboard Viewer refreshes (e.g., every 10 or 30 seconds).
- **Customization:** Extract discrete coordinate numbers or GeoJSON geometries from `raw_json` in your SQL query to power real-time asset tracking.

---

## 🖥️ Dashboard Viewer (Monitoring Mode)

The **View Dashboards** mode is designed for operational control rooms and continuous monitoring.

### 🚀 Launching the Monitor
1. Click the **🖥️ View Dashboards** button in the toolbar.
2. Select one or multiple dashboards from the list.
3. Click **Launch Monitor**.

### 🛠️ Interactive Features
- **Immersive Mode**: The browser enters native Fullscreen to provide a dedicated monitoring experience.
- **The Tiled Grid**: Dashboards are arranged in a 2-column responsive grid.
- **Drag & Drop**: Grab a tile by its header to move it to a different position in the grid.
- **Layout Toggles**: Use the `↕️` icon on any tile to alternate between **Standard Width** (half row) and **Full Width** (takes the entire row).
- **Persistence**: Your custom order and tile sizes are automatically saved to your browser's `localStorage` for future sessions.

### 🔄 Auto-Refresh Engine
A floating control bar allows you to adjust how frequently the data updates:
- **Intervals**: 10 seconds, 30 seconds (default), 1 minute, or 5 minutes.
- **Background Update**: All tiles refresh simultaneously based on the selected rate.

### 📭 Handling "No Data"
If a query returns zero results, the tile displays a clean **"No Data Available"** state with a 📭 placeholder, ensuring the monitor remains aesthetically organized even when data is sparse.

---

## 💡 Best Practices & SQL Patterns

### Recommended Time Aggregation (Hourly)
```sql
SELECT strftime('%Y-%m-%d %H:00', "$when") as time_bucket, count(*) as total
FROM v_messages
WHERE "$type" = 'BinduCVEvent'
GROUP BY time_bucket
ORDER BY time_bucket DESC
LIMIT 24;
```

### Spatial Hydration (Operational Map)
To power a `map` widget, extract discrete coordinates (`pos_x`, `pos_y`) and entity identifiers from message payloads:
```sql
SELECT 
  "$when" AS timestamp,
  json_extract(raw_json, '$.fields.beacon_id') AS entity_id,
  json_extract(raw_json, '$.fields.status') AS status,
  CAST(json_extract(raw_json, '$.fields.coordinates_x') AS REAL) AS pos_x,
  CAST(json_extract(raw_json, '$.fields.coordinates_y') AS REAL) AS pos_y,
  json_extract(raw_json, '$.fields.where_locagram') AS geom
FROM v_messages
WHERE "$type" IN ('BLEBeaconSeen', 'BLEBeaconLost')
ORDER BY "$when" DESC
LIMIT 20;
```

### Complex Multi-Stage Analytics (Common Table Expressions / CTEs)
Bindu's SQL validator natively supports multi-stage Common Table Expressions (`WITH ... AS (...)`) for advanced aggregation, rolling windows, and lagged metrics:

```sql
WITH daily_summary AS (
  SELECT 
    strftime('%Y-%m-%d', "$when") AS day_date,
    json_extract(raw_json, '$.fields.category') AS category,
    COUNT(*) AS event_count
  FROM v_messages
  WHERE "$type" = 'StorageRiskAssessment'
  GROUP BY day_date, category
)
SELECT day_date, category, event_count
FROM daily_summary
ORDER BY day_date DESC
LIMIT 30;
```

---

## 💻 CLI Management via `bindu-studio dashboard`

Dashboards can be declaratively managed, exported, imported, and validated across environments using the `bindu-studio` CLI (note the singular command `dashboard`):

* **List active dashboards**:
  ```bash
  bindu-studio dashboard list --auth console:password
  ```
* **Export all dashboard definitions to JSON**:
  ```bash
  bindu-studio dashboard export --out ./dashboards-backup.json --auth console:password
  ```
* **Import and deploy dashboards from JSON**:
  ```bash
  bindu-studio dashboard import --file ./my-dashboards.json --auth console:password --yes
  ```
* **Validate dashboard JSON definitions offline**:
  ```bash
  bindu-studio dashboard validate --file ./my-dashboards.json
  # In developer workspaces without global install, pass base directory:
  bindu-studio dashboard validate --base-dir . --file ./my-dashboards.json
  ```

---

## 📺 Standalone Kiosk Dashboard Viewer (Root & Edge Nodes)

In addition to the console administration wizard, Bindu provides a standalone, ultra-lightweight **Kiosk Dashboard Viewer** (`/viewer`). It is designed for wall-mounted TVs, shop floor iPads, picking stations, and executive displays on both Root and Edge Nodes.

### Key Capabilities:
* **Direct Station Filtering**: Display specific dashboards without user selection:
  ```
  https://<node-ip>:<port>/viewer?id=store-floor-inventory,store-misplaced-items
  ```
* **Borderless Kiosk Mode**: Hide the toolbar for clean wall displays:
  ```
  https://<node-ip>:<port>/viewer?id=floor-kpi&kiosk=true&refresh=5000
  ```
* **Dual-Mode (Edge & Cloud)**: Edge nodes query local in-store SQLite ledgers, while Root displays fleet summaries.

👉 For complete station configuration, authentication options, and auto-refresh parameters, see the [Standalone Kiosk Dashboard Viewer User Manual](viewer_user_manual.md).

---

### Tips
- **Performance**: Limit your queries to the necessary time range to keep the UI snappy.
- **Escaping**: Remember that core Bindu fields (like `$type` and `$when`) must be quoted in SQL: `"$type"`.
- **Exiting**: To exit Monitoring Mode, press `Esc` or click the **Exit Mode** button in the viewer toolbar.
- **Delete semantics**: A deleted dashboard disappears immediately from the dashboard list because the loader excludes `fields.status = "deleted"`. Maintenance later materializes the purge of that retained scope.

---
*Bindu Dashboards — Monitoring is seeing what matters.*
