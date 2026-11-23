# Bindu Studio - System Configuration User Guide

Role of this document:

- Supporting operator guide for inspecting and editing runtime configuration
- Best used together with the canonical runtime logging guide when changing logging settings

For the canonical runtime logging behavior, see
`public/console/docs/Bindu Runtime Logging Model.md`.

This guide explains how to inspect and update system configuration using Bindu Studio.
It is written for operators and administrators.

---

## Overview

System configuration is the central runtime configuration for `bindu-root`.
It is stored in `config/binduConfig.json` and validated against a schema.

Bindu Studio is fully schema-driven and interactive:

- You never type section or field names.
- Studio discovers sections and fields from the schema.
- When the schema changes, the CLI adapts automatically.
- All changes are validated before saving.

Restart behavior:

- Some changes require restarting `bindu-root`.
- Studio tells you when a restart is needed.
- Restart is done with `bindu-studio reset-bindu`.
- `reset-bindu` uses the restart mechanism configured for the current root.
- In service-managed installs this may use systemd; in other installs it may use a different restart path.

Logging note:

- Bindu runtime logs to `stdout` or `stderr` in normal operation.
- Persistence and retention are handled by Linux `journald`, not by file logging inside Bindu.

---

## Commands at a glance

```
bindu-studio system-config get             Inspect configuration
bindu-studio system-config set             Change one or more fields (interactive)
bindu-studio system-config plan            Preview a change (interactive)
bindu-studio system-config console-access  Configure Console, Kiosk, and Viewer TLS mode
bindu-studio system-config validate        Validate full config
bindu-studio system-config export          Export configuration to file or JSON
bindu-studio system-config import          Import configuration from file
bindu-studio system-config merge           Merge partial configuration
bindu-studio system-config list            List sections and fields
```

---

## system-config get (inspect)

**What it does**

- Lets you pick a section
- Shows a readable tree of fields, descriptions, and current values

**Flow**

1) Pick a section
2) View the section tree

**Example session**

```
SYSTEM CONFIG — SECTIONS

1) control_plane        Control plane settings
2) mqtt                 MQTT broker configuration
3) storage              Persistence and retention
4) tls                  Certificates and TLS
5) logging              Logging configuration
6) smtp                 SMTP server configuration

Select section [1-6]:
> 1

control_plane

- studio                   Bindu Studio REST listener
  - bind                    Bind address for Studio REST
    value: "127.0.0.1"
  - port                    Port for Studio REST
    value: 18080

- console                  Console UI listener
  - bind                    Bind address for Console UI
    value: "127.0.0.1"
  - port                    Port for Console UI
    value: 8080
  - exposure                Exposure policy for Console UI
    value: "local"

- console_ws               Local Console MQTT-over-WebSocket listener
  - bind                    Bind address for Console live WebSocket
    value: "127.0.0.1"
  - port                    Port for Console live WebSocket
    value: 9444
  - exposure                Exposure policy for Console live WebSocket
    value: "local"
```

**Notes**

- Each field shows:
  - the field name
  - a short description
  - the current value
- Missing values are shown as `value: <missing>`.

**JSON output (advanced)**

```
bindu-studio system-config get --json
```

---

## system-config set (edit one or more fields)

**What it does**

- Starts an interactive editing session by default
- Lets you update multiple fields in the same section
- Uses schema-driven prompts and validation
- Writes once when you finish (unless `--dry-run`)
- Also supports a non-interactive single-field mode with `--section`, `--field`, and `--value`

**Editing loop (how it works)**

- Select a field
- Change its value
- Return to the field list
- Repeat as needed
- Select **Finish** to end the session

**Example editing loop**

```
EDIT logging

1) format   string   current: "text"
2) level    string   current: "INFO"
3) output   string   current: "stdout"
0) Finish

Select field [0-3]:
> 2

Field: level
Description: Log verbosity level.
Current value: "INFO"

Allowed values:
1) "DEBUG"
2) "INFO"
3) "WARN"
4) "ERROR"

Select [1-4]:
> 1

EDIT logging

1) format   string   current: "text"
2) level    string   current: "DEBUG"
3) output   string   current: "stdout"
0) Finish

Select field [0-3]:
> 0
```

**Type-specific input examples**

Boolean:

```
New value:
1) true
2) false
Select [1-2]:
> 1
```

Enum (fixed choices):

```
Allowed values:
1) "never"
2) "if-needed"
3) "always"
Select [1-3]:
> 2
```

String / integer:

```
New value (integer):
> 8443
```

List (array):

```
New values (comma-separated):
> a, b, c
```

**Save semantics**

- Changes stay in memory during the session.
- When you select **Finish**:
  - the configuration is validated
  - the configuration is written to disk (unless `--dry-run`)
- If `--dry-run` is used:
  - nothing is written
  - a plan is shown instead

**Validation**

- Invalid values are rejected.
- Studio prints a short error and re-prompts.
- Nothing is written until the value is valid.

**Flags**

- `--dry-run` : show a plan only, do not write
- `--restart never|if-needed|always` : restart policy after saving
- `--auth user:pass` : only used if a restart is requested
- `--section <name> --field <path> --value <raw>` : non-interactive single-field update

**Non-interactive example**

```bash
bindu-studio system-config set \
  --section bindu_server \
  --field contact_email \
  --value bindu-e2e@email.io \
  --yes \
  --restart never
```

---

## system-config plan (preview change)

**What it does**

- Same interactive flow as `set`
- Never writes
- Always prints a plan

**Example output**

```
PLAN

Section: control_plane
Change:
- console.port: 8080 -> 8443

Write: no (plan)
Restart policy: n/a
```

Use this when you want to preview a change safely.

---

## Logging section notes

When editing `logging` in `binduConfig.json`:

- `level` (string): Log verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`; default: `INFO`).
- `format` (string): Output structure (`json` or `text`; default: `json`).
- `retention_days` (integer): Maximum retention duration in days enforced by daily maintenance (default: `7`).
- `max_size_mb` (integer): Maximum total log storage quota in megabytes (default: `500`).
- `output` (string): Output stream (`stdout` or `stderr`; default: `stdout`).

Example configuration:
```json
"logging": {
  "format": "json",
  "level": "INFO",
  "retention_days": 7,
  "max_size_mb": 500
}
```

> [!NOTE]
> **Update Compatibility & Maintenance Enforcement**:
> On existing deployments updated from older releases, if `retention_days` or `max_size_mb` are absent from `runtime/config/binduConfig.json`, Bindu automatically applies production defaults (`7` days and `500` MB) in memory upon loading the configuration.
> During daily scheduled maintenance (and manual triggers), Bindu enforces these quotas using `journalctl --vacuum-*` when running as a systemd service, and via directory pruning for standalone logs under `runtime/logs/`.
> To explicitly persist these values to disk, operators can run:
> `bindu-studio system-config set --section logging --field retention_days --value 7 --yes`
> For operational details, consult `Bindu Runtime Logging Model.md`.

---

## SMTP section notes

When editing `smtp` (SMTP server configuration for email delivery):

- `enabled` (boolean): Enable or disable the email system plugin (`true`/`false`).
- `server` (string): Hostname or IP of the outgoing SMTP server (e.g. `smtp.gmail.com`).
- `port` (integer): Port for connection (e.g. `587` for StartTLS, `465` for SSL, or `1025` for mock Mailpit).
- `username` (string): SMTP username/account email (e.g. `operator@example.com`).
- `password` (string): Password or App Password.
- `from_address` (string): Sender address.
- `tls_policy` (string): TLS policy (`mandatory`, `opportunistic`, or `none`).

To configure the SMTP credentials non-interactively via Studio CLI:
```bash
bindu-studio system-config set --section smtp --field server --value "smtp.gmail.com" --yes
bindu-studio system-config set --section smtp --field port --value 587 --yes
bindu-studio system-config set --section smtp --field enabled --value true --yes
```

---

## system-config validate

**What it does**

- Validates the full `binduConfig.json` against the schema
- Does not change anything

**When to use it**

- After upgrades
- After manual edits
- When troubleshooting

**Example**

```
bindu-studio system-config validate
```

---

## system-config console-access (remote access & TLS wizard)

**What it does**

- Configures Web Console, Dashboard Viewer, and Kiosk access mode
- Automatically synchronizes both HTTP/HTTPS and WebSocket WSS listener endpoints
- Eliminates manual JSON edits to prevent formatting corruption and crashes
- Validates the resulting configuration against `binduConfigurationSchema.json` before saving
- Automatically discovers and displays operational URLs and LAN IP endpoints

**Available Modes**

1. **`auto` (Native LAN HTTPS)**: Binds to `0.0.0.0` with `exposure: "lan"`. Signs a leaf certificate on boot with all discovered network SANs, authenticated by the Cluster Root CA. Kiosks and LAN tablets achieve zero-warning green locks via 1-click CA trust.
2. **`custom` (Enterprise BYOC)**: Binds to `0.0.0.0` with `exposure: "lan"` using customer-supplied TLS certificate and private key.
3. **`local` / `disabled` (Loopback HTTP)**: Binds strictly to `127.0.0.1` with `exposure: "local"`, disabling TLS. Remote access requires an encrypted SSH tunnel.

**Interactive Wizard:**

```bash
bindu-studio system-config console-access
```

**Non-Interactive Examples:**

```bash
# Enable Native LAN HTTPS with cluster CA:
bindu-studio system-config console-access --mode auto --yes

# Enable Enterprise BYOC:
bindu-studio system-config console-access \
  --mode custom \
  --cert-file /etc/ssl/certs/company.pem \
  --key-file /etc/ssl/private/company.key \
  --yes

# Revert to loopback-only (SSH tunnel mode):
bindu-studio system-config console-access --mode local --yes
```

---

## Restart behavior

- If `bindu-root` is not running, changes apply on next start.
- If `bindu-root` is running, some changes require a restart.

Studio will tell you if a restart is needed.
To restart:

```
bindu-studio reset-bindu --auth user:pass
```

---

## Tips for operators

- Use `get` to inspect before making changes.
- Use `plan` to preview changes without writing.
- Use `set` for a single safe change at a time.
- Use `validate` after upgrades or manual edits.
