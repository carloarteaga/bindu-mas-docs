# Bindu Messaging Architecture — Concepts & Rationale

Role of this document:

- Supporting rationale and design-intent guide
- Best used for understanding why Bindu separates ingress, persistence, routing, and plugin execution

For the canonical routing contract, see
`public/console/docs/Bindu Routing & Mesh Delivery — User Guid.md`.
For the operational flow, see
`public/console/docs/Bindu Message Lifecycle — User Manual.md`.

## Purpose

This page explains the architectural reasoning behind the current runtime
behavior. It is intentionally about design choices more than step-by-step
operations.

## 1. Governing idea

Bindu treats messages as semantic events with explicit contracts.

The runtime exists to preserve that meaning as messages move through the
system. The key design question is not only "where does a message go?" but
"which part of the system is allowed to decide that?"

## 2. Why there is a single ingress boundary

Bindu uses one ingress topic, `bindu/in`, because a system needs a single place
where new semantic facts become subject to validation, authorization, storage
rules, and execution.

Canonical topics are intentionally terminal outputs. They are for observation,
state distribution, and subscriptions. Treating them as ingress would mix
outputs back into semantic decision-making and create duplicates or loops.

## 3. Why the dispatcher is the semantic gate

The dispatcher is not just a parser. It is the boundary where the runtime
commits to a message as a governed fact.

Its role matters because it keeps semantics centralized:

- message structure is normalized once
- message-type validation happens once
- local handling, persistence, and forwarding are derived from declared policy

This prevents plugins, clients, or ad-hoc transports from becoming alternative
sources of truth.

## 4. Why local delivery and forwarding are separate

Bindu deliberately separates local relevance from remote propagation.

That distinction keeps the model predictable:

- local handling answers whether the current node should react
- forwarding answers whether the message should continue through the mesh
- persistence answers whether the current node may retain local history

These are related, but they are not the same decision.

## 5. Why persistence is local-only

Persistence is tied to the node that handles the message locally.

This avoids ambiguous semantics such as "persist everywhere the message passes
through" and keeps storage behavior explicit. A forward-only message may travel
through the mesh without creating unintended copies on every hop.

This is one of Bindu's most important design boundaries:

- routing is about reach
- persistence is about local retention

## 6. Why plugins do not own transport authority

Plugins react to semantic messages and may emit new semantic messages, but they
do not decide transport behavior directly.

That separation keeps extensions powerful without letting them silently redefine
platform behavior. A plugin can express intent by emitting a message; the
runtime still applies validation, routing, and storage policy.

## 7. Why presence is a delivery aid, not topology truth

Presence helps the routing engine recover and resume delivery when endpoints
become usable again.

It is useful because delivery in a distributed system depends on liveness, but
presence does not replace declared topology or message intent. Topology comes
from configuration. Routing comes from contracts. Presence only helps delivery
catch up to reality.

## 8. Message design principles

- Declarative semantics before mechanics
- Local delivery and forwarding are independent decisions
- Persistence is local-only
- Canonical publishes are terminal outputs
- Plugins extend behavior without owning transport authority

## 9. Mental model

Use this model when reasoning about a Message Type:

- routing decides where it is relevant
- persist decides whether local storage is allowed
- retention decides how much local history remains
- longevity decides how long the message still matters

Bindu treats messages as governed flows, not ad-hoc transport artifacts.
