# 🤖 AGENTS.md — AI Solution Creator Adapter for Bindu MAS

> [!IMPORTANT]
> **CANONICAL SPECIFICATION NOTICE**:
> This file is a lightweight entrypoint adapter for **OpenAI Codex, ChatGPT, GitHub Copilot, Cursor, and Agentic Toolchains**.
> 
> The authoritative, model-agnostic architectural specification governing how AI Agents design, validate, package, certify, and deploy Multi-Agent System (MAS) solutions on Bindu is located in:
> 
> 👉 **[`public/console/docs/BINDU.md`](./BINDU.md)**

## Mandatory Instruction for Autonomous Agents

When tasked with creating, modifying, deploying, or verifying a Bindu MAS solution, business schemas (`poc_message_types.json`), CEL kinetics, AI semantic reactors, dashboards, BPR plugins, or tutorials:

1. **Read Canonical Specification:** Consult [`public/console/docs/BINDU.md`](./BINDU.md) for the complete 3-Layer Invariants, Level 0 DAG, the 6 Standard Artifacts, and the Universal 5-Phase Deployment Pipeline.
2. **Zero-Plaintext Wire Guarantee:** Never emit plaintext credentials in JSON manifests or wire payloads. Always use `"secret:KEY_NAME"` or `secret('KEY_NAME')` backed by the encrypted vault (`accessControlConfig.brs`).
3. **Validation-First Discipline:** Always execute `bindu-studio <cel-reactor|semantic-ai-reactor|rest-reactor|plugin> dry-run` with representative payloads before deployment. Syntactic `validate` alone is never sufficient.
4. **Native Observability:** Query ledger state strictly via the native REST endpoint `POST /api/v1/messages/sql` on `v_messages`. Never inspect raw `.db` files directly on disk.
5. **Deductive Logic vs. Perceptual AI (Horn Clause Contract):** CEL reactors execute formal Horn clause deduction ($P_1 \land P_2 \dots \rightarrow Q$) in sub-microsecond Go memory. AI models (SLMs, VLMs, TypeSafe `Jev`) are strictly **Ground Term Extractors** ($P_1$) or bounded synthesizers. Never delegate business policy, safety cutoffs, or threshold derivations to an unconstrained AI prompt.
6. **Mobile & Leaf Client Rule:** When developing mobile applications (Android/iOS) or handheld terminal integrations, NEVER embed or run a `bindu-node` daemon inside mobile OSes. Implement mobile devices strictly as Leaf Clients over MQTT TLS/mTLS following [`mobile_and_leaf_client_guide.md`](./mobile_and_leaf_client_guide.md) and reference wrappers in [`mobile-client-sdk/`](./mobile-client-sdk/).

Refer to [`BINDU.md`](./BINDU.md) for full architectural patterns, CLI cheat sheets, and the 15 MAS archetypes.
