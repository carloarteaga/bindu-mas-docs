# Remote Bindu Console & Viewer/Kiosk Access Guide

> [!NOTE]
> **HUMAN OPERATOR / SYSADMIN REFERENCE ONLY**
> This document is intended for human system administrators, plant engineers, and platform infrastructure operators. Autonomous AI Agents building Multi-Agent Systems (MAS) should NOT use this document and must strictly follow the Whitelisted Curriculum defined in [AGENTS.md](./AGENTS.md).

Role of this document:
- Authoritative reference for connecting to a remote Bindu Root Console and Viewer/Kiosk dashboards.
- Explains the four client access architectures: **Native LAN HTTPS (Cluster CA)**, **Enterprise BYOC**, **SSH Tunneling**, and **Reverse Proxy**.

---

## 1. Architectural Overview & Client Access Models

> [!IMPORTANT]
> **CRITICAL ARCHITECTURAL DISTINCTION: Edge Nodes vs. Display Clients**
> - **Edge Nodes (`bindu-node` on Drones, Gateways, Industrial PCs):**
>   - Completely **headless** Go server daemons (zero GUI, zero browsers, NO Chromium).
>   - Communicate strictly via **mTLS on port 8884** (Mesh Protocol).
>   - Never access the HTTP console, never run `curl /console/ca.crt`, and never run `update-ca-certificates`.
>   - Mesh certificates are provisioned and rotated automatically by Bindu. Operators NEVER SSH into 200 drones or edge nodes one-by-one.
> - **Display Clients (Laptops, Tablets, Smart TVs, Dedicated Wall Kiosks):**
>   - External devices running a web browser used by **human operators** to view the Web Console (`/console/`) or dashboards (`/viewer`).
>   - In Corporate deployments (**Enterprise BYOC**), corporate laptops/tablets already trust `*.company.com` via MDM; **zero manual certificate installation is needed on any client**.
>   - In standalone air-gapped plants (**Native LAN HTTPS**), operators can download the cluster CA to trust the IP in their browser, or configure a dedicated hardware screen (e.g. an HDMI TV kiosk).

Bindu supports four distinct access models for human operators and display clients:

```
+---------------------------------------------------------------------------------------------------+
| ACCESS MODEL             | TARGET CLIENTS                       | SECURITY & CERTIFICATE CONTRACT |
+--------------------------+--------------------------------------+---------------------------------+
| 1. Native LAN HTTPS      | Wall-mounted Kiosks, Plant Tablets,  | HTTPS/WSS with Cluster CA.      |
|    (Cluster CA + SANs)   | Smart TVs, Operator LAN browsers     | Green lock via 1-click CA trust.|
+--------------------------+--------------------------------------+---------------------------------+
| 2. Enterprise BYOC       | Corporate Intranets, Cloud VPCs,     | Customer-supplied certificates  |
|    (Own Certs)           | *.empresa.com domains                | (fullchain.pem + privkey.pem).  |
+--------------------------+--------------------------------------+---------------------------------+
| 3. SSH Port Forwarding   | Dev environments, single admin       | Plain HTTP/WS over loopback     |
|    (Loopback Tunnel)     | workstations, strict air-gap         | (127.0.0.1) tunnel. No CA needed|
+--------------------------+--------------------------------------+---------------------------------+
| 4. Reverse Proxy Ingress | Edge gateways with existing Nginx,   | Handled by external ingress     |
|    (Caddy / Nginx)       | Traefik, or corporate load balancer  | terminating TLS.                |
+--------------------------+--------------------------------------+---------------------------------+
```

---

## 2. Model 1: Native LAN HTTPS with Cluster CA (Kiosk & Operator Tablets)

This is the recommended out-of-the-box model for industrial facilities, agricultural plants, retail kiosks, and OT subnets where devices connect directly via private IP (`192.168.x.x` or `10.x.x.x`).

### How It Works:
1. Bindu Root inspects its internal cluster CA (`runtime/tls/ca.crt`).
2. It dynamically discovers local IP interfaces and hostnames, signing a leaf certificate with all discovered Subject Alternative Names (SANs).
3. The Web Console (`38080`) serves HTTPS and the Web Console WebSocket (`39444`) serves WSS.
4. Bindu serves the public CA certificate without authentication at `GET /console/ca.crt`.

### Enabling Native LAN HTTPS via Bindu Studio:

> [!IMPORTANT]
> **Never edit `binduConfig.json` manually on disk.** Manual JSON editing carries risk of formatting errors that cause the coordinator service to fail on startup. Always use `bindu-studio` for schema-validated, atomic configuration updates.

Run the interactive console access wizard:
```bash
bindu-studio system-config console-access
```
Or configure it non-interactively in one command:
```bash
bindu-studio system-config console-access --mode auto --yes
```
Studio automatically:
- Configures `control_plane.console` to `0.0.0.0` with `exposure: "lan"` and `tls.mode: "auto"`.
- Synchronizes `control_plane.console_ws` to `0.0.0.0` with `exposure: "lan"`.
- Validates the configuration against `binduConfigurationSchema.json`.
- Writes atomically to disk and prompts to restart `bindu-root`.

### Client Setup Workflows: Step-by-Step CA Trust Installation by Operating System

To achieve a permanent **green lock / secure connection** and eliminate the *"Your connection is not private / Not secure zone"* warning on all browsers and devices across the LAN:

---

### Understanding Browser Security States: Interstitial Bypass vs. Permanent Green Lock (`🔒`)

When connecting to the Web Console or Viewer directly over HTTPS on an internal IP address (e.g., `https://192.168.1.50:39450/viewer`), modern browsers evaluate two distinct layers of security:

```
+-------------------------------------------------------------------------------------------------------+
| USER ACTION                         | INTERSTITIAL PAGE WARNING         | ADDRESS BAR INDICATOR       |
|                                     | ("Your connection is not private")|                             |
+-------------------------------------+-----------------------------------+-----------------------------+
| 1. First Visit (Default)            | 🔴 BLOCKED                        | 🔴 Not Secure (Red / Warn)  |
|                                     | Full-screen warning shown         |                             |
+-------------------------------------+-----------------------------------+-----------------------------+
| 2. Click "Advanced" -> "Proceed"    | 🟢 BYPASSED                       | ⚠️ (x) Not secure           |
|    (Temporary site exception)       | Dashboard page renders normally   | (Address bar warning stays) |
+-------------------------------------+-----------------------------------+-----------------------------+
| 3. Install Cluster Root CA          | 🟢 ELIMINATED                     | 🔒 Secure (Green Padlock)   |
|    in Browser / OS Authorities      | Page opens instantly              | Full zero-warning trust     |
+-------------------------------------+-----------------------------------+-----------------------------+
```

#### Why does the address bar still show `(x) Not secure` after clicking "Proceed"?
Clicking **"Advanced" -> "Proceed to <IP> (unsafe)"** only creates a *temporary site exception* in browser memory. It tells the browser: *"I am aware the authority is unrecognized, but load the HTML anyway"*.
* **The page loads normally**: Dashboard tiles, metrics, and live charts render cleanly without the blocking screen.
* **Cryptographic trust is NOT established**: The browser address bar will continue to display `(x) Not secure` until the issuing certificate authority (`bindu-ca.crt`) is imported into the device's or browser's **Trusted Certificate Authorities** store.
* **Why Linux Google Chrome / Chromium behaves differently than `curl`**: On Linux desktop workstations, running `sudo update-ca-certificates` registers the CA in the system OpenSSL/GnuTLS store (which command-line tools like `curl`, `wget`, and `openssl` trust immediately). However, **Google Chrome and Chromium maintain an independent internal NSS Certificate Store** (`chrome://certificate-manager/`). To turn `(x) Not secure` into the solid green padlock `🔒`, the certificate must be imported into Chrome's **Authorities** tab or into `$HOME/.pki/nssdb`.

---

#### 1. Download the Cluster Root CA:
Download `bindu-ca.crt` using any of the following methods:
* Click the **`🔒 Root CA`** button in the Viewer toolbar (`/viewer`).
* Click the **`🔒 Download Cluster Root CA`** link on the Console Login page (`/console/login.html`).
* Download directly via browser or curl from any Root or Edge node:
  ```bash
  curl -k -o bindu-ca.crt https://<node-or-root-ip>:<port>/api/v1/ca.crt
  ```
* Or export via Studio CLI on the Root machine:
  ```bash
  bindu-studio tls export-ca --out bindu-ca.crt
  ```

---

#### 2. Install on Client Devices (1-Time Setup):

##### A. Microsoft Windows (10 / 11):
* **Option 1: Graphical Wizard (GUI)**:
  1. Double-click the downloaded `bindu-ca.crt` file.
  2. Click **Install Certificate...**
  3. Select Store Location: **Local Machine** (requires admin rights) or **Current User** -> Click **Next**.
  4. Select **Place all certificates in the following store** -> Click **Browse...**
  5. Select **Trusted Root Certification Authorities** (*Entidades de certificación raíz de confianza*) -> Click **OK**.
  6. Click **Next** -> Click **Finish**. A security prompt will confirm: *"The import was successful."*
  7. Restart your browser (Chrome / Edge / Firefox). The address bar turns into `🔒 Secure`.

* **Option 2: PowerShell (Administrator CLI)**:
  ```powershell
  Import-Certificate -FilePath .\bindu-ca.crt -CertStoreLocation Cert:\LocalMachine\Root
  ```

##### B. Apple macOS:
* **Option 1: Keychain Access (GUI)**:
  1. Double-click `bindu-ca.crt` or open **Keychain Access** (*Acceso a Llaveros*).
  2. Select the **System** (or **login**) keychain from the left sidebar and drag `bindu-ca.crt` into it.
  3. Double-click the imported **Bindu Cluster CA** certificate in the list.
  4. Expand the **Trust** (*Confiar*) section.
  5. Change **When using this certificate** (*Al usar este certificado*) to **Always Trust** (*Confiar siempre*).
  6. Close the dialog and authenticate with your Mac administrator password / Touch ID.
  7. Restart your browser (Safari / Chrome). The address bar turns into `🔒 Secure`.

* **Option 2: Terminal (CLI)**:
  ```bash
  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain bindu-ca.crt
  ```

##### C. Linux Desktop (Google Chrome / Chromium / Brave):
Google Chrome and Chromium on Linux use an internal NSS certificate database. Choose either the graphical browser interface or the terminal:

* **Option 1: Chrome / Chromium Browser GUI (Recommended & Quickest)**:
  1. Open Chrome and navigate to: `chrome://certificate-manager/` (or `chrome://settings/certificates`).
  2. Select the **Authorities** tab (*Entidades de certificación*).
  3. Click **Install / Import** (or click the three dots `⋮` at the top right of the Authorities list -> **Import**).
  4. Select your downloaded `bindu-ca.crt` file.
  5. In the trust prompt (*"Edit certificate trust settings"*), check the box:
     - `[x] Trust this certificate for identifying websites` (*"Confiar en este certificado para identificar sitios web"*).
  6. Click **OK**.
  7. Refresh the Console or Viewer page (`F5`) or open a new tab.
  8. **Result**: The address bar instantly displays the solid secure padlock (`🔒`), completely eliminating `(x) Not secure`.

* **Option 2: Terminal CLI (System Store + Chrome NSS Database)**:
  ```bash
  # 1. Register in OS system store (for curl, python, and system tools)
  sudo cp bindu-ca.crt /usr/local/share/ca-certificates/bindu-ca.crt
  sudo update-ca-certificates

  # 2. Register in Chrome/Chromium NSS database
  sudo apt install -y libnss3-tools
  certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n "Bindu Cluster CA" -i bindu-ca.crt
  ```
  Restart Chrome to apply.

##### D. Linux Desktop (RHEL / CentOS / Fedora / Rocky):
* **Option 1: Chrome GUI**: Same as Option 1 above (`chrome://certificate-manager/` -> Authorities).
* **Option 2: Terminal CLI**:
  ```bash
  # 1. Register in OS system store
  sudo cp bindu-ca.crt /etc/pki/ca-trust/source/anchors/bindu-ca.crt
  sudo update-ca-trust extract

  # 2. Register in Chrome/Chromium NSS database
  sudo dnf install -y nss-tools
  certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n "Bindu Cluster CA" -i bindu-ca.crt
  ```

##### E. Apple iPadOS / iOS (Operator Tablets & iPhones):
1. In Mobile Safari, navigate to `https://<root-or-node-ip>:<port>/api/v1/ca.crt`.
2. A prompt appears: *"This website is trying to download a configuration profile. Do you want to allow this?"* -> Tap **Allow**.
3. Open **Settings** -> Tap **Profile Downloaded** (or *General* -> *VPN & Device Management*).
4. Tap **Bindu Cluster CA** -> Tap **Install** (top-right) -> Enter your device passcode -> Tap **Install** to confirm.
5. **Enable Full Trust**:
   - Go to **Settings** -> **General** -> **About** -> **Certificate Trust Settings** (at the bottom).
   - Under *Enable full trust for root certificates*, toggle **Bindu Cluster CA** to **ON** (Green).
   - Tap **Continue** on the warning prompt.
6. Now both the Console and all Viewer dashboards open with full HTTPS green locks and zero security prompts.

##### F. Android Tablets & Phones (Android 11 / 12 / 13 / 14):
1. In Chrome, download `bindu-ca.crt` from `https://<root-or-node-ip>:<port>/api/v1/ca.crt`.
2. Open **Settings** -> Search for **Install a certificate** (or navigate to *Security* / *Security & Privacy* -> *More security settings* -> *Encryption & credentials*).
3. Tap **Install a certificate** -> Choose **CA certificate**.
4. A prompt will appear (*"Your data won't be private..."*) -> Tap **Install anyway**.
5. Select the downloaded `bindu-ca.crt` file.
6. A confirmation appears: *"CA certificate installed"*. All Viewer and Console dashboards now render with full SSL trust.

##### G. Dedicated External HDMI Display Screen (e.g., Raspberry Pi connected to Factory Wall TV):
If an external mini-PC or Raspberry Pi with a graphical desktop is connected via HDMI to a physical display screen:
```bash
# 1. Download and register the Cluster CA into the display device's local trust store
sudo curl -k -s -o /usr/local/share/ca-certificates/bindu-ca.crt https://<node-or-root-ip>:<port>/api/v1/ca.crt
sudo update-ca-certificates

# 2. Launch Chromium in kiosk mode (Zero warnings, permanent green lock)
chromium-browser \
  --kiosk \
  --noerrdialogs \
  --disable-infobars \
  https://192.168.1.50:39450/viewer/?dashboard=potato-intake-and-optical-monitor
```

---

## 3. Model 2: Enterprise BYOC (Bring Your Own Certificate)

For enterprises with their own Corporate Public Key Infrastructure (Active Directory Certificate Services, FreeIPA) or commercial wildcard certificates (`*.plant.company.com`):

Enable BYOC directly using `bindu-studio` (never edit JSON manually):
```bash
bindu-studio system-config console-access \
  --mode custom \
  --cert-file /etc/ssl/certs/company_fullchain.pem \
  --key-file /etc/ssl/private/company_privkey.pem \
  --yes
```
Studio validates the paths, atomically configures `console` and `console_ws`, and prompts for service restart.
Bindu loads the certificates directly and serves the console and WebSockets under corporate root trust.

---

## 4. Model 3: SSH Port Forwarding (Loopback Tunnel)

This model preserves the zero-configuration loopback workflow, ideal for developers or single-administrator maintenance without modifying network policies.

To ensure the server is configured for local loopback:
```bash
bindu-studio system-config console-access --mode local --yes
```

### Default Ports:
- **Console UI:** `38080` (or `8080`)
- **Console WebSocket:** `39444` (or `9444`)

### Steps:

#### 1. From the admin machine, open the SSH tunnel:
```bash
ssh -N \
  -L 38080:127.0.0.1:38080 \
  -L 39444:127.0.0.1:39444 \
  operator@192.168.1.50
```
Keep this terminal open while using the console.

Why both forwards are needed in this workflow:
- `38080` serves the admin Console UI
- `39444` carries the local Console MQTT-over-WebSocket channel used by live Console data

#### 2. Open the Console in the local browser:
```text
http://localhost:38080/console/
```

### Verification:
```bash
curl -v http://127.0.0.1:39444/
```
`HTTP 400 Bad Request` is normal for a plain HTTP probe to a WebSocket endpoint and confirms that the listener is alive.

---

## 5. Model 4: Production Reverse Proxy (Caddy / Nginx)

In facilities where an existing reverse proxy handles network ingress:

### Caddy Example (`Caddyfile`):
```caddy
bindu.local {
    reverse_proxy 127.0.0.1:38080
    reverse_proxy /ws 127.0.0.1:39444
    tls internal
}
```

---

## 6. Remote Debugging & Service Inspection over SSH

Bindu runtime logs normally go to `stdout` / `stderr` and are persisted by
Linux `journald` when the service runs under `systemd`.

### Root service status

```bash
ssh operator@192.168.1.50 'systemctl status bindu-root --no-pager'
```

### Follow root logs live

```bash
ssh operator@192.168.1.50 'journalctl -u bindu-root -f'
```

### Root logs from the last 2 hours

```bash
ssh operator@192.168.1.50 'journalctl -u bindu-root --since "2 hours ago" --no-pager'
```

### Root logs from the last 5 minutes

```bash
ssh operator@192.168.1.50 'journalctl -u bindu-root --since "5 minutes ago" --no-pager'
```

### Root logs for an exact time window

```bash
ssh operator@192.168.1.50 'journalctl -u bindu-root --since "2026-03-13 10:00:00" --until "2026-03-13 10:15:00" --no-pager'
```

### Node equivalents

On a remote node, use the same pattern with `bindu-node`:

```bash
ssh <node-host> 'systemctl status bindu-node --no-pager'
ssh <node-host> 'journalctl -u bindu-node -f'
ssh <node-host> 'journalctl -u bindu-node --since "2 hours ago" --no-pager'
ssh <node-host> 'journalctl -u bindu-node --since "5 minutes ago" --no-pager'
```

### Debugging workflow

For recent incidents, a good sequence is:

1. check `systemctl status`
2. inspect the last 5 minutes
3. widen to the last 2 hours if needed
4. use `-f` only while reproducing live behavior

For the canonical runtime logging model and retention guidance, see
`public/console/docs/Bindu Runtime Logging Model.md`.
