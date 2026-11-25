# Bindu Studio — User Guide (Command Overview)

`bindu-studio` is the operator CLI for product lifecycle, runtime rollout, mesh operations, plugins, and system configuration.

## Command Surface (Current)

Current top-level commands:

- `access-control` — access-control configuration tools.
- `cel-reactor` — inspect, export, import, merge, and validate CEL declarative transformation rules.
- `completion` — shell completion scripts.
- `config-state` — show whether config changes require restart.
- `dashboards` — export, import, list, and validate declarative dashboards.
- `doctor` — diagnostics commands.
- `geo-locagram` — manage global WGS84 coordinates (`NodeGeoLocation`), indoor Cartesian planes (`NodeLocalGeometry`), batch import/export, offline validation, and spatial event queries.
- `hall` — hierarchical command/flag help tree.
- `help` — help for any command.
- `license` — inspect/update installed license token.
- `lifecycle` — product import and activation.
- `mesh` — inspect/update mesh topology and node metadata.
- `message-samples` — export persisted samples from SQL and replay them through `bindu/in`.
- `message-types` — inspect, import, export, update, and hot-reload message types.
- `plugin` — create/certify/apply/remove/list plugins.
- `query` — execute read-only SQL queries and rehydrate persisted message state to target gateway nodes via `bindu/in`.
- `reset-bindu` — restart `bindu-root` using the current installation's restart mechanism.
- `rest-reactor` — inspect, export, import, merge, dry-run, and validate REST call reactor configurations.
- `root-service` — manage `bindu-root` service lifecycle.
- `runtime` — runtime desired snapshots and runtime update delivery.
- `schedule-event` — schedule, cancel, list, and purge distributed time-based events, timers, and digital twin awakening state machines.
- `secrets` (alias `integration-passwords`) — manage encrypted integration credentials, third-party API keys, and storage tokens in `accessControlConfig.brs`.
- `security` (alias `audit`, `scan`) — audit Go runtime and third-party dependencies for reachable vulnerabilities via static call-graph reachability analysis.
- `self-update` — update `bindu-studio` binary.
- `semantic-ai-reactor` — export, import, merge, dry-run, and inspect active semantic AI reactor deployments.
- `solution` — declarative MAS solution manifests and bundles (`MASSolutionSet`) for AI-driven deployment and cloning.
- `system-config` — inspect/validate system configuration.

Use this to print the live command tree from your current binary:

```bash
bindu-studio hall
bindu-studio hall --format json
```

## Intent by Domain

- Product install/update/activation:
  `lifecycle`
- Runtime config/plugins rollout to nodes:
  `runtime`
- Plugin development and assignment:
  `plugin`
- Declarative CEL, REST, and Semantic AI Reactors:
  `cel-reactor`, `rest-reactor`, `semantic-ai-reactor`
  *(Note: All reactor validation commands strictly enforce the Anti-Recursion Safety Invariant `E_REACTOR_RECURSIVE_LOOP` — output/response message types cannot match trigger/input message types).*
- Distributed Time-Based State Machine & Timers:
  `schedule-event` (`schedule`, `cancel`, `list`, `purge`)
- Geospatial & Indoor Cartesian Planes:
  `geo-locagram` (`list`, `show`, `set`, `delete`, `geocode`, `local-plane`, `local-set`, `local-delete`, `import`, `export`, `validate`, `query`)
- Observability and Visualizations:
  `dashboards`
- Query & Node State Rehydration:
  `query` (`bindu-studio query rehydrate`), `message-samples`
- Mesh topology, live diagnostics, node control, and emergency recovery:
  `mesh` (topology: `tree`, `add`, `bulk-add`, `edit`, `move`, `delete`, `remove`, `apply`; diagnostics: `health`, `ping`, `node-presence`, `node-status`, `node-version`; control: `node-refresh`, `node-restart`; recovery: `rescue-parent`, `rescue-all`)
- Configuration and policy:
  `system-config`, `access-control`, `license`
- Operational tooling & security:
  `doctor`, `security`, `config-state`, `root-service`, `reset-bindu`, `self-update`

## Guide Map

- Plugins: `bindu-studio-plugin-user-guide.md`
- CEL Reactors: `cel_reactor_user_guide.md`
- REST Reactors: `rest-reactor-user-manual.md`
- Semantic AI Reactors: `AISemanticReactorSet Schema & Authoring Guide for AI.md`
- Scheduled Events & Time Engine: `scheduled_events_user_guide.md`
- Geo-Locagram & Spatial Semantics: `bindu-studio-geo-locagram-user-guide.md`
- Dashboards: `dashboards_user_guide.md`
- SQL Queries & State Rehydration: `queries_user_guide.md`
- Mesh: `Bindu Studio Mesh User Manual.md`
- Node deploy/runtime rollout details: `Bindu Studio Mesh Node Deployment.md`
- Lifecycle (product import/activation): `lifecycle-user-manual.md`
- System config: `Bindu Studio - System Configuration User Guide.md`
- Access control: `Bindu Access Control User Manual.md`
- MAS Solutions & AI Cloner: `bindu_studio_solution_user_guide.md`

## Notes

- Hidden/internal commands are not part of operator workflows and are intentionally excluded from this guide.
- The authoritative source is always the running binary (`bindu-studio hall` / `bindu-studio <cmd> -h`).

## Scheduled Events & Timers Highlights

Notable `schedule-event` workflows for edge time orchestration:

```bash
# 1. Schedule a 30s delay timer on edge node
bindu-studio schedule-event schedule \
  --node node-dell-1 \
  --id silo-quarantine-batch-001 \
  --action recondition_wakeup \
  --producer lot-potato-4891 \
  --group coldchain \
  --delay-sec 30 \
  --payload '{"batch_id":"BATCH-4891","target_fryer_temp":178.5}'

# 2. Inspect active timers and awakening history
bindu-studio schedule-event list --node node-dell-1

# 3. Cancel a pending timer
bindu-studio schedule-event cancel --node node-dell-1 --id silo-quarantine-batch-001 --reason "Manual batch release"

# 4. Purge completed and expired timers from the ledger (leaves active future schedules)
bindu-studio schedule-event purge --older-than 24h
```

## Query & Rehydration Highlights

Notable `query` workflows for programmatic state rehydration:

```bash
# Rehydrate latest world geometry to a specific gateway node
bindu-studio query rehydrate \
  --sql 'SELECT * FROM v_messages WHERE "$type" = '\''NodeLocalGeometry'\'' ORDER BY "$when" DESC LIMIT 1' \
  --targets node-1 \
  --auth console:Pass123

# Rehydrate MAS solution set to multiple edge nodes with JSON output
bindu-studio query rehydrate \
  --sql 'SELECT * FROM v_messages WHERE "$type" = '\''MASSolutionSet'\''' \
  --targets node-1,dell-debian1 \
  --auth console:Pass123 \
  --json
```

## Runtime Highlights

Notable `runtime` workflows in the current CLI:

- `bindu-studio runtime baseline status|diff|accept`
- `bindu-studio runtime desired show|commit`
- `bindu-studio runtime update wizard`
- `bindu-studio runtime config-backup list`
- `bindu-studio runtime config-backup create --note "..."`
- `bindu-studio runtime config-restore <backup-id>`

`runtime config-backup` and `runtime config-restore` are local root operations over `runtime/config`; they do not require `--auth`.

## Mesh Operations & Recovery Highlights

Notable `mesh` workflows for operational diagnostics and disaster recovery:

```bash
# 1. Cluster-wide health snapshot
bindu-studio mesh health

# 2. Inspect node presence, advertised IP endpoints, and lane readiness
bindu-studio mesh node-presence dell-debian1
bindu-studio mesh presence node-1 --json

# 3. Pull deep runtime telemetry (CPU, RAM, disk, active BPR plugins, reactors)
bindu-studio mesh node-status dell-debian1

# 4. Measure end-to-end mTLS MQTT ping round-trip time
bindu-studio mesh ping node-1 --count 3 --auth console:StrongPass123

# 5. Trigger immediate state telemetry recalculation from an edge node
bindu-studio mesh node-refresh dell-debian1 --reason "Post-rollout audit" --auth console:StrongPass123

# 6. Graceful edge node service restart
bindu-studio mesh node-restart node-1 --reason "Applying kernel updates" --yes --auth console:StrongPass123

# 7. Targeted emergency ParentPresence rescue (after node IP drift)
bindu-studio mesh rescue-parent --node-id dell-debian1 --node-ip 10.0.5.52 --auth console:StrongPass123

# 8. Batch emergency ParentPresence rescue across multiple nodes (subnets drift)
bindu-studio mesh rescue-all --nodes "node-1=10.0.5.51,dell-debian1=10.0.5.52" --yes --auth console:StrongPass123
bindu-studio mesh rescue-all --inventory /etc/bindu/production-nodes.txt --yes --auth console:StrongPass123
```

## Security & Vulnerability Auditing Highlights

`bindu-studio security audit` performs static call-graph reachability analysis against the official Go Vulnerability Database (`govulncheck`), ensuring zero false positives by alerting strictly on vulnerabilities actively reachable in compiled code paths:

```bash
# 1. Standard reachable vulnerability audit
bindu-studio security audit

# 2. Audit with verbose breakdown (including uncalled transitive dependencies)
bindu-studio security audit --show-all

# 3. Machine-readable JSON output for automated CI/CD and DevOps ingestion
bindu-studio security audit --format json
```

