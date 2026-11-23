# 📋 Tutorial Plan: Heavy Equipment Condition-Based Monitoring (CBM) & Fleet Health MAS
### Non-Invasive Bolt-On BLE Telemetry, Store-and-Forward Drive-By Ingestion, Crest Factor + Kurtosis CEL Gating, and SLM Mechanical Diagnostics

---

## 1. Executive Summary & Objective

This plan details the implementation specification for the **Heavy Equipment Non-Invasive Condition-Based Monitoring (CBM) & Fleet Health Multi-Agent System (MAS)** in Bindu.

The solution proves that high-value mechanical assets ($500k to $3M each) operating in severe, disconnected environments (open-pit mining, quarrying, and demolition) can be governed by a sovereign, event-driven Multi-Agent System without modifying factory wiring, violating OEM CAN-bus warranties, or incurring expensive cloud SaaS telemetry subscriptions.

---

## 2. Ontology & Message Types Mapping

```
                                      [Bolt-on Triaxial BLE Beacon]
                                                    │
                                      (1-minute sampling, ±16g MEMS)
                                                    │
                                  ┌─────────────────┴─────────────────┐
                                  ▼                                   ▼
                         [Realtime Streaming]               [Drive-By Ring Buffer]
                                  │                                   │
                                  │ (BLE Advertisement)               │ (Burst of 30-60 samples)
                                  ▼                                   ▼
                    MachineVibrationSample              MachineVibrationBatchIngress
                                  │                                   │
                                  └─────────────────┬─────────────────┘
                                                    ▼
                                    [Bindu Sovereign Mesh Ledger]
                                            (v_messages)
                                                    │
                                  ┌─────────────────┴─────────────────┐
                                  ▼                                   ▼
                       [peak_g >= 8.5g && WORKING]       [crest >= 4.8 && kurtosis >= 4.2]
                                  │                                   │
                                  ▼                                   ▼
                       EquipmentAbuseIncident              BearingDefectSuspected
                                  │                                   │
                                  ▼                                   ▼
                      [Executive Abuse Alert]            [Zero-RAG Dynamic Context Query]
                                                                      │
                                                         (Hydrates 10 historical samples)
                                                                      │
                                                                      ▼
                                                      [Cognitive Semantic AI Reactor]
                                                        (qwen2.5-coder:0.5b / SLM)
                                                                      │
                                                                      ▼
                                                       PredictiveMaintenanceDiagnosis
                                                                      │
                                                       (RUL, Severity, Work Order Priority)
```

### Registered Ontologies:
1. `MachineVibrationSample`: Granular 1-minute vibration sample ($producer = machine_id, count retention = 100).
2. `MachineVibrationBatchIngress`: Drive-by burst collection bundle ($producer = machine_id, count retention = 20).
3. `EquipmentAbuseIncident`: Structural shock / impact overload incident ($producer = machine_id, count retention = 50).
4. `BearingDefectSuspected`: Early mechanical degradation trigger emitted by CEL ($producer = machine_id, count retention = 50).
5. `PredictiveMaintenanceDiagnosis`: Actionable RUL and work order emitted by Cognitive SLM ($producer = machine_id, count retention = 50).

---

## 3. The 4 Autonomous Agents of the Solution

### Agent 1: The Sensor Edge (Beacon & BPR Ingestion)
* **Contract Emitted:** `MachineVibrationSample`
* **Role:** Ingests raw acceleration, applies high-speed DSP / TinyML on the beacon or BPR gateway, and publishes classified operational state (`OFF`, `IDLE`, `WORKING`) along with mathematical metrics: RMS, Peak G, Crest Factor, and Kurtosis.

### Agent 2: The Structural Shock & Abuse Guard (Google CEL)
* **Contract Consumed:** `MachineVibrationSample`
* **Contract Emitted:** `EquipmentAbuseIncident`
* **Trigger Condition:** `has(payload.peak_g) && double(payload.peak_g) >= 8.5 && string(payload.operating_state) == 'WORKING'`
* **Execution Latency:** $< 2$ microseconds.
* **Role:** Detects equipment misuse (e.g. operators violently hammering rock with an excavator bucket or dropping 15-ton boulders directly into a dumper bed).

### Agent 3: The Bearing Defect Pre-Filter (Google CEL)
* **Contract Consumed:** `MachineVibrationSample`
* **Contract Emitted:** `BearingDefectSuspected`
* **Trigger Condition:** `has(payload.crest_factor) && has(payload.kurtosis) && double(payload.crest_factor) >= 4.8 && double(payload.kurtosis) >= 4.2`
* **Role:** Implements ISO 13373 dual-metric gating. Ensures that high vibration due to hard digging does NOT trigger false alarms. Only awakens the Cognitive SLM when mechanical impact micro-pitting is mathematically verified.

### Agent 4: The Reliability Chief Engineer (Semantic AI Reactor / SLM)
* **Contract Consumed:** `BearingDefectSuspected`
* **Contract Emitted:** `PredictiveMaintenanceDiagnosis`
* **Context Hydration:** Executes parameterized SQL against `v_messages` to retrieve the historical progression of RMS, Peak, Crest Factor, and Kurtosis for that specific machine.
* **Role:** Correlates the defect signature with operational hours, estimates Remaining Useful Life (RUL), and outputs a structured work order with exact shop recommendations.

---

## 4. Operational Web Console Dashboards

1. **Dashboard 1:** `mining-fleet-utilization-dashboard`
   - Real-time fleet overview showing operating states, RMS energy, Crest Factor, battery voltage, and ingestion mode.
2. **Dashboard 2:** `mining-abuse-incidents-dashboard`
   - Live audit trail of structural impact shocks exceeding safety thresholds, with incident IDs and mechanical recommendations.
3. **Dashboard 3:** `mining-predictive-maintenance-dashboard`
   - Actionable work order matrix displaying diagnosed failure modes, defect severity, RUL hours, and dispatch priority.

---

## 5. Deployment Verification Checklist

- [x] Level 0: `poc_message_types.json` validated via `bindu-studio message-types validate`
- [x] Level 1: `poc_cel_reactor_config.json` verified with zero errors via `bindu-studio cel-reactor dry-run`
- [x] Level 2: `poc_semantic_reactor_config.json` verified with zero errors and zero warnings via `bindu-studio semantic-ai-reactor validate`
- [x] Level 2: `poc_dashboards.json` structured with discrete `$producer` matching `fields.id`
- [x] Level 3: `mining_mas_solution.json` validated via `bindu-studio solution validate`
- [x] Level 4: `deploy_mas_solution.sh` deployment script created and chmod +x
- [x] Simulation: `simulate_mining_fleet.go` dry-run tested and verified
