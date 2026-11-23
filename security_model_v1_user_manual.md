# Bindu Security Model & Hardening Architecture (v1)
## Mesh Cryptography, Transport Isolation & Access Control

This document provides the authoritative, complete specification of the security architecture in Bindu. It is intended for security engineers, operators, developers, and AI agents.

---

## 1. Architectural Security Foundations

Bindu is built on a **Defense-in-Depth, Zero-Trust Execution Model** divided into three strict security planes:

1. **Mesh Infrastructure Plane:** Authenticated peer-to-peer mTLS routing, certificate signing, and autonomous lifecycle management.
2. **Application & Integration Plane:** Scoped REST APIs, role-based access control (RBAC), topic-level MQTT ACLs, and anti-spoofing provenance.
3. **Runtime & Plugin Sandboxing Plane:** Cryptographic secret protection (PBKDF2), kernel-level process containment, and strict environment variable whitelisting.

```mermaid
flowchart TD
    subgraph MESH["1. Mesh Infrastructure Plane (mTLS / Port 8884)"]
        direction TB
        RootCA["Root CA & Signer (/sign-node, /renew-node)"]
        MeshNodes["Mesh Nodes (Root & Gateway Nodes)"]
        RootCA <-->|mTLS + AEAD Ciphers| MeshNodes
        MeshNodes -->|Strict Topic Ingress| Ingress["bindu/in"]
        MeshNodes <--|Queue Egress| Egress["bindu/out/node_id"]
    end

    subgraph APP["2. Application & Integration Plane (MQTTS 8883 / WSS 8084 / WS 8083)"]
        direction TB
        ExternalApps["External Apps & Sensors (MQTTS)"]
        WebSPA["External Web Dashboards (WSS)"]
        LocalConsole["Bindu Web Console (Loopback WS 8083)"]
        RBAC["Access Control (Admin & Application Roles)"]
        Tokens["Ephemeral Session Tokens (sessionID UUID)"]
        
        ExternalApps --> RBAC
        WebSPA --> RBAC
        LocalConsole --> Tokens
    end

    subgraph SANDBOX["3. Runtime & Sandboxing Plane"]
        direction TB
        ConfigStore["Encrypted Storage (PBKDF2 100k iters + AES-GCM)
        • accessControlConfig.brc (RBAC & Users)
        • accessControlConfig.brs (Integration Secrets Vault)"]
        SecretRef["Zero-Plaintext Wire Guarantee (secret:KEY_NAME references)"]
        BPRSandbox["BPR Plugin Sandbox (Whitelisted Env + Setpgid + SIGKILL)"]
        ConfigStore --> SecretRef
    end

    MESH --- APP
    APP --- SANDBOX
```

---

## 2. Network Channels & Transport Isolation

Bindu exposes four purpose-built communication channels, each with distinct cryptographic guarantees and listener bindings:

| Channel | Port | Protocol | Encryption / TLS Mode | Authentication Method | Primary Consumer / Purpose |
|:---|:---|:---|:---|:---|:---|
| **1. mesh** | `8884` | MQTT / TCP | **mTLS** (RequireAndVerifyClientCert, AEAD only) | X.509 Node Certificates | Inter-node mesh routing between Root and Gateway nodes. |
| **2. mqtts** | `8883` | MQTT / TCP | **TLS** (NoClientCert, AEAD only) | Username / Password (RBAC) | External backend applications, sensors, and IoT collectors. |
| **3. wss-integration**| `8084` | MQTT / WSS | **TLS** (WSS, AEAD only) | Username / Password (RBAC) | External browser SPAs and third-party web dashboards. |
| **4. ws-console** | `39444` / `8083` | MQTT / WS or WSS | **Plain WS** (Loopback) or **WSS** (LAN/CA) | Ephemeral Session Token (`sessionID`) | Local Bindu Web Console UI & Kiosk/Viewer. |

---

## 3. Web Console & Kiosk Transport Security Architecture

Bindu provides transport isolation for both administrative operations and operator dashboard viewers across four operational configurations:

### 3.1 Transport Modes & CLI Configuration

> [!IMPORTANT]
> **Zero Manual Editing Rule:** Operators must never edit `binduConfig.json` manually on disk. Manual edits bypass schema validation and risk invalid JSON syntax that stops the engine on boot. Always configure transport modes using `bindu-studio system-config console-access` or `bindu-studio system-config set/merge`.

- **1. Native LAN HTTPS (Mode: "auto")**: Recommended for industrial plants, farms, and kiosk displays. Bindu Root automatically discovers active network interfaces, issues a leaf certificate with all SANs (IPs and hostnames) signed by the cluster CA (`runtime/tls/ca.crt`), and serves HTTPS (`38080`) and WSS (`39444`). Clients establish permanent green-lock trust by installing the public CA downloaded directly from `GET /console/ca.crt`.
  - Enable via CLI: `bindu-studio system-config console-access --mode auto --yes`
- **2. Enterprise BYOC (Mode: "custom")**: Allows enterprise infrastructure teams to supply their own corporate certificates (`fullchain.pem` and `privkey.pem`) issued by their Enterprise PKI or commercial wildcard authority.
  - Enable via CLI: `bindu-studio system-config console-access --mode custom --cert-file /path/to/cert.pem --key-file /path/to/key.pem --yes`
- **3. Loopback & SSH Tunneling (Mode: "local" / "disabled")**: Strictly binds listeners to `127.0.0.1`. Remote access is tunneled through an encrypted SSH pipe (`ssh -L 38080:127.0.0.1:38080 -L 39444:127.0.0.1:39444 user@root-ip`).
  - Enable via CLI: `bindu-studio system-config console-access --mode local --yes`
- **4. Reverse Proxy**: Ingress handled by external reverse proxies (Caddy, Nginx, Traefik) terminating TLS.

### 3.2 Ephemeral Session Authentication Guarantee
Regardless of the transport mode used, the web console authenticates to the MQTT broker using the active `sessionID` UUID obtained upon login. **No plaintext user passwords are ever stored in server memory or sent across the WebSocket channel.**

### 3.3 Remote Secure Access via SSH Tunneling (Loopback Mode)
When administrators manage a remote Bindu Root node without exposing LAN ports:

```bash
ssh -L 38080:127.0.0.1:38080 -L 39444:127.0.0.1:39444 user@bindu-root-ip
```

Once established, browse to `http://127.0.0.1:38080/console/`.

For complete kiosk installation scripts (Raspberry Pi, Android tablets, Windows), consult [`how-to-manage-bindu-root-ssh.md`](./how-to-manage-bindu-root-ssh.md).

---

## 4. REST API Defense-in-Depth & Middleware Scoping

To prevent privilege escalation and unauthorized access, REST API endpoints are compartmentalized into three distinct authorization levels:

```
+---------------------------------------------------------------------------------------------------+
| MIDDLEWARE               | VALID CREDENTIALS                    | ACCESSIBLE ROUTES               |
+--------------------------+--------------------------------------+---------------------------------+
| RequireAdminAuth         | Basic Auth (Admin User) OR           | /api/v1/lifecycle/*             |
|                          | Active Admin Session Cookie          | /api/v1/mesh/*                  |
|                          | (X-Enrollment-Token is REJECTED)     | /api/v1/storage/*, /logs, etc.  |
+--------------------------+--------------------------------------+---------------------------------+
| RequireEnrollmentAuth    | X-Enrollment-Token OR                | /api/v1/tls/preflight-node      |
|                          | Admin Basic Auth / Session           | /api/v1/tls/sign-node           |
+--------------------------+--------------------------------------+---------------------------------+
| RequireRenewalAuth       | Active mTLS Client Certificate OR    | /api/v1/tls/renew-node          |
|                          | X-Enrollment-Token OR Admin Auth     |                                 |
+--------------------------+--------------------------------------+---------------------------------+
```

### Key Security Guarantees:
- **Zero Escalation via Enrollment Tokens:** Nodes provisioned with an `X-Enrollment-Token` cannot perform administrative tasks, modify mesh topology, or deploy plugins.
- **CORS Protection:** Cross-Origin Resource Sharing (CORS) is restricted to same-origin and local loopback origins; wildcards (`*`) are disallowed on state-mutating endpoints.

---

## 5. MQTT Topic Isolation & Provenance Anti-Spoofing

### Authhook ACL Rules (`authhook.OnACLCheck`)
1. **Mesh Nodes (mTLS):**
   - **Publishing:** Restricted strictly to `bindu/in` (ingress dispatch) or `bindu/out/<child-id>` (parent forwarding). Nodes cannot publish directly to application topics (e.g., `VideoEvent/cam1`).
   - **Subscribing:** Restricted strictly to `bindu/out/<local-node-id>` (egress queue) and `bindu/in`. Wildcard subscriptions (`#` or `+/+/+`) are strictly rejected.
2. **Application Clients (Username/Password):**
   - Restricted to topics authorized by their assigned role in `accessControlConfig.brc` (`admin` or `application`).
   - Access to internal routing topics (`bindu/*`) is denied.

### Dispatcher Anti-Spoofing (`bindudispatcher.OnPublish`)
The Bindu Dispatcher inspects every message passing through `bindu/in` and guarantees metadata provenance:
- For mTLS mesh clients, `$user` is stamped as `"mesh:<node_id>"` (derived from the TLS peer certificate).
- For authenticated clients, `$user` is stamped as the verified username.
- Any attempt by a client to forge or overwrite `$user` in the payload is discarded and normalized by the engine.

---

## 6. PKI Lifecycle, Licensing & Cryptography

### Licensing Authority Public Key Pinning
- Bindu embeds an authoritative Root Licensing Public Key (`NO2DaqOGmz27xgAgKbz6wvwdn47/J7f7J9D/943R6QU`).
- License tokens and installer tokens are validated against pinned authorities. Tokens signed by unapproved keys are rejected at startup.
- In test/development environments, arbitrary keys can be allowed only via the explicit environment variable `BINDU_DEV_ALLOW_UNPINNED_LICENSE=1`.

### Autonomous mTLS Certificate Renewal
- Node certificates are issued with a 365-day validity period.
- Active nodes monitor certificate expiration in the background. When 30 days of validity remain (or at 80% lifetime), the node initiates renewal via `/api/v1/tls/renew-node`.
- The Root validates the node's existing mTLS certificate and signs the new CSR without requiring manual operator intervention or service restarts.

### Modern AEAD Ciphers
All TLS-enabled listeners and dialers enforce TLS 1.2 minimum and restrict cipher suites to modern AEAD algorithms:
- `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256` / `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384` / `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256` / `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256`

---

## 7. Cryptographic Storage & File System Security

### Encrypted User & Role Storage (`.brc`)
- **Key Derivation (PBKDF2):** `config/accessControlConfig.brc` is encrypted using AES-256-GCM with keys derived via **PBKDF2-HMAC-SHA256** using **100,000 iterations** and a dynamic 16-byte random salt.
- **Format Header (`v2:`):** Files generated by current releases use the `v2:` header prefix with embedded per-file salt.
- **Transparent Migration:** Existing `v1` legacy files are automatically decrypted on load and upgraded to `v2:` upon the next save.

### Encrypted Integration Secrets Vault (`.brs`)
Bindu protects all third-party integration credentials (API keys, S3 credentials, webhook tokens) in a dedicated encrypted vault: `config/accessControlConfig.brs`.
- **Vault Cryptography:** Symmetrically encrypted with AES-256-GCM + PBKDF2 (100,000 iterations). Plaintext tokens never appear in configuration JSON files, environment files, or repository manifests.
- **CLI Management:** Governed exclusively via `bindu-studio secrets` (`set`, `get`, `list`, `delete`, `validate`) with automated masking and interactive prompt support.
- **Cross-Component Consumption:** Secrets stored in `.brs` are resolved dynamically by:
  - **`RESTReactorSet`**: Evaluated dynamically via CEL `secret('KEY_NAME')` in headers or URLs.
  - **`AISemanticReactorSet`**: Resolved via `"api_key_secret": "secret:KEY_NAME"` for TypeSafe or OpenAI LLM providers.
  - **`MediaStorageSet`**: Resolved via `"secret_key": "secret:KEY_NAME"` for S3/MinIO bucket access.

### Zero-Plaintext Wire & Ledger Guarantee (Secret Reference Architecture)
A primary vulnerability in distributed event architectures is credential leakage across message queues and persistent ledgers. In Bindu:
1. **Never on the Wire:** Passwords, API tokens, and access keys MUST NOT travel across the MQTT bus or inter-node mesh replication channels.
2. **Never in SQLite Ledgers:** Messages stored in `v_messages` contain only immutable reference pointers matching the canonical pattern `secret:KEY_NAME` (`^secret:[a-zA-Z0-9_-]+$`).
3. **Ingress Semantic Enforcement:** Any field defined with the semantic type `"secret"` in a MessageType schema strictly rejects plaintext submissions at the perimeter. Incoming messages containing raw tokens are discarded with schema validation errors.
4. **Dynamic In-Memory Resolution:** Credentials are only decrypted into volatile RAM at the moment of execution via `celext.ResolveSecret()`.
5. **Edge Theft Resilience:** If an edge gateway node is physically compromised or its SQLite database copied, the attacker obtains zero third-party API keys or S3 credentials—only non-resolvable reference labels.

### File Permissions
Bindu automatically enforces strict POSIX file permissions:
- Configuration and data directories: `0700` (`rwx------`)
- Certificate keys, credentials, and databases: `0600` (`rw-------`)

---

## 8. BPR Plugin Sandboxing & Process Containment

Plugins running under the **Bindu Plugin Runtime (BPR)** are external OS processes executing untrusted or third-party logic. Bindu isolates them using two containment layers:

### 1. Environment Variable Whitelisting
Bindu scrubs all host secrets before spawning child plugin processes.
- **Allowed System Variables:** `PATH`, `TMPDIR`, `TEMP`, `TMP`, `LANG`, `LC_ALL`, `LC_CTYPE`, `TZ`, `USER`, `HOME`.
- **Allowed BPR Contract Variables:** `BINDU_PLUGIN_TOKEN`, `BINDU_PLUGIN_NAME`, `BINDU_PLUGIN_VERSION`, `BINDU_NODE_ID`, `BINDU_NODE_ROLE`.
- **Blocked / Stripped Variables:** `BINDU_MASTER_KEY`, `BINDU_ENROLLMENT_TOKEN`, `AWS_*`, `MINIO_*`, `DATABASE_*`, `SECRET_*`, and any sensitive host credentials.

### 2. Process Group Isolation & Parent Death Signal (Linux)
- **Process Group (`Setpgid: true`):** Plugins run in their own process group, isolating signal handling.
- **Parent Death Signal (`Pdeathsig: syscall.SIGKILL`):** If the Bindu host daemon crashes or terminates, the Linux kernel automatically delivers `SIGKILL` to all running plugin processes, preventing orphaned or zombie background execution.

---

## 9. Security Summary & Operational Checklist

| Security Guarantee | Mechanism | Verification |
|:---|:---|:---|
| **No Password in Heap** | Ephemeral `sessionID` token auth | Tested in `accesscontrol/contracts_test.go` |
| **No Rogue Licenses** | Authoritative Public Key Pinning | Tested in `internal/licensing/verify_test.go` |
| **No Plugin Secret Theft** | Environment Whitelisting (`buildSanitizedEnv`) | Tested in `executioneng/bpr_session_robustness_test.go` |
| **No Topic Hopping** | MQTT Broker ACL Checks in Authhook | Tested in `authhook/authhook_matrix_test.go` |
| **No REST Escalation** | Scoped `RequireAdminAuth` Middleware | Tested in `bindurestapi/basic_auth_middleware_test.go` |
| **No Certificate Outages** | Autonomous `/api/v1/tls/renew-node` | Tested in `bindurestapi/tls_sign_test.go` |
| **Strong Key Derivation** | PBKDF2 (100,000 iters) with salt | Tested in `accesscontrol/encryptedaccesscontrol_test.go` |
| **Zero Plaintext on Wire/Ledger** | Encrypted vault (`.brs`) + reference pattern (`secret:KEY_NAME`) | Tested in `messagetypes/storage_semantic_test.go` |