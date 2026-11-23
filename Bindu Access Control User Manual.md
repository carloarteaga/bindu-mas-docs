# Bindu Access Control User Manual (Bindu Studio)

This guide explains how to initialize and manage access control using Bindu Studio.
It is intended for operators and administrators.

---

## Overview

Access Control in Bindu defines who can publish or subscribe to application messages.
It does not manage system messages; system messages are internal and not user-managed.

Roles are fixed and exclusive:

- There are only two roles: `admin` and `application`.
- Each user has exactly one role.
- `admin` has full privileges; `application` is data-plane only.

---

## Where the Config Lives & Encryption Architecture

Default encrypted config path:

```
config/accessControlConfig.brc
```

### Encryption & Storage Architecture
- **Encryption Standard:** The file is encrypted at rest using **AES-256-GCM**.
- **Key Derivation (PBKDF2):** Encryption keys are derived using **PBKDF2-HMAC-SHA256** with **100,000 iterations** and a dynamic 16-byte random salt.
- **Format Header (`v2:`):** Configurations saved by current versions are prefixed with `v2:` and store the per-file random salt in the header.
- **Automatic Migration:** Legacy `v1` encrypted files are seamlessly decrypted on startup and automatically upgraded to `v2:` on the next save.
- **File Permissions:** Strictly managed with POSIX `0600` (`rw-------`).

Bindu-root loads it at startup. If Bindu-root is running, changes require a restart to take effect.

---

## Session Security & Ephemeral Token Authentication

When users authenticate via the Web Console or REST API:
- **Zero In-Memory Password Storage:** Bindu does **not** store plaintext user passwords in server RAM.
- **Ephemeral Session Tokens:** Console WebSocket connections to MQTT authenticate using an ephemeral `sessionID` UUID token. The MQTT Authhook directly validates the active session token without ever requiring or transmitting the user's real password across WebSocket channels.

---

## One-Time Initialization

### Command

```
bindu-studio access-control init
```

### Example Interactive Session

```
Admin username [console]
> console
Admin password:
Confirm password:
Created: config/accessControlConfig.brc
```

### What Gets Created

- An initial admin user you define during init.
- Admin and application roles with permissions derived from the schema.

### If You Run Init Twice

You will see an error:

```
already initialized
```

Initialization only runs once. If the file already exists, it will not be overwritten.

---

## Viewing the Configuration

### Command

```
bindu-studio access-control show
```

### Output

- Users table: `USERNAME | ROLE`
- Roles table: `ROLE | PERMISSIONS`
- Permissions are shown as `action:category` (e.g., `publish:application`)

The internal system user is not shown.

---

## Validating the Configuration

### Command

```
bindu-studio access-control validate
```

### What It Checks

- Schema validity
- Structure and required fields
- Role constraints (admin/application only, one role per user)

If validation fails, it prints a list of errors.

---

## User Management Workflows

### Add User

**What it does**
Adds a new user with a password and exactly one role.

**When to use it**
When onboarding a new operator or service account.

**Command**

```
bindu-studio access-control add-user
```

**Example Interactive Flow**

```
Username
> alice
Password policy: min 10 chars, at least 1 uppercase, 1 lowercase, and 1 number.
Password:
Confirm password:
Select role for user:
  1) admin
  2) application
Select number
> 2
User added: alice
Role: application
```

**Notes**
- If the username already exists, you will be asked to confirm overwrite.
- Each user must have exactly one role (admin or application).

---

### Delete User

**What it does**
Deletes an existing user from access control.

**When to use it**
When removing access for a user or service account.

**Command**

```
bindu-studio access-control delete-user
```

**Example Interactive Flow**

```
Select user to delete:
  1) alice
  2) admin
Select number
> 1
Delete user alice? (y/N)
> y
User deleted: alice
```

**Notes**
- You cannot delete the last remaining user.
- Confirmation is always required.

---

### Set Password

**What it does**
Updates the password for an existing user.

**When to use it**
When rotating credentials or responding to security policy changes.

**Command**

```
bindu-studio access-control set-password
```

**Example Interactive Flow**

```
Select user:
  1) admin
  2) alice
Select number
> 1
Password policy: min 10 chars, at least 1 uppercase, 1 lowercase, and 1 number.
Password:
Confirm password:
Password updated for user: admin
```

---

## Password Policy

Passwords must satisfy all of the following:

- Minimum length: 10 characters
- At least 1 uppercase letter (`A-Z`)
- At least 1 lowercase letter (`a-z`)
- At least 1 number (`0-9`)

**Valid examples**
- `HelloWorld1`
- `BinduSecure9`

**Invalid examples**
- `short1` (too short)
- `helloworld1` (missing uppercase)
- `HELLOWORLD1` (missing lowercase)
- `HelloWorld` (missing number)

---

## Integration Secrets & Third-Party API Keys (`bindu-studio secrets`)

Bindu provides centralized, encrypted secret management in `accessControlConfig.brs` (AES-256-GCM + PBKDF2) without requiring external vaults or cleartext environment variables on edge hardware.

These secrets are consumed by:
- **`RESTReactorSet`**: evaluated dynamically in CEL expressions via `secret('KEY')` in `request.headers` or `request.url`.
- **`AISemanticReactorSet`**: resolved via `"api_key_secret": "secret:KEY"` for TypeSafe or OpenAI LLM providers.
- **`MediaStorageSet`**: referenced via `"access_key": "secret:KEY"` and `"secret_key": "secret:KEY"` for S3/MinIO.
- **Message Types with `"type": "secret"`**: any field declared as `"type": "secret"` in a Message Type schema enforces that incoming messages supply a reference matching `^secret:[a-zA-Z0-9_-]+$`. Plain-text credentials (e.g. passwords or API keys) are strictly rejected upon message ingress, preventing accidental credential leaks across the MQTT bus and cluster ledger.

### Commands Overview

```bash
# Set an integration secret interactively (prompts with masked input)
bindu-studio secrets set TYPESAFE_API_KEY

# Set an integration secret non-interactively
bindu-studio secrets set TYPESAFE_API_KEY --value "ts_live_abcdef123" --yes
echo "ts_live_abcdef123" | bindu-studio secrets set TYPESAFE_API_KEY --stdin --yes

# List configured secrets (displays names, length, and sha256 fingerprint, masked by default)
bindu-studio secrets list
bindu-studio secrets list --json

# Get a specific secret (masked by default, use --reveal to show plaintext)
bindu-studio secrets get TYPESAFE_API_KEY
bindu-studio secrets get TYPESAFE_API_KEY --reveal

# Delete a secret
bindu-studio secrets delete TYPESAFE_API_KEY --yes

# Validate active secrets configuration
bindu-studio secrets validate
```

> [!NOTE]
> **Command Alias**:
> The command `bindu-studio secrets` can also be invoked as `bindu-studio integration-passwords` or as a subcommand `bindu-studio access-control secrets`.

---

## When Changes Take Effect

- **If bindu-root is not running:** changes apply on next start.
- **If bindu-root is running:** restart is required.

To restart bindu-root:

```
bindu-studio reset-bindu --auth user:pass
```

---

## Troubleshooting

**access-control not initialized**
- Run: `bindu-studio access-control init`

**already initialized**
- The encrypted config already exists; init will not overwrite it.

**password does not meet policy**
- Ensure it meets length, uppercase, lowercase, and number requirements.

**cannot delete last user**
- At least one user must remain. Create another user before deleting the last one.

**changes not taking effect**
- If bindu-root is running, restart it:

```
bindu-studio reset-bindu --auth user:pass
```

---

## Quick Reference

```
bindu-studio access-control init         Initialize encrypted access control
bindu-studio access-control show         Show users and roles (ASCII tables)
bindu-studio access-control validate     Validate access control config
bindu-studio access-control add-user     Add a new user (interactive)
bindu-studio access-control delete-user  Delete a user (interactive)
bindu-studio access-control set-password Set a user password (interactive)
```
