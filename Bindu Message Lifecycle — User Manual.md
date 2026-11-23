# Bindu Message Lifecycle — User Manual

Role of this document:

- Supporting operational lifecycle guide
- Best used when tracing how a message moves through ingress, local handling, persistence, and forwarding

For canonical routing semantics, see
`public/console/docs/Bindu Routing & Mesh Delivery — User Guid.md`.

## Purpose

This document explains the current lifecycle of a semantic message in Bindu:

- ingress
- validation
- local delivery
- persistence
- forwarding
- expiry

It is intended for operators, plugin authors, and message-type authors.

## 1. Scope of this guide

This guide focuses on the runtime journey of a message after it is created.
It is about execution flow, not full routing contract design.

A Bindu message is still governed by its Message Type:

- `routing`
- `persist`
- retention
- longevity

## 2. Minimal routing context

Current routing is an explicit list of directions.

Valid routing values are:

- `local`
- `parent`
- `child`

Examples:

```json
{ "routing": ["local"] }
{ "routing": ["parent"] }
{ "routing": ["local", "parent"] }
{ "routing": ["child"] }
```

`both` is not the current routing model.

This document only uses routing values to explain lifecycle effects. For the
full semantics of intent, `$node-dest`, and mesh delivery, use the routing
guide.

## 3. Lifecycle overview

Current high-level flow:

```text
bindu/in -> dispatcher -> validation -> local delivery? -> persist? -> forward?
```

Important boundary:

- `bindu/in` is the only ingress topic
- canonical topics are terminal outputs and must not re-enter routing

## 4. Message creation

A message may be created by:

- MQTT ingress
- external publishers
- plugins
- system schedulers
- built-in system plugins

Important fields at creation time:

- `$type`
- `$when`
- `$producer`
- `$group`
- `$node-dest` (optional targeted child routing)

## 5. Routing checkpoints inside the lifecycle

### `local`

- makes the message eligible for local runtime handling
- enables plugin delivery and local observers/subscribers
- enables local persistence if `persist=true`

### `parent`

- adds an upstream forwarding step if a parent exists
- does not imply local handling by itself
- on root, there is no upstream hop

### `child`

- adds child forwarding work to the routing queue
- if `$node-dest` selects a specific child, forwarding narrows to that branch
- does not imply local handling by itself

## 6. Persistence semantics

`persist` controls local storage only.

Rules:

- `persist=true` does not mean "persist everywhere"
- persistence happens only when the message is delivered locally
- forward-only routing does not persist

Examples:

- `["local"] + persist=true` -> local store allowed
- `["parent"] + persist=true` -> forward only, no local persistence
- `["local","parent"] + persist=true` -> local persistence plus upstream forward

## 7. Longevity and expiry

Longevity defines how long a message remains relevant.

Operationally:

- expired messages should not continue circulating in queues
- expiry affects forwarding relevance
- retention and longevity are different concepts

Mental model:

- routing = where
- persist = whether local storage is allowed
- retention = how much local history remains
- longevity = how long the message still matters

## 8. Presence messages in the lifecycle

`NodePresence` and `ParentPresence` are important control-plane messages.

Current operational role:

- publish fresh reachability information
- reopen lanes waiting on fresh presence
- provide endpoint hints for routing

Presence is helpful for delivery recovery, but it is not the semantic authority
of topology.

## 9. Forwarding vs local delivery

Forwarding and local delivery are separate decisions.

Forwarding:

- async
- queue-based
- endpoint-resolved
- may depend on fresh presence data

Local delivery:

- plugin delivery
- local observers/subscribers
- optional local persistence

## 10. Ping / pong example

Current style example:

```json
{ "name": "MessagePing", "routing": ["local", "child"], "persist": false }
{ "name": "MessagePong", "routing": ["local", "parent"], "persist": false }
```

Meaning:

- `MessagePing` is observable locally and may be forwarded to a child
- `MessagePong` is observable locally and may be forwarded back upstream

This example is useful as a lifecycle trace. For the authoritative routing
interpretation, prefer the routing guide.

## 11. Common mistakes to avoid

- assuming `persist=true` means remote nodes must store the message
- assuming forwarding implies local delivery
- treating presence as durable state authority
- reasoning with `both` as if it were current routing syntax

## 12. Summary

Bindu message lifecycle is governed by explicit contracts:

- ingress enters through `bindu/in`
- dispatcher validates and normalizes
- routing decides local and/or remote relevance
- persistence is local-only and contract-driven
- forwarding is explicit, queued, and endpoint-resolved
