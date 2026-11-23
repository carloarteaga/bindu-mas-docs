# How to Train an AI Agent to Use Bindu and Create Plugins

## Authentication Rule For AI Agents

Most `bindu-studio` admin commands require Basic auth against `bindu-root`.

Canonical rule for agents:

```bash
export BINDU_STUDIO_AUTH=console:<password>
```

If the environment variable is not set, the agent must include:

```bash
--auth console:<password>
```

Do not assume an implicit local session.

## Purpose

This guide defines a practical training profile for AI agents that operate Bindu and create BPR plugins safely.

It is the canonical AI-training entry point for plugin work.

## 1) Minimum System Model the AI Must Learn

The AI must internalize these facts first:

1. Bindu is message-centric.
2. Message Types are contracts (not just schema).
3. Plugins are external processes with BPR contract.
4. `bindu-studio` is the operator interface.
5. For plugin lifecycle, the UX is intention-first:
   - create/update code: `plugin certify` + `plugin apply --bundle`
   - assign existing code: `plugin apply --plugin`
   - remove/unassign: `plugin remove`

## 2) Command Intent Model (Must Be Memorized)

### Publish new plugin code

```bash
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz --targets-from-mesh
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
```

Target syntax rule:
- `--target` is `os/arch` (with `/`) and repeatable.
- `--targets-from-mesh` includes only platforms found in mesh.
- It does not imply all Linux targets.
- For Raspberry Pi 32-bit OS, use canonical `linux/armv7` (legacy `linux/arm` is accepted as compatibility alias).

### Assign existing plugin to node(s) without changing code

```bash
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
```

### Remove plugin from targets

```bash
bindu-studio plugin remove --plugin my-plugin --target all --uninstall --purge
```

### Rollout runtime update to already deployed nodes

```bash
bindu-studio runtime update wizard --nodes <node-a,node-b> --yes
```

## 2.1) Canonical End-to-End Deployment Sequence (Must Follow)

When training an AI for real deployments (plugins on root + nodes), enforce this exact order:

1. Install and start root.
2. Add nodes to mesh and apply (`mesh add`, `mesh apply`).
3. Create plugin (`plugin init`) and validate (`plugin doctor`).
4. Certify with node-aware targets:
   - `--targets-from-mesh`
   - plus host target if root platform differs from mesh nodes:
     - `--target $(go env GOOS)/$(go env GOARCH)`
5. Publish code to root (`plugin apply --bundle ... --target root`).
6. Assign plugin to nodes (`plugin apply --plugin ... --target node --node ...`).
7. Build first-install node bundles (`mesh node-deploy`).
8. Install/start nodes; plugins start automatically from assignment.

Canonical command block:

```bash
bindu-studio mesh add
bindu-studio mesh apply
bindu-studio plugin init --profile ai-safe --id my-plugin --outdir ./my-plugin
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH)
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin apply --plugin my-plugin --target node --node <node-id>
bindu-studio mesh node-deploy --nodes <node-id> --out ./dist/nodes
```

Rule for already deployed nodes:
- Do not reinstall.
- Use runtime rollout:
```bash
bindu-studio runtime update wizard --nodes <node-id> --yes
```

## 2.2) Re-Certify + Re-Apply Same Plugin ID (Transparent Update)

Use this when plugin is already active/assigned and code changed.

```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
bindu-studio plugin certify --source ./my-plugin --out ./my-plugin.tar.gz \
  --targets-from-mesh \
  --target $(go env GOOS)/$(go env GOARCH) \
  --json
bindu-studio plugin apply --bundle ./my-plugin.tar.gz --target root
bindu-studio plugin who-uses --plugin my-plugin
```

Expected behavior:
- plugin code is replaced for same `id`
- existing root/node assignments stay
- old non-active staged candidates are auto-cleaned
- `remove` / `undeploy` are not needed for normal updates

If later `mesh node-deploy` reports missing `linux/amd64`, re-run certify exactly with `--targets-from-mesh` (and host target), then apply again.

## 3) AI Plugin Authoring Workflow

Train the AI to always run this sequence:

1. Understand required Message Types (`in` / `out`).
2. Scaffold plugin:

```bash
bindu-studio plugin init --profile ai-safe --id my-plugin --outdir ./my-plugin
```

3. Implement logic in `main.go`.
4. Validate:

```bash
bindu-studio plugin doctor --source ./my-plugin --pre-release --strict
```

5. Certify and deploy (`--bundle` flow).
6. Verify runtime:

```bash
bindu-studio plugin list
bindu-studio plugin who-uses --plugin my-plugin
```

## 4) Guardrails for AI Agents

The AI must never:

- write directly to SQLite files
- depend on hardcoded DB paths
- bypass `plugin certify` for production deployment
- confuse assignment (`--plugin`) with code rollout (`--bundle`)

The AI must always:

- use read-only queries for analytics/debugging
- validate Message Type compatibility before deploy
- keep plugin changes idempotent and observable

## 5) Query Training Rules (for Plugins and Operators)

1. Query surface is read-only (`SELECT` only).
2. One statement per request.
3. Use `v_messages` and quote `$` columns.
4. Prefer bounded windows and `LIMIT`.

Example:

```sql
SELECT "$when", "$type", json_extract(fields, '$.count') AS count
FROM v_messages
WHERE "$type" = 'HelloWorldStats'
ORDER BY datetime("$when") DESC
LIMIT 100
```

## 6) Verification Checklist the AI Should Execute

Before proposing "done", the AI should verify:

1. `plugin doctor` passes.
2. `plugin certify` succeeds.
3. `plugin apply` output shows activated/assigned as expected.
4. `plugin list` and `plugin who-uses` match target intent.
5. For node rollout, runtime update path is executed.

## 7) Recommended Curriculum Order

1. `What is Bindu.md`
2. `message_types_user_guide.md`
3. `queries_user_guide.md`
4. `bindu-studio-plugin-user-guide.md`
5. `plugin-init-generated-code-tutor.md`
6. `bindu-bpr-go-sdk-developer-manual.md`
7. `Bindu Plugins — BPR Runtime Spec (AI-Assisted, Deterministic).md`

## 8) Outcome

If trained with this model, an AI agent can:

- design valid Message Types
- scaffold and implement plugins
- deploy/update/remove plugins with the correct intent
- operate root/node rollout safely without low-level lifecycle confusion
