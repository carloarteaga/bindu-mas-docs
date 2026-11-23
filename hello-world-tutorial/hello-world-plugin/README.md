# hello-world-plugin

This is the curated Hello World plugin source used by the tutorial.

Behavior:
- emits `HelloWorld` every 5 minutes (`trigger=cron`)
- emits `HelloWorldStats` every 15 minutes (`trigger=cron.2`)
- consumes `HelloWorldConfig`
- bootstraps the current greeting from the latest persisted config
- computes stats from read-only SQL against `v_messages`

Run from this directory:

```bash
bindu-studio plugin doctor --source . --pre-release --strict
bindu-studio plugin certify --source . --out ./hello-world-plugin.tar.gz --json
```

Notes:
- This plugin imports `bindunet/sdk` from the surrounding Bindu module.
- In the repo and in installed releases, `go.mod` points to that module with `replace bindunet => ../../../../..`.
- `plugin.json` and `Handshake()` in `main.go` must stay aligned.
- Queries are read-only and scoped to `v_messages`.
- This source is intended to be a working tutorial plugin, not a raw scaffold.
