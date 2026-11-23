# 🔌 Bindu Universal Modbus BPR Plugin

The **Bindu Modbus BPR Plugin** is an industrial-grade, OS-isolated Go binary implementing the Bindu Plugin SDK (`bindunet/sdk`). It acts as a universal bridge translating high-level semantic Bindu events into low-level Modbus TCP read/write commands for Programmable Logic Controllers (PLCs), Variable Frequency Drives (VFDs), and environmental climate systems.

---

## 🏛️ Architecture & SDK Compliance

* **Communication:** JSON over standard I/O (`stdin` / `stdout`) with the Bindu host runtime.
* **Network Role:** Modbus TCP Client dialing configured industrial controllers over TCP (default `127.0.0.1:5020`).
* **Isolation:** Standalone OS process with zero side-effects, zero exposed HTTP endpoints, and pure deterministic message translation.

---

## 📜 Ingress Semantic Message Contracts

### 1. `ModbusConfig`
Reconfigures the active connection target dynamically at runtime:
```json
{
  "$type": "ModbusConfig",
  "$node-dest": "root",
  "$group": "config",
  "$producer": "operator",
  "$user": "admin",
  "$when": "2026-10-04T12:00:00Z",
  "fields": {
    "gateway_node_id": "root",
    "server_host": "192.168.1.150",
    "server_port": 502,
    "unit_id": 1,
    "timeout_ms": 1500
  }
}
```

### 2. `ModbusSet`
Writes a holding register or coil to the target PLC:
```json
{
  "$type": "ModbusSet",
  "$node-dest": "root",
  "$group": "mesh",
  "$producer": "potato-thermodynamics",
  "$user": "cel:engine",
  "$when": "2026-10-04T12:00:00Z",
  "fields": {
    "target_device": "fryer_controller",
    "register_type": "holding",
    "register_address": 200,
    "value": 178.5,
    "scale_factor": 10.0,
    "description": "Fryer oil temperature setpoint (1785 = 178.5 °C)"
  }
}
```

### 3. `ModbusGet`
Reads holding or input registers and emits `ModbusData`:
```json
{
  "$type": "ModbusGet",
  "$node-dest": "root",
  "$group": "mesh",
  "$producer": "truck-monitor",
  "$user": "operator",
  "$when": "2026-10-04T12:00:00Z",
  "fields": {
    "target_device": "truck_logger",
    "register_type": "holding",
    "register_address": 300,
    "quantity": 1,
    "scale_factor": 10.0,
    "request_id": "req-transit-temp-01"
  }
}
```

---

## 📤 Outbound Egress Events

### 1. `ModbusStatus`
Emitted after every write operation:
```json
{
  "$type": "ModbusStatus",
  "$group": "mesh",
  "$producer": "root",
  "$user": "bindu-modbus-plugin",
  "$when": "2026-10-04T12:00:01Z",
  "fields": {
    "command_id": "cmd-1728043201000",
    "target_device": "fryer_controller",
    "register_address": 200,
    "written_value": 1785.0,
    "status": "SUCCESS",
    "error_message": "",
    "timestamp": "2026-10-04T12:00:01Z"
  }
}
```

### 2. `ModbusData`
Emitted after successful read queries:
```json
{
  "$type": "ModbusData",
  "$group": "mesh",
  "$producer": "root",
  "$user": "bindu-modbus-plugin",
  "$when": "2026-10-04T12:00:01Z",
  "fields": {
    "request_id": "req-transit-temp-01",
    "target_device": "truck_logger",
    "register_address": 300,
    "raw_values": [65],
    "scaled_value": 6.5,
    "timestamp": "2026-10-04T12:00:01Z"
  }
}
```
