# Bindu Routing Log Triage

> [!NOTE]
> **HUMAN OPERATOR / SYSADMIN REFERENCE ONLY**
> This document is intended for human system administrators and platform infrastructure operators. Autonomous AI Agents building Multi-Agent Systems (MAS) should NOT use this document and must strictly follow the Whitelisted Curriculum defined in [AGENTS.md](./AGENTS.md).

Role of this document:

- Supporting troubleshooting reference for routing and presence recovery
- Best used after the canonical routing guide when interpreting runtime logs

For current routing semantics, start with
`public/console/docs/routing_and_directionality_guide.md` and `public/console/docs/Bindu Routing & Mesh Delivery — User Guid.md`.

Use this page to interpret the most important routing and presence log lines during mesh recovery tests.

This is an operator-facing cheat sheet. It answers:

- which endpoint source was chosen
- whether the failure is routing, TCP reachability, or MQTT
- whether a node actually emitted fresh presence

## 1) Read These Fields First

When debugging a failed `root -> child` or `child -> root` delivery, look for:

- `dial.selected`
- `source=...`
- `endpoint=...`
- the concrete error text:
  - `no route to host`
  - `connect timeout`
  - `connection refused`
- whether you see `execution.trigger.emitted` for `NodePresencePlugin`

These lines usually tell you enough to classify the failure quickly.

## 2) What `source=` Means

`source=` does not mean "what the broker believes." It means "how Bindu selected the endpoint URL."

### `source=session`

This is now a legacy signal.

In the active publish path, Bindu does not use local transport/session state as routing authority.

If you still see `source=session`, it is coming from an older compatibility path or historical logs, not the minimal presence scheduler.

### `source=presence`

This is now a legacy signal for endpoint selection.

In the active publish path, `NodePresence` / `ParentPresence` act as publication triggers, not as the source of the endpoint URL.

If you still see `source=presence`, it is coming from an older compatibility path or historical logs.

### `source=config`

Bindu selected the endpoint from stable mesh/configuration.

This means:

- the active scheduler is using the structural routing path
- endpoint selection is not waiting on live session or fresh presence

### `source=last_known`

This is also a legacy signal.

The active minimal routing path does not depend on last-known transport hints.

## 3) What The Error Text Means

### `no route to host`

Example:

`dial tcp 192.168.12.186:1883: connect: no route to host`

This is a network reachability failure before MQTT.

Interpretation:

- the kernel could not find a working path to that IP
- the remote broker may be healthy, but the host was not reachable from this node at that moment
- this is not an MQTT auth/session problem

Common causes:

- child still asleep
- interface not back yet after wake
- wrong subnet/VLAN route
- stale IP from old presence

### `connect timeout`

Example:

`publisher: connect timeout`

This means:

- TCP/TLS connect did not complete in time
- there may be a route, but the far end did not accept or complete the connection fast enough

Common causes:

- remote node is booting
- packet loss
- firewall silently dropping
- broker port reachable only intermittently

### `connection refused`

This means:

- the host was reachable
- the target port was closed or no listener accepted the connection

Common causes:

- service is down
- wrong port
- Mochi/bindu process not listening yet

## 4) How To Confirm A Real Presence Recovery

Do not assume `run_now` means the node actually published `NodePresence`.

### Not sufficient

These lines only mean the schedule ran:

- `execution.run_now schedule_id=NodePresencePlugin:node.presence.tick`
- `execution.trigger.run plugin=NodePresencePlugin trigger=node.presence.tick`

### The line that matters

You want to see:

- `execution.trigger.emitted plugin=NodePresencePlugin ... count=1`

That means the plugin actually emitted a fresh `NodePresence`.

After that, you should also see:

- `type=NodePresence`
- `routing.queue.enqueue endpoint=parent ... type=NodePresence`

If `run_now` appears repeatedly but `execution.trigger.emitted` does not, the node is not actually publishing fresh presence on that attempt.

In the current minimal model, a fresh `Presence` matters because it re-opens a publish attempt after a failed cycle exhausted retries.

## 5) Why Restart Does Not Mean "No Memory"

Restarting `bindu-node` or `bindu-root` clears in-memory connection state, but it does not necessarily clear persisted presence records.

After a restart:

- persisted `NodePresence` / `ParentPresence` may still exist
- mesh config still exists

So after restart it is normal to see:

- `source=config` in the active scheduler path
- a new publish attempt when a fresh `Presence` is received

This is expected behavior.

## 6) Fast Diagnosis Patterns

### Pattern A: `source=config` + `no route to host`

Meaning:

- Bindu used the structural configured address
- that IP was not reachable at that moment

Conclusion:

- routing logic selected the configured address
- network path was not available yet

### Pattern B: `source=config` after restart

Meaning:

- runtime fell back to bootstrap/config successfully

Conclusion:

- the new recovery path is working
- the node/root is no longer blocked waiting for fresh presence

### Pattern C: `run_now` without `execution.trigger.emitted`

Meaning:

- `NodePresencePlugin` was triggered
- but no fresh message was actually emitted

Conclusion:

- do not treat this as a successful re-announcement

### Pattern D: repeated `publish.waiting_for_presence` after a failed attempt

Meaning:

- the last publish cycle exhausted its retries
- the destination is now paused until a new `ParentPresence` or `NodePresence` arrives

Conclusion:

- this is expected in the minimal model
- no background reconnect loop should continue running

## 7) Minimum Log Sequence For "Recovery Worked"

For a healthy recovery after sleep/wake or restart, look for this sequence:

1. node selects parent endpoint (`source=config`)
2. node successfully reconnects
3. node logs `execution.trigger.emitted ... NodePresencePlugin ... count=1`
4. root receives `NodePresence`
5. root can route `MessagePing`
6. destination node receives `MessagePing`
7. `MessagePong` returns

If the sequence stops before step 3, presence recovery did not complete.

If it stops after step 3, inspect root-side endpoint selection and network reachability.

## 8) Quick Rule

Use this shortcut:

- `source=` tells you where the address came from
- the error text tells you whether the failure is network or service level
- `execution.trigger.emitted` tells you whether fresh presence was actually published

That combination is usually enough to classify the incident.
