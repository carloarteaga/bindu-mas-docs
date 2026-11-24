# Bindu Sovereign AI Fabric — Comprehensive MAS Feature List

Official capability catalog and architectural reference for the Bindu Sovereign AI Fabric and Multi-Agent Systems (MAS) specification corpus.

---

## 1. 100% Distributed Systems Requirements

* **Go-Native Single-Heap Engine & Sub-15W Footprint:**
  Compile-time role segregation (`bindu-root` vs `bindu-node`) running on a unified Go runtime. Achieves >10,000 events/sec CEL filtering and >2,500 msg/sec SQLite WAL persistence on sub-15W quad-core edge gateways (Intel N95, Raspberry Pi 4) with 38–46 MB resident RAM and zero container/Kubernetes runtime overhead.
  * *Reference Docs:* [`Bindu Architecture Overview.md`](./Bindu Architecture Overview.md) · [`Bindu in 3 Layers.md`](./Bindu in 3 Layers.md) · [`What is Bindu.md`](./What is Bindu.md)

* **mTLS 1.3 Mesh Topology & Sovereign Root CA:**
  Peer-to-peer mTLS 1.3 encrypted mesh networking with mutual identity verification. Features an internal autonomous Root Certificate Authority with programmatic bootstrapping (`/api/v1/ca.crt`), eliminating third-party cloud PKI dependencies.
  * *Reference Docs:* [`mtls_and_certificate_lifecycle_guide.md`](./mtls_and_certificate_lifecycle_guide.md) · [`mesh_user_guide.md`](./mesh_user_guide.md) · [`Bindu Node Installation Guide.md`](./Bindu Node Installation Guide.md)

* **Strict Hardware Ingress Boundary Contract:**
  Ingress is strictly gated through MQTT TLS 1.3 (Port 8883) and Secure WebSockets (Port 8084) on topic `bindu/in`. Generic HTTP POST message publication (`/api/v1/publish`) is explicitly prohibited to enforce strict transport-layer backpressure and deterministic delivery.
  * *Reference Docs:* [`raw_ingress_and_mqtt_adapters_guide.md`](./raw_ingress_and_mqtt_adapters_guide.md) · [`Bindu Messaging Architecture — Concepts & Rationale.md`](./Bindu Messaging Architecture — Concepts & Rationale.md) · [`publish_message_user_guide.md`](./publish_message_user_guide.md)

* **Dual-Purge Architecture & Autonomous Maintenance:**
  Semantic ledger pruning combining autonomous low-traffic nightly cleanup (`Store.PurgeScheduledEvents`) with authenticated on-demand REST API (`POST /api/v1/schedule-events/purge`) and Studio CLI commands, eliminating SQLite database bloat without human intervention.
  * *Reference Docs:* [`scheduled_events_user_guide.md`](./scheduled_events_user_guide.md) · [`retention_policies_guide.md`](./retention_policies_guide.md)

* **Zero-RAG Embedded WAL Ledger (`v_messages`):**
  High-performance embedded SQLite WAL ledger paired with BadgerDB disk spooling. Enables sub-millisecond local SQL queries across real-time and historical state without external database servers, vector stores, or network round-trips.
  * *Reference Docs:* [`queries_user_guide.md`](./queries_user_guide.md) · [`viewer_user_manual.md`](./viewer_user_manual.md) · [`message_user_guide.md`](./message_user_guide.md)

* **Mobile & Leaf Client Architecture Contract:**
  Handheld terminals, smartphones (Android/iOS), and field scanners are strictly classified as Leaf Clients (never mesh routing nodes). Features production-ready Kotlin and Swift SDKs connecting over MQTTS with automatic WGS84 GPS projection.
  * *Reference Docs:* [`mobile_and_leaf_client_guide.md`](./mobile_and_leaf_client_guide.md) · [`mobile-client-sdk/README.md`](./mobile-client-sdk/README.md)

* **Atomic Monotonic Lifecycle & Zero-Symlink Updates:**
  Airtight release management enforcing strictly monotonic versions (`0.X.Y`), symlink-free OTA bundle distribution (`binduUpdate-*.tar.gz`), atomic root/node activation, and instant rollback.
  * *Reference Docs:* [`lifecycle-user-manual.md`](./lifecycle-user-manual.md) · [`bindu_reset_and_reload_lifecycle_guide.md`](./bindu_reset_and_reload_lifecycle_guide.md)

---

## 2. Semantic EPN (Semantic Event Processing Networks)

* **Strict Typed Envelopes & Semantic Partition Triplet:**
  Every event is wrapped in a standardized envelope governed by the semantic triplet `($type, $group, $producer)`, guaranteeing deterministic validation, auditability, and partition-level state isolation.
  * *Reference Docs:* [`message_types_user_guide.md`](./message_types_user_guide.md) · [`Message Types for AI Agents.md`](./Message Types for AI Agents.md) · [`Bindu Bootstrap Schema.md`](./Bindu Bootstrap Schema.md)

* **Declarative Sliding-Window Retention Policies:**
  Dynamic pruning policies (`retentionMode: "count"`, 20–100 items for telemetry streams; `retentionValue: 1` for state definitions) preventing edge storage exhaustion from high-frequency sensor streams.
  * *Reference Docs:* [`retention_policies_guide.md`](./retention_policies_guide.md) · [`message_types_user_guide.md`](./message_types_user_guide.md)

* **Mesh Routing & Directionality Invariants:**
  Declarative event routing rules (`local`, `parent`, `child`). Strictly enforces bottom-up propagation for sensor streams and top-down distribution for edge configuration, preventing mesh reflection loops.
  * *Reference Docs:* [`routing_and_directionality_guide.md`](./routing_and_directionality_guide.md) · [`Bindu Routing & Mesh Delivery — User Guid.md`](./Bindu Routing & Mesh Delivery — User Guid.md) · [`topics_user_guide.md`](./topics_user_guide.md)

* **Real-Time Geo-Locagram Spatial Projection (`$where`):**
  Native support for WGS84 coordinates (`lat`, `lon`, `alt`) embedded in event envelopes, automatically projected onto the Web Console Geo-Locagram map in real time without external GIS databases.
  * *Reference Docs:* [`Geo-locagram-User-Guide.md`](./Geo-locagram-User-Guide.md) · [`Geo-Locagram-Query-Patterns.md`](./Geo-Locagram-Query-Patterns.md) · [`bindu-studio-geo-locagram-user-guide.md`](./bindu-studio-geo-locagram-user-guide.md)

* **Raw Ingress Adapters & Ephemeral RAM Routing:**
  Zero-overhead binary ingestion supporting legacy industrial hardware (RTSP, Modbus, BLE, RFID). Translates proprietary byte frames directly in volatile RAM via Google CEL without polluting disk WAL storage.
  * *Reference Docs:* [`raw_ingress_and_mqtt_adapters_guide.md`](./raw_ingress_and_mqtt_adapters_guide.md) · [`Bindu Messaging Architecture — Concepts & Rationale.md`](./Bindu Messaging Architecture — Concepts & Rationale.md)

* **Node Presence & Fault Detection:**
  Autonomous heartbeat broadcasting (`NodePresence`) with timeout-driven disconnect detection, providing decentralized visibility of mesh topology health.
  * *Reference Docs:* [`Bindu Presence Model.md`](./Bindu Presence Model.md) · [`Bindu Routing Log Triage.md`](./Bindu Routing Log Triage.md)

---

## 3. MAS / Multi-Agent Solution Architecture

* **The 6 Standard Solution Artifacts & Level 0 DAG:**
  Declarative solution authoring structured into a verifiable Directed Acyclic Graph: Schemas (`poc_message_types.json`), CEL Reactors, Semantic AI Reactors, SQL Dashboards, BPR Plugins, and the atomic Solution Manifest (`solution_<slug>.json`).
  * *Reference Docs:* [`BINDU.md`](./BINDU.md) · [`how-to-build-a-bindu-solution.md`](./how-to-build-a-bindu-solution.md) · [`mas-solution-tutorial/tutorial_guide.md`](./mas-solution-tutorial/tutorial_guide.md)

* **Google CEL Kinetic Reactors (Horn Clause Engine):**
  Sub-microsecond deterministic logic execution evaluating Horn clauses ($P_1 \land P_2 \dots \rightarrow Q$) in compiled Go memory, writing immutable derivation proof trees into SQLite WAL with zero LLM hallucinations.
  * *Reference Docs:* [`cel_reactor_user_guide.md`](./cel_reactor_user_guide.md) · [`cel_context_and_functions_guide.md`](./cel_context_and_functions_guide.md) · [`cle-reactor-tutorial/tutorial_guide.md`](./cle-reactor-tutorial/tutorial_guide.md)

* **Semantic AI Reactors (The 3-Tier CEL+AI Sandwich):**
  Bounded cognitive inference where SLMs and VLMs function strictly as Ground Term Extractors ($P_1$) or Bounded Synthesizers over pre-proven context, featuring dynamic Zero-RAG hydration and Zero-Plaintext vault secrets.
  * *Reference Docs:* [`semantic_reactor_user_guide.md`](./semantic_reactor_user_guide.md) · [`AISemanticReactorSet Schema & Authoring Guide for AI.md`](./AISemanticReactorSet Schema & Authoring Guide for AI.md) · [`semantic-ai-reactor-tutorial/tutorial_guide.md`](./semantic-ai-reactor-tutorial/tutorial_guide.md)

* **BPR External Process Plugin Contract:**
  OS-level binary process isolation for external device integration (cameras, Modbus PLCs, serial buses). Communication is strictly standard JSON over `stdin`/`stdout`, eliminating hidden HTTP sidecars or shared-memory crashes.
  * *Reference Docs:* [`plugin_execution_model_user_guide.md`](./plugin_execution_model_user_guide.md) · [`bindu-bpr-go-sdk-developer-manual.md`](./bindu-bpr-go-sdk-developer-manual.md) · [`hello-world-plugin-and-spa-tutorial.md`](./hello-world-plugin-and-spa-tutorial.md)

* **Declarative REST Reactors:**
  In-engine REST integration mapping external webhooks and HTTP APIs into typed SEPN events and outbound dispatches without writing custom microservices.
  * *Reference Docs:* [`rest_reactor_user_guide.md`](./rest_reactor_user_guide.md) · [`rest-reactor-user-manual.md`](./rest-reactor-user-manual.md) · [`rest-reactor-tutorial/tutorial_guide.md`](./rest-reactor-tutorial/tutorial_guide.md)

* **Comprehensive Pedagogical Tutorial Corpus:**
  Complete, runnable real-world MAS implementations:
  * **Edge CV:** Native C++ ONNX Runtime & YOLOX-S inference with on-demand bootstrap ([`edge-cv-tutorial/tutorial_guide.md`](./edge-cv-tutorial/tutorial_guide.md)).
  * **Potato Processing:** Predictive thermodynamic control with Modbus TCP/IP client ([`potato-processing-pipeline-tutorial/tutorial_guide.md`](./potato-processing-pipeline-tutorial/tutorial_guide.md)).
  * **RFID EPC Logistics:** Industrial warehouse asset tracking and anti-collision filtering ([`rfid-epc-tutorial/tutorial_guide.md`](./rfid-epc-tutorial/tutorial_guide.md)).
  * **BLE Beacons:** Personnel safety and micro-location tracking ([`ble-tutorial/tutorial_guide.md`](./ble-tutorial/tutorial_guide.md)).
  * **Agroforestry Carbon:** Amazon Suzano carbon sequestration auditing ([`amazon-suzano-carbon-tutorial/tutorial_guide.md`](./amazon-suzano-carbon-tutorial/tutorial_guide.md)).
  * **Mining Fleet:** Predictive heavy machinery telemetry and maintenance dispatch ([`mining-heavy-equipment-tutorial/tutorial_guide.md`](./mining-heavy-equipment-tutorial/tutorial_guide.md)).

---

## 4. bindu-studio / AI Tooling

* **Universal 5-Phase Deployment Pipeline:**
  Standardized methodology collapsing distributed systems engineering into five deterministic steps: *Intent Alignment -> Synthesis & Schema Lock -> Local Certification (`dry-run`) -> Atomic Packaging -> Mesh Rollout*.
  * *Reference Docs:* [`BINDU.md`](./BINDU.md) · [`Co-Creation Guide.md`](./Co-Creation Guide.md) · [`Bindu Day 1 — Solution Workspace and AI Agent Setup Guide.md`](./Bindu Day 1 — Solution Workspace and AI Agent Setup Guide.md)

* **Autonomous AI Agent Operating Guardrails:**
  Comprehensive persona contracts and prompt specifications for LLMs and autonomous coding assistants (`AGENTS.md`, `GEMINI.md`, `CLAUDE.md`), guaranteeing schema consistency and adherence to platform invariants.
  * *Reference Docs:* [`AGENTS.md`](./AGENTS.md) · [`GEMINI.md`](./GEMINI.md) · [`CLAUDE.md`](./CLAUDE.md) · [`What Bindu Means to an AI Agent.md`](./What Bindu Means to an AI Agent.md)

* **Mandatory Dry-Run Verification Contract:**
  Strict developer contract requiring execution of `bindu-studio <cel-reactor|semantic-ai-reactor|rest-reactor|plugin> dry-run` to prove compilation and template hydration before deployment.
  * *Reference Docs:* [`bindu-studio-plugin-user-guide.md`](./bindu-studio-plugin-user-guide.md) · [`bindu_studio_solution_user_guide.md`](./bindu_studio_solution_user_guide.md)

* **Studio CLI Command Arsenal:**
  Industrial CLI suite for node provisioning, mesh topology inspection, real-time event sniffing, reactor certification, and scheduled event maintenance.
  * *Reference Docs:* [`Bindu Studio — User Guide (Command Overview).md`](./Bindu Studio — User Guide (Command Overview).md) · [`Bindu Studio Mesh User Manual.md`](./Bindu Studio Mesh User Manual.md)

* **Atomic Solution Bundling & Rollout Wizard:**
  Automated solution lifecycle tool (`bindu-studio mas bundle -> deploy -> apply`) enforcing schema synchronization to edge nodes prior to reactor activation.
  * *Reference Docs:* [`bindu_studio_solution_user_guide.md`](./bindu_studio_solution_user_guide.md) · [`mas-deployment-guide.md`](./mas-deployment-guide.md) · [`mas-complete-lifecycle-tutorial/tutorial_guide.md`](./mas-complete-lifecycle-tutorial/tutorial_guide.md)

* **Web Console Observability & SQL Dashboards:**
  Real-time browser-based dashboard engine (`public/console`) supporting live SQL queries against `v_messages`, configurable canvas zoom physics, and topological inspection.
  * *Reference Docs:* [`dashboards_user_guide.md`](./dashboards_user_guide.md) · [`ontology_user_guide.md`](./ontology_user_guide.md) · [`viewer_user_manual.md`](./viewer_user_manual.md)
