# Bindu Docs Navigation by Profile

Role of this document:

- Canonical navigation index for the distributed documentation set
- Best first stop when choosing a reading path by role or topic

Use this page as the entry point for the distributed docs library.

## Canonical Paths By Topic

- Overview:
  `Bindu Architecture Overview.md`
  Supporting docs: `What is Bindu.md`, `Bindu in 3 Layers.md`
- Routing:
  `routing_and_directionality_guide.md`
  Supporting docs: `Bindu Routing & Mesh Delivery — User Guid.md`, `Bindu Message Lifecycle — User Manual.md`
- Mesh:
  `Bindu Studio Mesh User Manual.md`
  Supporting docs: `mesh_user_guide.md`, `Bindu Studio Mesh Node Deployment.md`
- Integration:
  `Bindu-Integration-User-Manual.md`

## 1) Operator (day-to-day operations)

Recommended order:

1. `Bindu Studio — User Guide (Command Overview).md`
2. `how-to.md`
3. `bindu_reset_and_reload_lifecycle_guide.md`
4. `lifecycle-user-manual.md`
5. `Bindu Studio Mesh User Manual.md`
6. `Bindu Studio Mesh Node Deployment.md`
7. `Bindu Node Installation Guide.md`
8. `bindu-studio-plugin-user-guide.md`
9. `message_types_user_guide.md`
10. `publish_message_user_guide.md`
11. `queries_user_guide.md`
12. `Bindu Runtime Logging Model.md`

## 2) Plugin Developer

Recommended order:

1. `bindu-studio-plugin-user-guide.md`
2. `plugin-init-generated-code-tutor.md`
3. `plugin_execution_model_user_guide.md`
4. `bindu-bpr-go-sdk-developer-manual.md`
5. `Bindu Plugins — BPR Runtime Spec (AI-Assisted, Deterministic).md`

## 3) Autonomous AI Agent Profile (MAS Solution Creator)

AI Agents acting as Multi-Agent System (MAS) Solution Creators must treat Bindu as an immutable black-box platform. Follow the canonical **Whitelisted 6-Tier Curriculum** in `AGENTS.md` / `GEMINI.md`:

1. Master Specification: `AGENTS.md` / `GEMINI.md` (Audience Declaration & 6-Tier Curriculum)
2. Methodology & Deployment: `mas-deployment-guide.md`, `how-to-build-a-bindu-solution.md`
3. Ontology & Routing: `message_types_user_guide.md`, `Message Types for AI Agents.md`, `routing_and_directionality_guide.md`, `retention_policies_guide.md`
4. Hardware Plugins: `bindu-bpr-go-sdk-developer-manual.md`, `bindu-studio-plugin-user-guide.md`
5. Logic & Dashboards: `cel_reactor_user_guide.md`, `semantic_reactor_user_guide.md`, `rest-reactor-user-manual.md`, `dashboards_user_guide.md`
6. Solution Bundling: `bindu_studio_solution_user_guide.md`, `queries_user_guide.md`

## 4) Reference / Deep Dive

Use these for detailed semantics and architecture:

- `mesh_user_guide.md`
- `Bindu Routing & Mesh Delivery — User Guid.md`
- `Bindu Routing Log Triage.md`
- `message_user_guide.md`
- `Bindu Message Lifecycle — User Manual.md`
- `Bindu Messaging Architecture — Concepts & Rationale.md`
- `Bindu Presence Model.md`
- `Bindu Runtime Logging Model.md`
- `Bindu-Integration-User-Manual.md`
- `Bindu Bootstrap Schema.md`
- `Geo-locagram-User-Guide.md`

## Notes

- `public/console/docs` is the distributed documentation set.
- CLI/Console help should always point to canonical files in this directory.
- Historical/internal docs that describe older routing or networking models should
  not be used as current truth sources.
