# Bindu in 3 Layers

Role of this document:

- Supporting mental-model overview
- Best used as a compact companion to the architecture overview

For the canonical public architecture map, see
`public/console/docs/Bindu Architecture Overview.md`.

**Semantics · Runtime · Operations**

Bindu is a technical mandala: a layered system where meaning, execution, and operation are separated by design.

## Layer 1: Semantics

What does this mean?

This layer declares intent, contracts, and policy:

- Message Types
- schema and validation contracts
- routing and retention semantics
- permissions and scope

Semantics does **not** execute code.

## Layer 2: Runtime

How does this happen?

This layer enforces declared intent:

- root/node runtime
- message validation and routing
- retention and lifecycle enforcement
- plugin execution supervision

Runtime does **not** invent behavior.

## Layer 3: Operations

How do humans and AI operate the system?

This layer applies and observes:

- bindu-studio workflows
- Console inspection and control surfaces
- bundles, rollout, diagnostics

Operations does **not** bypass Semantics or Runtime contracts.

## Why this matters

This separation makes Bindu:

- explainable
- auditable
- safer for AI-assisted operation
- easier to evolve without hidden coupling

## Mental model

`Semantics -> Runtime -> Operations`

Lower layers never depend on higher layers.

That is why Bindu can be both strict and adaptive: it is a technical mandala, not ad-hoc glue.
