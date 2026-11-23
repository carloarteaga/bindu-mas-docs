# Scheduled System Email BPR & Mesh Status Audit Tutorial ⏰📧

This tutorial guides you through setting up an autonomous EPN (Event Processing Network) that periodically audits the health status of all nodes on your mesh, compiles the metrics, resolves out-of-band attachments via S3, and sends a formatted email report using the **System Email BPR Plugin**.

---

## 1. Setup S3 Media Storage (MinIO)
Run MinIO locally to host attachments.

```bash
export MINIO_ROOT_USER="bindu-core-agent"
export MINIO_ROOT_PASSWORD="bindu-media-123"

# Create storage path
mkdir -p ~/bindu-media-test

# Start MinIO
minio server ~/bindu-media-test --address ":9000" --console-address ":9001"
```

1. Navigate to the MinIO Console at `http://localhost:9001`, log in, and create a bucket named **`raw`**.
2. First, save your MinIO credential in the encrypted vault:
   ```bash
   go run ./cmd/bindu-studio access-control secrets set minio_media_secret --value "bindu-media-123" --yes
   ```
   Then open the **Bindu Console UI** and register the provider under **📁 Media Storage**:
   * **Provider ID:** `minio_local`
   * **Endpoint URL:** `http://localhost:9000`
   * **Access Key:** `bindu-core-agent`
   * **Secret Key:** `secret:minio_media_secret`
3. Click **Apply & Broadcast Configuration**.

---

## 2. Configure SMTP in Bindu
The core root node loads SMTP configurations dynamically from `config/binduConfig.json`. To prevent plaintext credentials from being committed to Git or exposed across the mesh, store your password in the encrypted access control vault:

```bash
# Provision SMTP Password in the encrypted vault
go run ./cmd/bindu-studio access-control secrets set SMTP_PASSWORD --value "<YOUR_APP_PASSWORD>" --yes
```

Open `config/binduConfig.json` and configure your SMTP settings inside the `smtp` block referencing the secret:

```json
{
  "smtp": {
    "enabled": true,
    "server": "smtp.gmail.com",
    "port": 587,
    "username": "alerts@example.com",
    "password": "secret:SMTP_PASSWORD",
    "from_address": "alerts@example.com",
    "tls_policy": "mandatory"
  }
}
```
*(The native `bindu-email-plugin` automatically decrypts `secret:SMTP_PASSWORD` directly from `accessControlConfig.brs` in memory).*

---

## 3. Upload Attachment via Bindu REST API
We will use a Go script that requests a secure presigned upload URL from the Bindu REST API (`/api/v1/media/upload-url`) and PUTs our architectural diagram image `Bindu-AUM_AIM-HIM-KLIM.png` into the `raw` bucket.

The Go code is saved as `upload_image.go` in the tutorial folder. Run the upload command:

```bash
go run public/console/docs/system-email-bpr-tutorial/upload_image.go
```

Once successful, the image is accessible in out-of-band storage via the URI:
`minio_local://raw/Bindu-AUM_AIM-HIM-KLIM.png`

---

## 4. Deploy the Scheduled CEL Reactor (`CELReactorSet`)

The scheduled CEL Reactor wakes up periodically (configured via `cron` at `@every 1m` for demonstration) and queries the database for the latest `MessageState` row from each node and root. 

### 4.1 SQLite Context Query
To avoid formatting constraints in CEL, the SQLite query formats the summary lines for each node directly inside SQL using `printf` and aggregates them using SQLite's `group_concat`:

```sql
SELECT
  (SELECT 'ROOT NODE STATUS:\n' || "$producer" || ' (Role: ' || json_extract(raw_json, '$.fields.role') || '): CPU=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.cpu_usage_pct') AS REAL)) || '%, Mem=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.mem_used_pct') AS REAL)) || '%, Disk=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.disk_used_pct') AS REAL)) || '%, MsgCounts=' || COALESCE(json_extract(raw_json, '$.fields.message_counts_by_type'), '{}') FROM v_messages WHERE "$type" = 'MessageState' AND json_extract(raw_json, '$.fields.role') = 'root' ORDER BY "$when" DESC LIMIT 1) as root_summary,
  (SELECT group_concat(summary, char(10)) FROM (SELECT "$producer" || ' (' || json_extract(raw_json, '$.fields.role') || '): CPU=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.cpu_usage_pct') AS REAL)) || '%, Mem=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.mem_used_pct') AS REAL)) || '%, Disk=' || printf("%.2f", CAST(json_extract(raw_json, '$.fields.disk_used_pct') AS REAL)) || '%, MsgCounts=' || COALESCE(json_extract(raw_json, '$.fields.message_counts_by_type'), '{}') as summary FROM v_messages WHERE "$type" = 'MessageState' GROUP BY "$producer" HAVING MAX("$when") ORDER BY "$producer" ASC)) as mesh_summary
```

### 4.2 CEL Reactor JSON Payload
Use the following `CELReactorSet` configuration payload. Save it to a file or paste it directly in the console UI:

```json
{
  "$type": "CELReactorSet",
  "$node-dest": "root",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-08-14T11:00:00Z",
  "fields": {
    "gateway_node_id": "root",
    "reactors": [
      {
        "reactor_id": "mesh-health-email-reporter",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "SendEmailRequest",
        "context_query": "SELECT (SELECT 'ROOT NODE STATUS:\\n' || \"$producer\" || ' (Role: ' || json_extract(raw_json, '$.fields.role') || '): CPU=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.cpu_usage_pct') AS REAL)) || '%%, Mem=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.mem_used_pct') AS REAL)) || '%%, Disk=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.disk_used_pct') AS REAL)) || '%%, MsgCounts=' || COALESCE(json_extract(raw_json, '$.fields.message_counts_by_type'), '{}') FROM v_messages WHERE \"$type\" = 'MessageState' AND json_extract(raw_json, '$.fields.role') = 'root' ORDER BY \"$when\" DESC LIMIT 1) as root_summary, (SELECT group_concat(summary, char(10)) FROM (SELECT \"$producer\" || ' (' || json_extract(raw_json, '$.fields.role') || '): CPU=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.cpu_usage_pct') AS REAL)) || '%%, Mem=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.mem_used_pct') AS REAL)) || '%%, Disk=' || printf('%.2f', CAST(json_extract(raw_json, '$.fields.disk_used_pct') AS REAL)) || '%%, MsgCounts=' || COALESCE(json_extract(raw_json, '$.fields.message_counts_by_type'), '{}') as summary FROM v_messages WHERE \"$type\" = 'MessageState' GROUP BY \"$producer\" HAVING MAX(\"$when\") ORDER BY \"$producer\" ASC)) as mesh_summary",
        "trigger_condition": "has(db.rows) && size(db.rows) > 0 && db.rows[0].fields.root_summary != null",
        "transformation_expression": {
          "to": "'alerts@example.com'",
          "subject": "'Report Mesh State (' + string(head['$when']) + ')'",
          "body_text": "'ROOT NODE HEALTH REPORT:\n\n' + string(db.rows[0].fields.root_summary) + '\n\n---------------------------------\n\nALL REGISTERED NODES:\n\n' + string(db.rows[0].fields.mesh_summary)",
          "body_html": "'<h3>ROOT NODE HEALTH REPORT</h3><pre>' + string(db.rows[0].fields.root_summary) + '</pre><br/><h3>ALL REGISTERED NODES</h3><pre>' + string(db.rows[0].fields.mesh_summary) + '</pre>'",
          "attachments": "['minio_local://raw/Bindu-AUM_AIM-HIM-KLIM.png']"
        }
      }
    ]
  }
}
```

> [!TIP]
> You can also optionally specify `"cc"` and `"bcc"` recipients in the CEL `transformation_expression`:
> ```json
> "cc": "['ops-team@example.com', 'manager@example.com']",
> "bcc": "['audit-log@example.com']"
> ```

### 4.3 Deployment
Deploy the CEL Reactor using the Bindu CLI tool:
```bash
go run ./cmd/bindu-studio cel-reactor merge \
  --file ./public/console/docs/system-email-bpr-tutorial/poc_cel_reactor_config.json \
  --yes \
  --auth console:StrongPass123
```

---

## 5. Verify Executions
1. Check that the scheduled CEL Reactor fires every minute:
   * Open the **Console UI** and navigate to the **Messages** panel.
   * Verify that a `SendEmailRequest` message is published to the bus.
2. Monitor the SMTP logs:
   * The `bindu-email-plugin` will intercept the request, fetch the presigned URL from `/api/v1/media/resolve`, download `Bindu-AUM_AIM-HIM-KLIM.png` from MinIO, and dispatch the email.
3. Check the transaction status:
   * Verify that a `SendEmailResult` event is published back to the bus with `success = true`.
   * Open your mock SMTP mailbox (e.g. Mailpit at `http://localhost:8025`) and confirm receipt of the email with the subject `"Report Mesh State..."` and the diagram PNG attached correctly!
