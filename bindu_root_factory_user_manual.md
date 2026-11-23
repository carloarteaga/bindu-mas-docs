# Bindu Root Factory — User Manual (Zero → Run v1)

## Purpose

This document describes the **Bindu Root Factory workflow** for provisioning and running a Bindu mesh from **zero to operational**, focusing on **real-world usability**, **devices without UI**, and **deterministic deployment**.

The model prioritizes:
- simplicity
- explicit configuration
- reproducibility
- minimal operational friction

Updates and distributed reconfiguration are **out of scope** for this version.

---

## Core Principle

> **Root builds. Nodes apply.**  
> **Semantics are global. Execution is local.**

---

## Architecture Overview

Bindu provisioning is based on a **Root Factory** model:

- **Root** is the single place where:
  - the solution is defined
  - configuration is validated
  - plugins are built and tested
  - node bundles are generated

- **Nodes** are dumb by design:
  - they do not decide
  - they do not derive secrets
  - they only apply bundles and run

---

## Supported Deployment Models

### Model A — Root on a local PC (recommended)
- Root runs on a developer/operator PC.
- All build, test, and bundle generation happens locally.
- Bundles are copied to devices manually.

### Model B — Root in the cloud, Factory on PC
- Root runs in the cloud as a control plane.
- Operator downloads `mesh.json` and config locally.
- Bundles are built locally and applied to nodes.

> The **factory build** should run where tooling and debugging are comfortable.

---

## Step 1 — Install Root (Root Installer)

### What the Root Installer provides

The Root Installer installs:
- Bindu root binary
- Console (UI)
- Configuration directory structure
- Plugin development/build tooling
- TLS base configuration

### Required prerequisites (Root)

| Use case | Requirements |
|--------|--------------|
| Configuration only | None (prebuilt binaries) |
| Plugin development | Go toolchain |
| Advanced tooling | Optional Python (fabric/automation) |

> If the user does **not** build plugins, **no Go or Python is required**.

---

## Step 2 — Define the Mesh (`mesh.json`)

The mesh topology is defined **only in root**.

`mesh.json` describes:
- node identifiers
- platforms / architectures
- IP addresses and ports
- parent ↔ child relationships
- which plugins run on which node

There is **no discovery** and **no negotiation**.

---

## Step 3 — Develop and Test the Solution in Root

### Message Types
- Message Types and their schemas are owned by Bindu.
- Schemas are **immutable** at this stage.
- All nodes share the same semantic model.

### Plugins
- Legacy `.so` plugin development under `plugins-src/` is no longer supported; active plugin runtime uses built-in system plugins and external BPR packages.
- Built and validated in root
- Artifacts are produced into `plugins.d/`
- Tested in root before deployment

### Configuration
- All configuration lives under `config/`
- Includes:
  - message type definitions
  - mesh.json
  - plugin configuration

---

## Step 4 — Build Node Bundles (Bundle Builder)

The Bundle Builder assembles **per-node bundles**.

Each bundle contains:
- `bootstrap.json` (candidate parent addresses, mesh MQTT port, root signing URL)
- `identity/node_identity.json` (logical node ID)
- `bootstrap/ca.crt` (deployment CA certificate)
- `public/console/schema/**` (required message type schemas)
- `config/**` (node runtime configuration)
- `plugins.d/**` (plugins assigned to the node)
- `bindu-node` (launcher supervisor and runtime binary)
- `bindu-install-node` (self-contained node installer)

Bundles are deterministic and reproducible.

Output:
```text
dist/node-<node_id>-linux-<arch>-<version>.tar.gz
```

---

## Step 5 — Apply Bundle on Node (Zero → Run)

On the target device, extract the tarball and run `bindu-install-node`:

```bash
tar -xzf node-worker-01-linux-amd64-0.0.500.tar.gz
cd node-worker-01-linux-amd64-0.0.500
sudo ./bindu-install-node --bundle . --token "<enrollment-token>" --install-service --service-auto-start
```

This operation:
1. Validates bundle integrity and guards against private key leaks.
2. Checks TCP reachability to candidate parent addresses.
3. Executes preflight validation with Root (`POST /api/v1/tls/preflight-node`).
4. Generates a local 2048-bit RSA private key and submits a CSR to Root (`POST /api/v1/tls/sign-node`).
5. Stores signed certificates (`runtime/tls/server.crt`) and private keys (`runtime/tls/server.key`, mode `0600`).
6. Configures and registers a native Linux `systemd` service unit.
7. Enables and starts the `bindu-node` service supervisor.

No UI. No manual prompt requirements when token is supplied.

---

## Security Model

- **Mandatory mTLS:** All communication between nodes and root is encrypted and authenticated using mutual TLS (mTLS).
- **Private Key Isolation:** Private keys are generated strictly on the target node; they are never sent over the wire or stored in deployment bundles.
- **Role-Based File Permissions:** POSIX permissions strictly enforce `0700` on `runtime/tls`, `0600` on private keys, `0750`/`0640` on config and identity, and `0770`/`0660` on runtime data and logs.
- **Autonomous Renewal:** Active nodes autonomously monitor certificate expiration and renew their certificates via `POST /api/v1/tls/renew-node` before expiry.

---

## Summary

This workflow enables:
- Deterministic provisioning.
- Real-world usability on headless edge devices.
- Zero-trust cryptographic enrollment.
- Minimal operational friction with single-command deployment.
