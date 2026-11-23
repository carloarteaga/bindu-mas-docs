# 📡 Bindu BPR Plugin: Industrial LLRP RFID Reader & 11-Standard EPC Bit-Decoder

The `bindu-rfid-epc-plugin` connects Bindu to industrial RAIN RFID hardware readers via the standardized Low Level Reader Protocol (LLRP, ISO/IEC 24730-1). It features a zero-allocation bit-level decoding engine capable of unpacking all 11 EPCglobal Tag Data Standards (TDS 1.11 / 2.0).

---

## 🚀 Capabilities

* **Standard LLRP Client:** Connects via TCP to any LLRP-compliant reader (Impinj, Zebra, Alien) or the local `rfid-reader-simulator`.
* **11 EPC Standards Decoding:** Unpacks raw 96-bit hexadecimal tags into high-level semantic fields (`gtin14`, `sscc18`, `gln13`, etc.):
  1. SGTIN-96
  2. SSCC-96
  3. SGLN-96
  4. GRAI-96
  5. GIAI-96
  6. GSRN-96
  7. GDTI-96
  8. CPI-96
  9. SGCN-96
  10. ITIP-96
  11. GID-96
* **4-Antenna RF Telemetry:** Emits RF health metrics per antenna (TxPower, RSSI, Noise Floor, VSWR).
* **Pure Go BPR Runtime:** Zero CGo dependencies, communication strictly over `stdin/stdout`.

---

## 🛠️ Compilation & Certification

```bash
# Offline certification & dry-run
go run ./cmd/bindu-studio plugin certify --source public/console/docs/rfid-epc-tutorial/bindu-rfid-epc-plugin --dry-run

# Build certified release archive for amd64
go run ./cmd/bindu-studio plugin certify --source public/console/docs/rfid-epc-tutorial/bindu-rfid-epc-plugin --out public/console/docs/rfid-epc-tutorial/bindu-rfid-epc-plugin.tar.gz --target linux/amd64
```
