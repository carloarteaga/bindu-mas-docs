# Built-in System Watchdog CEL Reactor Tutorial 🛡️⚡

This tutorial guides you through understanding, simulating, and validating the built-in **System Watchdog CEL Reactor** (`system_hardware_monitor`).

Unlike custom user-defined reactors, this reactor is hardcoded in the Bindu Go core. It is an advanced example of a **unified Event Processing Agent (EPA)** that listens to the periodic `MessageState` telemetry source, applies rule validation, and dynamically maps exceptions to standard `ErrorEvent` alerts.

---

## 1. Didactic Architecture: Unified Telemetry Watchdog

In Bindu, the watchdog reactor listens to a single unified heartbeat message type (`MessageState`). When a message of this type arrives, the reactor wakes up and runs its trigger and transformation evaluations.

```
          [ MessageState ] ──► [ system_hardware_monitor ] ─► [ ErrorEvent ]
          (Heartbeat, 1m)
```

The watchdog handles a single message payload structure:
*   **`MessageState`:** Emitted periodically by the `NodeStatePlugin`, containing real-time host CPU, memory, and disk usage percent, along with the full process running statuses of BPR plugins, AI semantic reactors, and system plugins.

---

## 2. CEL Expressions

Because all metrics are consolidated in `MessageState`, the CEL expressions are extremely direct and easy to maintain.

### 2.1 Trigger Condition (`TriggerCondition`)
Ensures that an alert is fired only when a specific CPU/Memory/Disk threshold is exceeded OR when any BPR/AI/System process is not running.

```cel
(("cpu" in payload && "usage_pct" in payload.cpu && payload.cpu.usage_pct > 90.0) ||
 ("memory" in payload && "used_pct" in payload.memory && payload.memory.used_pct > 90.0) ||
 ("disk" in payload && "used_pct" in payload.disk && payload.disk.used_pct > 95.0) ||
 ("bpr_plugins" in payload && dyn(payload.bpr_plugins).exists(p, p.status != "running")) ||
 ("ai_reactors" in payload && dyn(payload.ai_reactors).exists(r, r.status != "active" && r.status != "sleeping")) ||
 ("system_plugins" in payload && dyn(payload.system_plugins).exists(sp, sp.status != "running")))
```

### 2.2 Transformation Expression
Generates the payload of the output `ErrorEvent` message. It uses inline conditionals to select the specific `error_type` and construct the `reason` dynamically depending on which telemetry metric violated the threshold:

```json
{
  "group_id": "\"mesh\"",
  "client_id": "head[\"$producer\"]",
  "username": "\"system\"",
  "origin_producer": "head[\"$producer\"]",
  "topic": "head[\"$type\"]",
  "reason": "((\"cpu\" in payload && payload.cpu.usage_pct > 90.0) || (\"memory\" in payload && payload.memory.used_pct > 90.0) || (\"disk\" in payload && payload.disk.used_pct > 95.0)) ? \"Host hardware resource threshold exceeded\" : \"Core system plugin or BPR process failure detected\"",
  "payload": "head[\"$type\"]",
  "error_type": "(\"cpu\" in payload && payload.cpu.usage_pct > 90.0) ? \"SYSTEM_HIGH_CPU\" : ((\"memory\" in payload && payload.memory.used_pct > 90.0) ? \"SYSTEM_HIGH_MEM\" : ((\"disk\" in payload && payload.disk.used_pct > 95.0) ? \"SYSTEM_HIGH_DISK\" : ((\"bpr_plugins\" in payload && dyn(payload.bpr_plugins).exists(p, p.status != \"running\")) ? \"BPR_PLUGIN_CRASH\" : ((\"ai_reactors\" in payload && dyn(payload.ai_reactors).exists(r, r.status != \"active\" && r.status != \"sleeping\")) ? \"AI_REACTOR_CRASH\" : \"SYSTEM_PLUGIN_CRASH\"))))",
  "timestamp": "head[\"$when\"]"
}
```

### 2.3 Raw `CELReactorSet` JSON Definition
You can find the complete `CELReactorSet` message configuration file representing this watchdog setup at:
📂 **[`poc_system_watchdog_config.json`](./poc_system_watchdog_config.json)**

Below is the raw structure:

```json
{
  "$type": "CELReactorSet",
  "$node-dest": "root",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "fields": {
    "gateway_node_id": "root",
    "reactors": [
      {
        "reactor_id": "system_hardware_monitor",
        "input_message_type": "MessageState",
        "output_message_type": "ErrorEvent",
        "context_query": "",
        "trigger_condition": "((\"cpu\" in payload && \"usage_pct\" in payload.cpu && payload.cpu.usage_pct > 90.0) || (\"memory\" in payload && \"used_pct\" in payload.memory && payload.memory.used_pct > 90.0) || (\"disk\" in payload && \"used_pct\" in payload.disk && payload.disk.used_pct > 95.0) || (\"bpr_plugins\" in payload && dyn(payload.bpr_plugins).exists(p, p.status != \"running\")) || (\"ai_reactors\" in payload && dyn(payload.ai_reactors).exists(r, r.status != \"active\" && r.status != \"sleeping\")) || (\"system_plugins\" in payload && dyn(payload.system_plugins).exists(sp, sp.status != \"running\")))",
        "transformation_expression": {
          "group_id": "\"mesh\"",
          "client_id": "head[\"$producer\"]",
          "username": "\"system\"",
          "origin_producer": "head[\"$producer\"]",
          "topic": "head[\"$type\"]",
          "reason": "((\"cpu\" in payload && payload.cpu.usage_pct > 90.0) || (\"memory\" in payload && payload.memory.used_pct > 90.0) || (\"disk\" in payload && payload.disk.used_pct > 95.0)) ? \"Host hardware resource threshold exceeded\" : \"Core system plugin or BPR process failure detected\"",
          "payload": "head[\"$type\"]",
          "error_type": "(\"cpu\" in payload && payload.cpu.usage_pct > 90.0) ? \"SYSTEM_HIGH_CPU\" : ((\"memory\" in payload && payload.memory.used_pct > 90.0) ? \"SYSTEM_HIGH_MEM\" : ((\"disk\" in payload && payload.disk.used_pct > 95.0) ? \"SYSTEM_HIGH_DISK\" : ((\"bpr_plugins\" in payload && dyn(payload.bpr_plugins).exists(p, p.status != \"running\")) ? \"BPR_PLUGIN_CRASH\" : ((\"ai_reactors\" in payload && dyn(payload.ai_reactors).exists(r, r.status != \"active\" && r.status != \"sleeping\")) ? \"AI_REACTOR_CRASH\" : \"SYSTEM_PLUGIN_CRASH\"))))",
          "timestamp": "head[\"$when\"]"
        },
        "cron": ""
      }
    ]
  }
}
```

---

## 3. Telemetry Exception Simulator Script

To verify that the watchdog triggers alerts correctly on a live mesh, compile and run the simulator script. It connects to the MQTT broker and publishes:
1. An unhealthy `MessageState` message with **98.5% CPU usage**.
2. An unhealthy `SystemVersion` message reporting that the **`hello-world-plugin` BPR process has stopped**.

The core engine will process these messages, run the Watchdog CEL Reactor, and publish **two separate `ErrorEvent` messages** to the mesh.

### Code: `publish_telemetry_exceptions.go`
You can save and compile this script locally at `./public/console/docs/cle-reactor-tutorial/publish_telemetry_exceptions.go`.

```go
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1883, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	flag.Parse()

	fmt.Println("=== Bindu Telemetry Exception Simulator ===")
	fmt.Printf("MQTT Broker: ssl://%s:%d | User: %s\n", *host, *port, *user)
	fmt.Println("--------------------------------------------------------")

	// Connect to MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-sysmon-sim-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	fmt.Println("Connected to MQTT successfully!")

	// 1. Publish Unhealthy MessageState (CPU threshold exceeded)
	msgStateID := fmt.Sprintf("sim-state-%s", uuid.NewString()[:8])
	stateMsg := map[string]any{
		"$type":       "MessageState",
		"$id":         msgStateID,
		"$group":      "mesh",
		"$producer":   "edge-node-sim",
		"$node":       "edge-node-sim", // EnvelopeField required by MessageState
		"$uptime_sec": 120,             // EnvelopeField required by MessageState
		"$user":       "system",
		"$when":       time.Now().UTC().Format(time.RFC3339),
		"fields": map[string]any{
			"cpu": map[string]any{
				"usage_pct": 98.5, // Alert limit is > 90%
			},
			"memory": map[string]any{
				"used_pct": 45.0,
			},
			"disk": map[string]any{
				"used_pct": 30.0,
			},
		},
	}
	publishMessage(client, stateMsg, "MessageState (High CPU)", msgStateID)

	time.Sleep(1 * time.Second)

	// 2. Publish Unhealthy MessageState (Stopped BPR plugin hello-world-plugin)
	msgVersionID := fmt.Sprintf("sim-version-%s", uuid.NewString()[:8])
	versionMsg := map[string]any{
		"$type":       "MessageState",
		"$id":         msgVersionID,
		"$group":      "mesh",
		"$producer":   "edge-node-sim",
		"$node":       "edge-node-sim", // EnvelopeField required by MessageState
		"$uptime_sec": 120,             // EnvelopeField required by MessageState
		"$user":       "system",
		"$when":       time.Now().UTC().Format(time.RFC3339),
		"fields": map[string]any{
			"bpr_plugins": []any{
				map[string]any{
					"id":      "hello-world-plugin",
					"version": "1.0.0",
					"status":  "stopped", // Alert triggers if != "running"
					"pid":     0,
				},
			},
		},
	}
	publishMessage(client, versionMsg, "MessageState (BPR Stopped)", msgVersionID)

	fmt.Println("\nExecution completed! Published 2 exception events to bindu/in.")
}

func publishMessage(client mqtt.Client, msg map[string]any, label string, msgID string) {
	payload, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal message: %v\n", err)
		return
	}

	token := client.Publish("bindu/in", 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to publish %s: %v\n", label, token.Error())
	} else {
		fmt.Printf("Published unhealthy %s (Message ID: %s)\n", label, msgID)
	}
}
```

---

## 4. Run the Simulation Walkthrough

To make execution seamless and automatically resolve the running MQTT broker port and authentication credentials of your active E2E mesh, run the wrapper bash script `simulate_exceptions.sh`:

1. **Verify Root/Node processes are running:**
   Ensure the local mesh is active.

2. **Execute the Simulation:**
   Run the wrapper script:
   ```bash
   ./public/console/docs/cle-reactor-tutorial/simulate_exceptions.sh
   ```
   This automatically reads target ports from the active E2E configurations (`.e2e-root/current/runtime/config/mesh.json` and `report.txt`) and invokes the Go script with the correct parameters for you.

3. **Verify Alert Outputs:**
   You can subscribe to MQTT messages to watch the incoming alerts:
   ```bash
   mosquitto_sub -t "bindu/in" -v
   ```
   You will see **two new `ErrorEvent` messages** emitted by the watchdog reactor matching your exception values.
