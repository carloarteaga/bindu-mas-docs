# Bindu Standalone Kiosk Dashboard Viewer — User & Operator Manual

The **Bindu Standalone Kiosk Dashboard Viewer** is an ultra-lightweight, embedded web application designed to present real-time operational dashboards on dedicated workstations, tablets, POS terminals, and wall-mounted screens across edge nodes and the central root gateway.

---

## 🏛️ 1. Overview & Architectural Philosophy

In real-world retail stores, distribution centers, altamar vessels, and manufacturing plants, floor staff and operators do not manage personal software accounts or type expiring corporate passwords every few minutes. Instead, displays operate as **Dedicated Stations (Kiosks)**:

* **Floor Workstation / Tablet**: Positioned in store aisles or picking carts; displays real-time item replenishment, shelf audits, or temperature alerts.
* **Manager Office PC**: Displays store-wide shrinkage metrics, inventory discrepancy audits, and shift throughput.
* **Corporate HQ Video Wall**: Displays aggregated fleet KPIs across 400+ stores.

```mermaid
flowchart TD
    subgraph EdgeStore["🏪 Store 042 (Edge Gateway: bindu-node)"]
        LocalSQLite[("💾 Local SQLite<br/>v_messages")]
        NodeQueryAPI["🔒 Query API<br/>POST /api/v1/messages/sql"]
        LocalSQLite --> NodeQueryAPI

        FloorKiosk["📱 Floor Tablet (Kiosk Mode)<br/>/viewer?id=floor-replenishment&kiosk=true"]
        ManagerPC["💻 Manager PC<br/>/viewer?id=store-audit-summary"]
        
        NodeQueryAPI -->|SELECT only| FloorKiosk
        NodeQueryAPI -->|SELECT only| ManagerPC
    end

    subgraph CloudRoot["☁️ Corporate Central (Root Gateway: bindu-root)"]
        CentralSQLite[("💾 Central SQLite<br/>v_messages (400 store summaries)")]
        RootQueryAPI["🔒 Root Query API<br/>POST /api/v1/messages/sql"]
        CentralSQLite --> RootQueryAPI

        CorpHQ["🖥️ Executive Video Wall<br/>/viewer?id=national-inventory-summary&kiosk=true"]
        RootQueryAPI -->|SELECT only| CorpHQ
    end

    EdgeStore -->|StoreInventorySummary<br/>routing: local, parent| CentralSQLite

    style EdgeStore fill:#f8fafc,stroke:#0284c7,stroke-width:2px;
    style CloudRoot fill:#f8fafc,stroke:#4f46e5,stroke-width:2px;
    style FloorKiosk fill:#dbeafe,stroke:#2563eb,stroke-width:2px;
    style CorpHQ fill:#ede9fe,stroke:#7c3aed,stroke-width:2px;
```

### Key Architectural Invariants
1. **Ultra-lightweight footprint**: The viewer is a self-contained single-page application (~15 KB) requiring zero heavy node dependencies or external daemons.
2. **Dual-Mode availability**: Runs natively on both `bindu-root` and `bindu-node`.
3. **Station-based URLs**: Workstations are configured once via URL parameters (`?id=...`, `?kiosk=true`); no login friction for floor workers.
4. **Strict Read-Only Enforcement**: Queries execute against `/api/v1/messages/sql` and are strictly validated by `ValidateReadOnlyQuery` (any token like `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ATTACH` is blocked).

---

## 🚀 2. Quick Start & Accessing the Viewer

### From the Admin Console
1. Navigate to **📊 Dashboards** in the Console sidebar.
2. In the top toolbar, click **`📺 Kiosk View`**.
3. The viewer opens in a new tab displaying all active dashboards.

### Direct URLs
* **Root Gateway (Cloud / Central)**:
  ```
  http://<root-ip>:8080/viewer
  https://<root-ip>:8443/viewer
  ```
* **Edge Node (Store / Facility)**:
  ```
  https://<node-ip>:<rest_query_port>/viewer
  ```

---

## 🎛️ 3. Direct URL Parameters & Station Filtering

The viewer is controlled declaratively via query string parameters. This allows IT administrators to configure kiosk shortcuts or system boot URLs without modifying source code.

| Parameter | Type | Default | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| **`id`** | `string` | *(all)* | Comma-separated list of specific dashboard IDs to display on this station. | `?id=floor-inventory,misplaced-items` |
| **`tag`** / **`group`** | `string` | *(all)* | Filter dashboards by label or semantic group. | `?tag=floor` or `?group=inventory` |
| **`kiosk`** | `bool` | `false` | When `true` or `1`, hides the top toolbar for borderless wall mounting and full-screen kiosks. | `?kiosk=true` |
| **`refresh`** | `integer` | `10000` | Auto-refresh interval in milliseconds. Set to `0` to pause automatic refresh. | `?refresh=5000` *(5 seconds)* |

---

## 🏪 4. Real-World Deployment Scenarios

### Scenario A: Store Floor Replenishment Tablet (Retail Aisle Kiosk)
Mounted on a stock cart or tablet holder in aisle 3. Shows stock replenishment alerts and misplaced items. Runs 24/7 in fullscreen kiosk mode:

```
https://192.168.1.10:8443/viewer?id=store-floor-replenishment,store-misplaced-items&kiosk=true&refresh=5000
```
* **Behavior**: Only renders the 2 floor widgets, updates every 5 seconds, and hides the top bar so floor staff cannot accidentally leave the view.

### Scenario B: Store Manager Office PC
Accessed from the manager's browser. Displays overall inventory accuracy, shrinkage alerts, and category throughput:

```
https://192.168.1.10:8443/viewer?id=store-inventory-audit,store-shrinkage-alerts&refresh=30000
```
* **Behavior**: Updates every 30 seconds with standard toolbar controls available.

### Scenario C: Corporate Executive Video Wall (Cloud Root)
A 75-inch wall-mounted monitor in regional operations. Displays nationwide inventory summaries across 400 stores:

```
https://root.retailcorp.com:8443/viewer?id=national-fleet-inventory-summary&kiosk=true&refresh=60000
```
* **Behavior**: Borderless display, updates every 60 seconds.

---

## 🔐 5. Security & Station Authentication Model

The viewer uses a multi-tiered security model designed for physical stations:

```mermaid
sequenceDiagram
    autonumber
    participant Kiosk as 📱 Station Browser (/viewer)
    participant Auth as 🛡️ accesscontrol Guard
    participant DB as 💾 SQLite Ledger (v_messages)

    Kiosk->>Auth: POST /api/v1/messages/sql
    alt Has Active Session Cookie or Basic Auth (role: application/admin)
        Auth->>DB: Execute SELECT Query (Validated Read-Only)
        DB-->>Kiosk: 200 OK + Rows JSON
    else No Credentials (First Boot)
        Auth-->>Kiosk: 401 Unauthorized
        Note over Kiosk: Shows Station Access Modal
        Kiosk->>Kiosk: Enter station credentials once (Saved to sessionStorage)
        Kiosk->>Auth: POST /api/v1/messages/sql with Authorization: Basic ...
        Auth->>DB: Execute SELECT Query
        DB-->>Kiosk: 200 OK + Rows JSON
    end
```

### 1. Role-Based Access Control (`role: "application"`)
* Users with role `application` are authorized to execute read-only queries against `/api/v1/messages/sql`.
* When an `application` user logs in through the Console login page (`/console/login.html`), they are **automatically redirected to `/viewer`** instead of the admin console.

### 2. Station Credentials Modal
If the station accesses the node without prior authentication, an in-page modal appears:
> **🔒 Station Access**  
> *This node requires station credentials to display active dashboards.*  
> `Username: [ app ]`  
> `Password: [ ******* ]`  
> `[ Authorize Station ]`

Once authorized, credentials are saved in `sessionStorage` for the browser lifetime.

### 3. URL Embedded Authentication (Auto-Login Kiosks)
For automated boot scripts (e.g. systemd launching Chromium in kiosk mode):
```bash
chromium-browser --kiosk "https://app:StrongPassword@192.168.1.10:8443/viewer?id=floor-inventory&kiosk=true"
```

---

## 📡 6. Edge-First Data Routing Contract (The 400-Store Rule)

To prevent saturating corporate WAN links and central cloud storage:

1. **Granular Telemetry Stays at the Edge (`routing: ["local"]`)**:
   - High-frequency sensor scans (RFID EPC reads, barcode scans, shelf coordinates) are stored **only in the local store node's SQLite database**.
   - These events use `"routing": ["local"]`.
2. **Kinetics Synthesis via CEL Reactor**:
   - A local CEL Reactor evaluates store kinetics in Go memory (e.g. counting total milk cartons, identifying misplaced goods).
   - Periodically, a scheduled reactor emits a **synthesized aggregate** event (`StoreInventorySummary`) with `"routing": ["local", "parent"]`.
3. **Propagating Dashboards Down to Edge Nodes**:
   - The message type `DashboardDefinitionSet` uses `"routing": ["local", "parent", "child"]`.
   - When deploying store-specific dashboards, set `$producer = target_gateway_node_id` (e.g., `store-042`). The mesh router dispatches the definitions directly to the store node over MQTT.

---

## 🖥️ 7. Interactive Controls & Display Features

* **View Selector (`🗂️ Views (X/Y)`)**: Opens the modal to dynamically select active views, filter by functional category, or search dashboards by name/ID.
* **Fullscreen Toggle (`⛶`)**: Enters native HTML5 fullscreen mode.
* **Refresh Selector**: Options for `5s`, `10s`, `30s`, `1m`, or `Paused`.
* **Manual Refresh (`🔄`)**: Triggers immediate query re-evaluation.
* **Tile Expand (`⤢`)**: Expands any tile to span 2 grid columns for wider table inspection or detailed trend analysis.
* **Timestamp Badge (`Updated: HH:MM:SS`)**: Live confirmation of ledger synchronization.
* **Sign Out (`🚪 Logout`)**: Safely revokes the active session cookie (`bindunet_session`) and clears station storage. On root console, redirects to `/console/login.html`; on edge nodes, resets to a clean state prompting for station credentials. In Kiosk mode (`?kiosk=true`), the toolbar and logout button are automatically hidden to prevent accidental sign-out on wall-mounted screens.

---

## 🔧 8. Enabling the Query Listener on Edge Nodes (`bindu-node`)

To allow edge nodes to serve the `/viewer` web interface and `/api/v1/messages/sql`, ensure the node has its `rest_query` channel declared in `mesh.json`:

```json
{
  "id": "store-042",
  "name": "Store 042 Edge Gateway",
  "connectivity_kind": "gateway",
  "bindu": {
    "channels": {
      "rest_query": {
        "bind": "0.0.0.0",
        "port": 8443
      }
    }
  }
}
```

When `rest_query` is configured, `bindu-node` automatically launches the secure TLS query server on startup, serving `/viewer` and `/api/v1/messages/sql`.

---

## 🔐 9. Eliminating the Browser 'Not Secure' Warning (1-Time CA Trust Setup by OS)

When connecting to the Viewer via HTTPS on a local IP or hostname (`https://192.168.x.x:<port>/viewer`), modern web browsers display a security warning (*"Your connection is not private / Not secure zone / NET::ERR_CERT_AUTHORITY_INVALID"*).

### Understanding the Two Security States: Page Bypass vs. Full Padlock (`🔒`)

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

### Downloading the Certificate (`bindu-ca.crt`):
* Click the **`🔒 Root CA`** button in the Viewer top toolbar.
* Or click **`Download CA`** inside the **`📖 Help`** modal.
* Or download directly from any node via `https://<node-or-root-ip>:<port>/api/v1/ca.crt`.

### Step-by-Step Installation by Operating System:

#### A. Microsoft Windows (10 / 11):
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

#### B. Apple macOS:
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

#### C. Linux Desktop (Google Chrome / Chromium / Brave):
Google Chrome and Chromium on Linux use an internal NSS certificate database. Choose either the graphical browser interface or the terminal:

* **Option 1: Chrome / Chromium Browser GUI (Recommended & Quickest)**:
  1. Open Chrome and navigate to: `chrome://certificate-manager/` (or `chrome://settings/certificates`).
  2. Select the **Authorities** tab (*Entidades de certificación*).
  3. Click **Install / Import** (or click the three dots `⋮` at the top right of the Authorities list -> **Import**).
  4. Select your downloaded `bindu-ca.crt` file.
  5. In the trust prompt (*"Edit certificate trust settings"*), check the box:
     - `[x] Trust this certificate for identifying websites` (*"Confiar en este certificado para identificar sitios web"*).
  6. Click **OK**.
  7. Refresh the Viewer page (`F5`) or open a new tab.
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

#### D. Linux Desktop (RHEL / CentOS / Fedora / Rocky):
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

#### E. Apple iPadOS / iOS (Operator Tablets & iPhones):
1. In Mobile Safari, navigate to `https://<root-or-node-ip>:<port>/api/v1/ca.crt`.
2. A prompt appears: *"This website is trying to download a configuration profile. Do you want to allow this?"* -> Tap **Allow**.
3. Open **Settings** -> Tap **Profile Downloaded** (or *General* -> *VPN & Device Management*).
4. Tap **Bindu Cluster CA** -> Tap **Install** (top-right) -> Enter your device passcode -> Tap **Install** to confirm.
5. **Enable Full Trust**:
   - Go to **Settings** -> **General** -> **About** -> **Certificate Trust Settings** (at the bottom).
   - Under *Enable full trust for root certificates*, toggle **Bindu Cluster CA** to **ON** (Green).
   - Tap **Continue** on the warning prompt.
6. Refresh the Viewer: the page loads immediately with full HTTPS green lock trust.

#### F. Android Tablets & Phones (Android 11 / 12 / 13 / 14):
1. In Chrome, download `bindu-ca.crt` from `https://<root-or-node-ip>:<port>/api/v1/ca.crt`.
2. Open **Settings** -> Search for **Install a certificate** (or navigate to *Security* / *Security & Privacy* -> *More security settings* -> *Encryption & credentials*).
3. Tap **Install a certificate** -> Choose **CA certificate**.
4. A prompt will appear (*"Your data won't be private..."*) -> Tap **Install anyway**.
5. Select the downloaded `bindu-ca.crt` file.
6. A confirmation appears: *"CA certificate installed"*. All Viewer dashboards now render with full SSL trust.

---

## ❓ FAQ & Troubleshooting

### Q: The viewer displays "No Dashboards Available"
* **Cause 1**: The requested `?id=...` does not match any deployed `DashboardDefinition`. Check for typos.
* **Cause 2**: The edge node has not received a `DashboardDefinitionSet`. Ensure the dashboard solution has been deployed to the node.

### Q: The viewer tile displays "Zero Records Returned"
* **Cause**: The SQL query executed successfully, but no matching rows exist in `v_messages`. Check the `$type` name or time filter in the dashboard definition's SQL query.

### Q: Can a station user modify database records or delete messages from `/viewer`?
* **No**. The endpoint `/api/v1/messages/sql` strictly invokes `sqlstorage.ValidateReadOnlyQuery()`. Any statement containing keywords such as `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, or `ATTACH` is immediately rejected with `400 Bad Request`.
