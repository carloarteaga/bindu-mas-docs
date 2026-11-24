# Bindu Multi-Agent Systems (MAS) — Official Specification & Knowledge Corpus

> **Official specification, architectural guides, and pedagogical corpus for building deterministic Multi-Agent Systems (MAS) with the Bindu Sovereign AI Fabric.**

Welcome to the central repository for designing, implementing, and deploying Multi-Agent Systems on Bindu. This corpus is maintained as the authoritative black-box knowledge base for software engineers, application architects, and AI coding agents (Cursor, Windsurf, Claude Code, Copilot, ChatGPT).

---

## 🧭 Entrypoints for AI Agents & Solution Creators

If you are an AI assistant or human engineer building a new MAS solution or extending an existing one, start with your model's native entrypoint adapter:

| AI Agent / Tooling | Entrypoint File | Purpose |
| :--- | :--- | :--- |
| **Universal Black-Box Spec** | [`BINDU.md`](./BINDU.md) | The master contract: 3-layer invariants, Level 0-4 Artifact DAG, and 5-phase deployment pipeline. |
| **Feature Inventory & AI Prompt** | [`Bindu_capabilities_and_competitors.md`](./Bindu_capabilities_and_competitors.md) | Documented 5-pillar architectural capability catalog + Objective AI Evaluation Challenge. |
| **Comprehensive Feature List** | [`MAS_FEATURE_LIST.md`](./MAS_FEATURE_LIST.md) | Granular inventory across Distributed Systems, Semantic EPN, MAS, and Tooling. |
| **Cursor / Codex / Windsurf** | [`AGENTS.md`](./AGENTS.md) | Standard coding agent instructions and rules. |
| **Google Gemini / Antigravity** | [`GEMINI.md`](./GEMINI.md) | Gemini-specific system prompt and operating rules. |
| **Anthropic Claude Code** | [`CLAUDE.md`](./CLAUDE.md) | Claude-optimized project guidelines. |
| **Day 1 Solution Workspace** | [`Bindu Day 1 — Solution Workspace and AI Agent Setup Guide.md`](./Bindu%20Day%201%20%E2%80%94%20Solution%20Workspace%20and%20AI%20Agent%20Setup%20Guide.md) | How to scaffold a new MAS workspace from scratch. |

---

## 🏛️ Core Architectural Guides

Deep-dive references for specific platform mechanics:

- **[Architectural Capabilities & Verifiable Feature Inventory](./Bindu_capabilities_and_competitors.md):** The complete documented capability catalog for objective AI and specialist technical evaluation across distributed systems physics, autonomous DevOps/OTA, Semantic EPN, bounded Horn clause AI, and MAS engineering.
- **[Comprehensive MAS Feature List](./MAS_FEATURE_LIST.md):** Granular breakdown of 100% distributed systems requirements, Semantic EPN, and AI tooling.
- **[Routing & Mesh Directionality Guide](./routing_and_directionality_guide.md):** Decision tree across bottom-up sensor telemetry (`["local", "parent"]`), top-down control (`["local", "child"]`), peer gossip, and private local states.
- **[Retention Policies & Semantic Partitioning Guide](./retention_policies_guide.md):** The Triplet contract `($type, $group, $producer)` and sliding-window count vs calendar day pruning.
- **[Distributed Scheduled Events & Timers](./scheduled_events_user_guide.md):** Precision time-based state machines (`SystemScheduledEvent`, `SystemCronJob`, `SystemScheduledEventCancel`).
- **[Secrets Management & Vault Security](./secrets_management_user_guide.md):** Hardware-bound encrypted key vault (`accessControlConfig.brs`) and the Zero-Plaintext Wire guarantee.
- **[Out-of-Band Media & S3 Storage](./media_storage_user_guide.md):** Detached binary storage (MinIO/S3), presigned URLs, and multimodal AI vision pipelines.
- **[Mobile & Leaf Client Contract](./mobile_and_leaf_client_guide.md):** Native Android/iOS telemetry ingest, Geo-Locagram WGS84 mapping, and MQTTS/WSS contracts.
- **[mTLS & Certificate Lifecycle Guide](./mtls_and_certificate_lifecycle_guide.md):** Zero-trust mesh identity, short-lived leaf certificates, and autonomous enrollment.

---

## 🏭 Industrial Solution Archetype Catalog

This repository includes complete, end-to-end reference tutorials and solution packages covering real-world physical and cognitive domains:

| Solution Directory | Industry / Domain | Architectural Highlights |
| :--- | :--- | :--- |
| [`mas-complete-lifecycle-tutorial/`](./mas-complete-lifecycle-tutorial/) | Baseline Reference | All 6 standard artifacts, minimal BPR plugin, CEL kinetics, AI reactor, and automated scripts. |
| [`mas-solution-tutorial/`](./mas-solution-tutorial/) | Packaging & Bundles | Declarative manifests (`solution.json`), multi-artifact versioning, and atomic distribution. |
| [`potato-processing-pipeline-tutorial/`](./potato-processing-pipeline-tutorial/) | Industrial Food & Agriculture | Sliced potato optical inspection via `llava-phi3`, thermodynamic CIS mitigation, Modbus TCP PLC driver, and schedule dashboards. |
| [`edge-cv-tutorial/`](./edge-cv-tutorial/) | Multimodal Edge Vision | Embedded ONNX Runtime (YOLOX-S), multi-camera RTSP capture via low-delay pipes, and CEL overlap arbitration. |
| [`semantic-ai-reactor-tutorial/`](./semantic-ai-reactor-tutorial/) | Pharmaceutical Cold Chain | Zero-RAG SQLite state hydration, TypeSafe System One (`jev-latest`) vs local Ollama, and parallel `noul`/`choice`/`score` primitives. |
| [`rfid-epc-tutorial/`](./rfid-epc-tutorial/) | Warehouse & Logistics | High-frequency EPC tag bursts, sliding-window retention (`count: 100`), temporal deduplication, and sensor aggregation. |
| [`ble-tutorial/`](./ble-tutorial/) | Indoor Asset Tracking | Bluetooth Low Energy beacon scanning, advertisement parsing, proximity event routing, and presence tracking. |
| [`mining-heavy-equipment-tutorial/`](./mining-heavy-equipment-tutorial/) | Heavy Mining & CBM | Triaxial vibration telemetry ($\pm 16g$), drive-by store-and-forward ring buffer timeline reconstruction, Crest Factor/Kurtosis CEL gating, and SLM predictive maintenance RUL. |
| [`amazon-suzano-carbon-tutorial/`](./amazon-suzano-carbon-tutorial/) | Airborne Forestry & ESG | Autonomous drone data-mule harvesting, sub-canopy solar BLE sentinel stations, stoichiometric CEL Horn clauses, and verifiable carbon credit proof trees. |
| [`system-email-bpr-tutorial/`](./system-email-bpr-tutorial/) | Critical Alerting | Secure SMTP email delivery via BPR plugin with dynamic secret resolution when CEL detects critical anomalies. |
| [`media-storage-tutorial/`](./media-storage-tutorial/) & [`cleanup`](./media-storage-cleanup-tutorial/) | Vision Object Storage | MinIO/S3 image ingestion, presigned upload URLs, multimodal vision inference, and automated 2-stage TTL garbage collection. |
| [`rest-reactor-tutorial/`](./rest-reactor-tutorial/) | Enterprise ERP Integration | Declarative zero-code REST webhook egress bridges triggered by ledger events. |
| [`using-system-message-types/`](./using-system-message-types/) | Fleet Observability | Cluster audit of `MessageState` heartbeats, fleet storage health scoring, and anomaly detection. |

---

## 📦 The 6 Standard Solution Artifacts

Every Bindu MAS application consists of declarative artifacts that form a verifiable Directed Acyclic Graph (DAG):

1. **`poc_message_types.json` (Level 0):** Schema ontology defining types, validation constraints, retention, and routing.
2. **`poc_cel_reactor_config.json` (Level 2):** Deterministic Google CEL kinetic reactors and Horn clause rule engines.
3. **`poc_semantic_reactor_config.json` (Level 2):** Cognitive SLM/VLM reactors with dynamic context hydration and Zero-Plaintext vault secrets.
4. **`poc_dashboards.json` (Level 2):** Real-time Zero-RAG SQL monitoring dashboards for the Web Console.
5. **Standalone Go BPR Plugins (Level 1):** OS-level binary plugins executing external I/O (cameras, Modbus PLCs, BLE scanners) communicating strictly via standard JSON `stdin`/`stdout`.
6. **`solution_<slug>.json` (Level 3):** Master solution manifest binding all schemas, reactors, dashboards, and hardware node assignments into an atomic deployment bundle.

---

## 📜 License

Documentation, specifications, and reference tutorial code are licensed under the MIT License.
