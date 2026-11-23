# MAS Complete Lifecycle BPR Plugin (`bindu-mascl-plugin`) 🦀⚡

This plugin is a native Go BPR (Bindu Plugin Runtime) worker participating in the **MAS Complete Lifecycle Tutorial**.

## Contract
- **ID**: `bindu-mascl-plugin`
- **Version**: `1.0.0`
- **Runtime**: `bpr` (External OS process communicating via standard `stdin/stdout` JSON protocol)
- **Inputs**: `MASclTrigger`
- **Outputs**: `MASclOcurrence`
- **Schedules**: `0 */15 * * * *` (`cron`)

## Certification & Build
To certify and build the multi-target distribution bundle:
```bash
bindu-studio plugin certify \
  --source ./public/console/docs/mas-complete-lifecycle-tutorial/mascl-bpr-plugin \
  --out ./public/console/docs/mas-complete-lifecycle-tutorial/mascl-bpr-plugin-1.0.0.tar.gz \
  --targets-from-mesh
```
