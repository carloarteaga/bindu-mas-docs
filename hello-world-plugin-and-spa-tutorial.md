# Hello World Plugin + SPA Tutorial

This tutorial shows one complete path:

1. generate a BPR plugin with `bindu-studio plugin init`
2. understand each generated file
3. adjust the schedules so:
   - `HelloWorld` emits every 2 minutes
   - `HelloWorldStats` emits every 5 minutes
4. deploy or run the plugin
5. use the standalone SPA in `public/console/docs/hello-world-tutorial/hello-world-spa`
6. see how the SPA combines:
   - MQTT over WSS
   - read-only SQL over REST
   - Geo-Locagram representations from `NodeGeoLocation`

## Auth Contract

Studio commands in this tutorial talk to `bindu-root` and require Basic auth.

Recommended for operator automation and AI-driven workflows:

```bash
export BINDU_STUDIO_AUTH=console:<password>
```

If you do not export it, add `--auth console:<password>` to the Studio commands below.

Important source note:

- `public/console/docs/hello-world-tutorial/hello-world` is the raw scaffold/tutorial version and still contains TODO-style starter logic.
- `public/console/docs/hello-world-tutorial/hello-world-plugin` is the curated working plugin source to use when you want to actually run, doctor, certify, and deploy the tutorial plugin.
- Both sources now follow the canonical Go SDK contract: import `bindunet/sdk` and resolve it through the surrounding Bindu module declared in `go.mod`.

## 1) Message Types Used by This Tutorial

This tutorial depends on:

- [hello-world-message-types.json](./hello-world-tutorial/hello-world-message-types.json)

The three message types are:

- `HelloWorldConfig`
- `HelloWorld`
- `HelloWorldStats`

Import them before generating or running the plugin:

```bash
bindu-studio message-types import \
  --in-path ./public/console/docs/hello-world-tutorial/hello-world-message-types.json
```

## 2) Generate the Plugin with Studio

Use the dual-cron template if you want to understand the scaffold generation path:

```bash
bindu-studio plugin init \
  --no-interactive \
  --profile ai-safe \
  --template hello-world-dual-cron \
  --id hello-world \
  --outdir ./hello-world
```

That scaffold gives you:

- one input: `HelloWorldConfig`
- two outputs: `HelloWorld`, `HelloWorldStats`
- a starter `main.go`
- a valid `plugin.json`
- README and sample payloads

If your goal is to exercise the tutorial with the corrected plugin source, use:

- [hello-world-plugin](./hello-world-tutorial/hello-world-plugin)

That curated source already contains the working trigger and event logic.

## 3) Generated Files and What Each One Means

The important files are:

- `plugin.json`
  - declarative contract
  - plugin identity
  - inputs / outputs
  - schedules

- `main.go`
  - actual runtime logic
  - `Handshake()` must stay aligned with `plugin.json`
  - `OnTrigger()` handles cron triggers
  - `OnEvent()` handles inbound `HelloWorldConfig`

- `README.md`
  - operator runbook for the plugin

- `go.mod`
  - isolated Go module for plugin build/certify

- `examples/in/*.json`
  - sample inbound envelopes

- `examples/out/*.json`
  - sample output envelopes

## 4) Scaffold vs Curated Tutorial Plugin

There are two useful sources in this tutorial:

- `hello-world`
  - raw Studio scaffold
  - good for understanding what `plugin init` generates
  - still contains starter/TODO-style logic
- `hello-world-plugin`
  - curated working source
  - good for `plugin doctor`, `plugin certify`, assignment, and deployment
  - this is the source you should use for an actual tutorial run

## 5) Curated `plugin.json` Used by the Working Tutorial Plugin

Use:

```json
{
  "id": "hello-world-plugin",
  "name": "Hello World Plugin",
  "version": "1.0.0",
  "runtime": "bpr",
  "in": [
    "HelloWorldConfig"
  ],
  "out": [
    "HelloWorld",
    "HelloWorldStats"
  ],
  "schedules": [
    {
      "cron": "0 */5 * * * *",
      "trigger": "cron"
    },
    {
      "cron": "0 */15 * * * *",
      "trigger": "cron.2"
    }
  ]
}
```

Meaning:

- `cron` = every 5 minutes
- `cron.2` = every 15 minutes

## 6) Why This Tutorial Still Shows the Generated Schedules

The dual-cron template currently generates:

- `HelloWorld` every 5 minutes
- `HelloWorldStats` every 15 minutes

For this tutorial and SPA demo we want a faster loop:

- `HelloWorld` every 2 minutes
- `HelloWorldStats` every 5 minutes

So after `plugin init`, edit `plugin.json`.

## 7) Example `plugin.json` for the Raw Scaffold Version

Use:

```json
{
  "id": "hello-world",
  "name": "Hello World",
  "version": "1.0.0",
  "runtime": "bpr",
  "in": [
    "HelloWorldConfig"
  ],
  "out": [
    "HelloWorld",
    "HelloWorldStats"
  ],
  "schedules": [
    {
      "cron": "0 */2 * * * *",
      "trigger": "cron"
    },
    {
      "cron": "0 */5 * * * *",
      "trigger": "cron.2"
    }
  ]
}
```

Meaning:

- `cron` = every 2 minutes
- `cron.2` = every 5 minutes

That 2m/5m version is useful as a teaching example, but the curated plugin source shipped with this tutorial is the `hello-world-plugin` directory shown above.

## 8) Final Trigger Logic for This Tutorial

The key rule is simple:

- `trigger=cron` emits only `HelloWorld`
- `trigger=cron.2` emits only `HelloWorldStats`

That keeps message intent clear and makes the SPA easier to explain.

In the curated tutorial plugin under:

- `/opt/bindu/root/my-plugins/hello-world-plugin`

the `OnTrigger()` logic was adjusted to:

```go
switch trigger.ID {
case "cron":
    return []sdk.Message{{
        Type: "HelloWorld",
        Payload: map[string]any{
            "greeting": p.currentGreeting(),
        },
    }}, nil
case "cron.2":
    stats, err := buildHelloWorldStats(ctx)
    if err != nil {
        return nil, err
    }
    return []sdk.Message{stats}, nil
default:
    ctx.Logger().Printf("ignoring unknown trigger id=%q", trigger.ID)
    return nil, nil
}
```

## 9) Runtime Behavior

This tutorial version behaves like this:

1. every 2 minutes:
   - emits `HelloWorld`

2. every 5 minutes:
   - emits `HelloWorldStats`

3. if `HelloWorldConfig` arrives:
   - the scaffold leaves `OnEvent()` ready for you to implement custom behavior

At this stage the tutorial plugin is intentionally simple:

- it demonstrates contract shape
- schedule separation
- WSS subscription target for the SPA
- read-only query integration pattern

## 10) Validate and Package the Plugin

Run:

```bash
bindu-studio plugin doctor --source ./hello-world-plugin --pre-release --strict
bindu-studio plugin certify --source ./hello-world-plugin --out ./hello-world-plugin.tar.gz --json
```

If you are using the live plugin at:

- `/opt/bindu/root/my-plugins/hello-world-plugin`

then run:

```bash
bindu-studio plugin doctor --source /opt/bindu/root/my-plugins/hello-world-plugin --pre-release --strict
bindu-studio plugin certify \
  --source /opt/bindu/root/my-plugins/hello-world-plugin \
  --out /opt/bindu/root/my-plugins/hello-world-plugin.tar.gz \
  --json
```

## 11) Deploy Pattern

Use the standard node-only plugin rollout if the plugin runs on nodes.

If the plugin is intended for root only:

```bash
bindu-studio plugin certify \
  --source /opt/bindu/root/my-plugins/hello-world-plugin \
  --out /opt/bindu/root/my-plugins/hello-world-plugin.tar.gz

bindu-studio plugin apply \
  --bundle /opt/bindu/root/my-plugins/hello-world-plugin.tar.gz \
  --target root \
  --auth console:SuperSecret1
```

If the plugin is intended for nodes, use the node rollout flow instead of `--target root`.

## 12) The SPA for This Tutorial

The tutorial SPA is here:

- [hello-world-spa/index.html](./hello-world-tutorial/hello-world-spa/index.html)

Queries used by the SPA:

- [latest_node_geo_locations.sql](./hello-world-tutorial/hello-world-spa/queries/latest_node_geo_locations.sql)
- [hello_world_counts_by_gateway.sql](./hello-world-tutorial/hello-world-spa/queries/hello_world_counts_by_gateway.sql)

## 13) What the SPA Demonstrates

The SPA demonstrates the supported public integration surface for browser reads plus the public WSS integration surface.

### 13.1 MQTT over WSS

It subscribes to:

- `HelloWorldStats/defaultGroup/+`

and appends every arriving `HelloWorldStats` JSON to the bottom list.

Important:

- the SPA does **not** use the admin-only `/api/v1/mqtt/credentials` helper used by the local Console
- it uses the public WSS integration endpoint exposed by root
- by default it derives:
  - `wss://<host from rootUrl>:8443`
- it uses the configured `credentials` value directly
- it does **not** call `/api/v1/mqtt/credentials`
- the REST base must be the public `rest_query` listener
- the WebSocket base must be the public `ws_integration` listener
- in the tutorial runtime example below, those ports are `8444` and `8443`
- those values come from `mesh.json` and are environment-specific, not fixed product constants

The page has these hardcoded variables near the top:

```js
const rootUrl = 'https://192.168.12.219:8444';
const credentials = 'console:SuperSecret1';
const wsIntegrationUrl = 'wss://192.168.12.219:8443';
const defaultWsPort = 8443;
```

If needed, set `wsIntegrationUrl` explicitly.

Direct integration shape:

- `rootUrl = https://<root-host>:<rest_query port>`
- `wsIntegrationUrl = wss://<root-host>:<ws_integration port>`

Temporary SSH-tunnel testing shape:

- `rootUrl = https://127.0.0.1:<forwarded rest_query port>`
- `wsIntegrationUrl = wss://127.0.0.1:<forwarded ws_integration port>`

### 13.2 Read-only SQL over REST

The SPA uses:

- `POST /api/v1/messages/sql`

for:

- latest `NodeGeoLocation` per node
- grouped counts of `HelloWorld` by gateway producer
- deriving the gateway locagram path from persisted message data

This is the only public REST integration endpoint used by the example.

It does **not** call:

- `/api/v1/mesh`
- `/api/v1/mqtt/credentials`
- `/api/v1/geo-locagram/locations`

### 13.3 Geo-Locagram Representations

The map is built from persisted `NodeGeoLocation` messages.

The SPA loads the latest `NodeGeoLocation` for each producer node via SQL and reads:

- `$where.locagram`
- `fields.geojson`

That means the page demonstrates how an external app can consume Geo-Locagram data without calling the specialized Geo-Locagram endpoint.

## 14) Map Overlay Logic

The overlay rule used in this tutorial is:

1. load latest geometries from `NodeGeoLocation`
2. use each `NodeGeoLocation` row as the spatial reference for its producer node
3. every time a `HelloWorldStats` arrives:
   - re-run the grouped `HelloWorld` count query
4. for each gateway count:
   - find the polygon that contains the gateway point
   - place the total inside that polygon or, if needed, inside the first polygon loaded from `NodeGeoLocation`

This keeps the tutorial simple while still showing a realistic cross-surface integration.

## 15) Why the SPA Re-queries `HelloWorld`

The SPA uses `HelloWorldStats` as a **live trigger**, not as the source of truth for the map overlay.

That is intentional.

It demonstrates a good integration pattern:

- use MQTT/WSS to know *when something changed*
- use read-only SQL to compute the latest stable view

That keeps the browser logic simple and auditable.

## 16) Suggested Demo Sequence

1. import `hello-world-message-types.json`
2. run `bindu-studio plugin init ... --template hello-world-dual-cron`
3. inspect generated files
4. edit `plugin.json` to 2-minute / 5-minute schedules
5. adjust `main.go` so trigger IDs emit separate message types
6. switch to the curated plugin source in `public/console/docs/hello-world-tutorial/hello-world-plugin`
7. run `plugin doctor`
8. certify and deploy
9. open:
   - `/console/docs/hello-world-tutorial/hello-world-spa/index.html`
10. set:
   - `rootUrl`
   - `credentials`
   - `wsIntegrationUrl`
   - using the `rest_query` / `ws_integration` ports from the active `mesh.json`
11. watch:
   - `HelloWorldStats` arrive live
   - grouped `HelloWorld` counts refresh
   - counts appear inside Geo-Locagram polygons

If you host the SPA outside Bindu, also copy:

- the SPA directory itself
- `public/console/vendor/leaflet`

and serve both under the same local web root so `/console/vendor/leaflet/...` resolves correctly.

## 17) Files to Inspect Together

If you want to study the whole tutorial end to end, inspect these files in order:

1. [hello-world-message-types.json](./hello-world-tutorial/hello-world-message-types.json)
2. [plugin-init-generated-code-tutor.md](./plugin-init-generated-code-tutor.md)
3. [hello-world-plugin/plugin.json](./hello-world-tutorial/hello-world-plugin/plugin.json)
4. [hello-world-plugin/main.go](./hello-world-tutorial/hello-world-plugin/main.go)
5. [hello-world-spa/index.html](./hello-world-tutorial/hello-world-spa/index.html)
6. [latest_node_geo_locations.sql](./hello-world-tutorial/hello-world-spa/queries/latest_node_geo_locations.sql)
7. [hello_world_counts_by_gateway.sql](./hello-world-tutorial/hello-world-spa/queries/hello_world_counts_by_gateway.sql)

## 18) Scope of This Tutorial

This tutorial is intentionally simple.

It teaches:

- plugin scaffold structure
- split schedules with trigger IDs
- WSS subscription from a standalone page
- read-only SQL integration using only `POST /api/v1/messages/sql`
- Geo-Locagram consumption from persisted messages

It does **not** try to hide the architecture.

That is the point of the example.
