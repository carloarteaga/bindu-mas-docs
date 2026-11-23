# 🔌 Modbus TCP Server Simulator

A lightweight, standalone Modbus TCP Server CLI simulator written in pure Go (standard library only) for testing PLC register integrations with Bindu BPR plugins and Multi-Agent Systems.

---

## 🚀 Quick Start

### Build & Run

```bash
# From this directory:
go build -o modbus-sim .
./modbus-sim -port 5020 -unit-id 1
```

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `5020` | TCP port to listen on |
| `-unit-id` | `1` | Modbus Slave / Unit ID |

---

## 🗄️ Standard Register Map

| Register Address | Register Type | Engineering Name | Units / Scale | Initial Value | Description |
|---|---|---|---|---|---|
| **`100`** | Holding Register | `Silo_Temp_SP` | °C $\times$ 10 | `180` (18.0 °C) | Storage climate control setpoint |
| **`101`** | Holding Register | `Silo_Duration_SP` | Days | `0` | Calculated reconditioning duration |
| **`102`** | Holding Register | `Silo_Fan_Speed_SP` | RPM | `1200` | Silo ventilation fan speed |
| **`200`** | Holding Register | `Fryer_Oil_Temp_SP` | °C $\times$ 10 | `1750` (175.0 °C) | Fryer heat exchanger setpoint |
| **`201`** | Holding Register | `Fryer_Belt_Speed_SP` | mm/s | `30` | Conveyor belt VFD speed setpoint |
| **`202`** | Holding Register | `Fryer_Exhaust_Blower_SP` | % | `80` | Fryer vapor hood exhaust blower |
| **`300`** | Input / Holding Register | `Truck_Avg_Transit_Temp` | °C $\times$ 10 | `65` (6.5 °C) | Inbound truck datalogger reading |
| **`301`** | Input / Holding Register | `Truck_Transit_Duration` | Hours | `18` | Inbound transit duration |

---

## 📝 Example Output

```
[MODBUS PLC SIMULATOR] Listening on 0.0.0.0:5020 (Unit ID: 1)
[MODBUS PLC SIMULATOR] Pre-loaded Registers:
  • Holding Reg 100:   180  [Silo_Temp_SP (°C * 10)]
  • Holding Reg 101:     0  [Silo_Duration_SP (days)]
  • Holding Reg 200:  1750  [Fryer_Oil_Temp_SP (°C * 10)]
  • Holding Reg 201:    30  [Fryer_Belt_Speed_SP (mm/s)]

[15:04:05.123] Client connected: 127.0.0.1:49210
[15:04:05.125] ⚡ WRITE_SINGLE_REGISTER (0x06) | From: 127.0.0.1:49210 | Reg: 100 [Silo_Temp_SP (°C * 10)] = 180 (18.0 °C)
[15:04:35.200] ⚡ WRITE_SINGLE_REGISTER (0x06) | From: 127.0.0.1:49210 | Reg: 200 [Fryer_Oil_Temp_SP (°C * 10)] = 1785 (178.5 °C)
[15:04:35.202] ⚡ WRITE_SINGLE_REGISTER (0x06) | From: 127.0.0.1:49210 | Reg: 201 [Fryer_Belt_Speed_SP (mm/s)] = 32 (32 mm/s)
```
