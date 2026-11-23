# System BPR Plugins: Email & System Monitoring User Guide 📧💻

Bindu isolates heavyweight operations, external networking, and monitoring telemetry from the core Go routines using **System BPR (Binary Process Reactor) Plugins**. These run as secure, sandboxed OS processes communicating with the core daemon via stdin/stdout messaging.

---

## 1. System BPR Overview

### 1.1 Automatic Provisioning
Unlike custom user BPR plugins which require manual configuration in `mesh.json`, System BPRs are **self-provisioning**. 
* During compilation and packaging, System BPR binaries are placed in `plugins.d/bpr/`.
* When the core Bindu daemon starts, it automatically detects their presence on disk and launches them.
* **Email BPR (`bindu-email-plugin`)** is run exclusively on the `root` gateway node.
* **System Monitor BPR (`bindu-sysmon-plugin`)** is run across all nodes.

### 1.2 Zero Direct Database Access Boundary
To maintain thread safety and prevent SQLite file locking errors:
* BPR processes do **not** link database drivers or open BadgerDB/SQLite files directly.
* Telemetry, calculations, and updates are computed by the BPR and emitted as event messages to stdout.
* The core daemon intercepts these events and writes them safely in a single-threaded database transaction handler.

---

## 2. System Email BPR (`bindu-email-plugin`)

The Email BPR listens for `SendEmailRequest` messages, automatically downloads presigned S3 media storage attachments, wraps the email layout, and dispatches it via SMTP.

### 2.1 Message Contract
#### `SendEmailRequest` (Ingress Command)
* **To**: Recipient email address (`string`, required)
* **Cc**: Carbon copy recipient email addresses (`array of strings` or comma/semicolon-separated string, optional)
* **Bcc**: Blind carbon copy recipient email addresses (`array of strings` or comma/semicolon-separated string, optional)
* **Subject**: Subject line header (`string`, required)
* **Body Text**: Plain text body message (`string`, required)
* **Body HTML**: Optional HTML-formatted body (`string`, optional)
* **Attachments**: List of out-of-band S3 media URIs (`array of strings`, optional). Example: `["minio_local://raw/report.png"]`.

##### Example Payload with CC and BCC:
```json
{
  "$id": "email-req-1731234567-001",
  "$type": "SendEmailRequest",
  "$group": "system",
  "$producer": "mesh-reporter",
  "$user": "system-scheduler",
  "$when": "2026-11-10T10:00:00Z",
  "fields": {
    "to": "operations@company.com",
    "cc": ["lead-architect@company.com", "oncall-mgr@company.com"],
    "bcc": ["compliance-audit@company.com"],
    "subject": "CRITICAL: Tank 04 Temperature Threshold Exceeded",
    "body_text": "Sensor TK-04 reported 88.5 C at 10:00:00Z. Emergency cooling activated.",
    "body_html": "<h3>Temperature Threshold Exceeded</h3><p>Sensor <b>TK-04</b> reported <b>88.5 &deg;C</b>. Emergency cooling activated.</p>",
    "attachments": ["minio_local://raw/telemetry_tank04_1000.png"]
  }
}
```

#### `SendEmailResult` (Egress Telemetry Event)
* **Request ID**: Original `$id` of the command message (`string`, required)
* **Success**: Boolean status flag (`boolean`, required)
* **Error Message**: Diagnostic details on failure (`string`, optional)
* **Sent At**: ISO timestamp of the SMTP transaction (`string`, required)

---

### 2.2 S3 Media Resolution & Attachment Presigning
When a `SendEmailRequest` contains an entry in `attachments` such as `minio_local://raw/dashboard.png`:
1. The `bindu-email-plugin` queries the local Bindu REST API endpoint `/api/v1/media/resolve?provider=minio_local&bucket=raw&key=dashboard.png`.
2. The core API returns a secure presigned S3 download URL.
3. The plugin downloads the binary payload in memory and attaches it as a Base64-encoded MIME block to the outgoing email message.

### 2.3 SMTP Configuration in binduConfig.json
The `bindu-email-plugin` reads SMTP server credentials and endpoints dynamically from the `smtp` block in the system configuration file `config/binduConfig.json`.

Example configuration block:
```json
"smtp": {
  "enabled": true,
  "server": "smtp.gmail.com",
  "port": 587,
  "username": "alerts@example.com",
  "password": "your_app_password",
  "from_address": "alerts@example.com",
  "tls_policy": "mandatory"
}
```

To configure these keys using Bindu Studio:
```bash
bindu-studio system-config set --section smtp --field server --value "smtp.gmail.com" --yes
bindu-studio system-config set --section smtp --field port --value 587 --yes
bindu-studio system-config set --section smtp --field enabled --value true --yes
bindu-studio reset-bindu --auth console:StrongPass123 --yes
```

### 2.4 Error Events & Triage
If an error occurs during SMTP startup or transmission, the plugin emits a structured BPR error envelope that Bindu converts to an `ErrorEvent` in the system log.

| Error Code | Severity | Cause | Remediation |
| :--- | :--- | :--- | :--- |
| `SMTP_CONFIG_READ_FAILED` | `error` | `binduConfig.json` could not be read (e.g., missing file or permission issues). | Ensure the config file is present in `config/` and has correct read permissions. |
| `SMTP_CONFIG_PARSE_FAILED` | `error` | `binduConfig.json` contains invalid JSON syntax. | Run `bindu-studio system-config validate` to check configuration integrity. |
| `SMTP_CONFIG_MISSING` | `error` | The `smtp` configuration block does not exist. | Add the block by executing `bindu-studio system-config set --section smtp`. |
| `SMTP_CONFIG_INVALID` | `error` | Missing required SMTP parameters (`server`, `port` or `from_address`). | Check variables; server/sender cannot be blank, port must be positive. |
| `SMTP_DISABLED` | `error` | `SendEmailRequest` triggered but `smtp.enabled` is set to `false`. | Enable using `bindu-studio system-config set --section smtp --field enabled --value true`. |
| `ENV_API_URL_MISSING` | `error` | `BINDU_QUERY_API_URL` environment variable is not defined. | Check system daemon environment injection logs. |
| `SMTP_ATTACHMENT_DOWNLOAD_FAILED` | `error` | Failed to download a requested attachment from local S3/MinIO bucket. | Confirm S3 bucket connectivity and key existence, verify resolved presigned URL. |
| `SMTP_SEND_FAILED` | `error` | SMTP server rejected authentication or connection (bad password, port blocks). | Verify SMTP credentials, application password, port (587 vs 465), and `tls_policy`. |

---

## 3. System Monitor BPR (`bindu-sysmon-plugin`)

The Sysmon BPR monitors node health and runs periodically on a cron schedule (`system.version.tick`) to audit executable hashes and report system updates.

### 3.1 Operations
* **Periodic Tick**: Triggers hourly (or custom configured schedules) to inspect all executable binaries under `tools/node/`, calculates SHA-256 checksums, and emits the `SystemVersion` payload.
* **Update Reconciliation**: Monitors `last-nodeupdate.json` and transitions the node rollout status from `"running"` to `"ok"` once version activation is confirmed on disk.
* **Platform Discovery**: Consumes general `MessageState` telemetry and emits a compact `NodePlatformUpdateEvent` message.

### 3.2 Error Events & Triage
If node audit checks or version tracking operations fail, the plugin emits a structured BPR error envelope that Bindu converts to an `ErrorEvent` in the system log.

| Error Code | Severity | Cause | Remediation |
| :--- | :--- | :--- | :--- |
| `CONFIG_VERSION_CORRUPT` | `warn` | `config/VERSION` exists but contains invalid data (non-integer characters). | Inspect the `/config/VERSION` text file. It should contain only a positive integer (e.g. `12`). |
| `NODE_TOOLS_DIR_READ_FAILED` | `error` | The directory `/tools/node/` is unreadable or has access permission errors. | Audit folder permissions on `<product_dir>/tools/node`. Ensure the Bindu daemon has read access. |

---

## 4. Next Steps & Tutorials
To see the system email BPR in action inside an automated EPN (Event Processing Network), proceed to the [Scheduled System Email BPR Tutorial](./system-email-bpr-tutorial/tutorial_guide.md).
