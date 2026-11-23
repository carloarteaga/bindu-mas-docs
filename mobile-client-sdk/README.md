# 📱 Bindu Sovereign Mobile Leaf Client SDK
### Canonical Reference Implementations for Android & iOS

This directory provides certified, ready-to-use reference wrappers for connecting mobile devices, handheld barcode/RFID terminals (Zebra, Honeywell), and tablets to the Bindu Sovereign AI Fabric.

---

## 📂 Structure

```text
mobile-client-sdk/
├── android/
│   └── BinduClient.kt       # Certified Kotlin implementation (Eclipse Paho MQTTv3)
├── ios/
│   └── BinduClient.swift    # Certified Swift implementation (iOS 16+, SwiftUI-ready)
└── README.md                # This document
```

---

## ⚡ Key Architectural Invariants

1. **Leaf Clients, NOT Nodes**:
   Mobile devices do NOT run a `bindu-node` daemon. They connect as lightweight, authenticated MQTT leaf clients over **MQTTS (Port 8883)** or **WSS (Port 8084)**.
2. **Real-Time Map Projection (`$where`)**:
   Adding GPS latitude, longitude, and altitude to the `$where` envelope field causes the Bindu Web Console **Geo-Locagram** map to automatically display real-time live markers without custom plugins.
3. **Ingress is strictly MQTT**:
   Publish events directly to `bindu/in` with QoS 1. There is no HTTP POST publish endpoint for event ingestion.
4. **Unique Hardware Producer**:
   `$producer` must be unique to the physical device (e.g., `handheld-s26-01`, IMEI, or tablet UUID) for clean retention partitioning.
5. **Zero-Touch Onboarding (Camera & No-Camera)**:
   Supports instant pairing via visual QR code (phones/tablets with camera) or 6-digit OTP PIN (rugged guns/scanners without camera or in high-security facilities).

---

## 🚀 Device Onboarding & Provisioning Workflows

### Option A: Devices with Camera (QR Code Pairing)
When configuring a device with a rear camera:
```kotlin
// Android (Kotlin)
val qrJson = scanQrCode() // From ZXing or CameraX
val config = BinduClient.parseEnrollmentQR(qrJson)

val caStream = BinduClient.fetchCACertificate("https://${config.gatewayHost}:8080", config.caUrl)
val client = BinduClient(context, config.serverUri, config.deviceId, config.clientToken)
client.connect(caStream)
```

### Option B: Devices without Camera (6-Digit OTP PIN Enrollment)
When configuring camera-less barcode guns (e.g., Zebra MC9300), vehicle terminals, or cleanroom devices:
```kotlin
// Android (Kotlin)
val config = BinduClient.enrollWithPIN(
    gatewayHttpUrl = "https://192.168.1.50:8080",
    pin = "849-201",
    hardwareId = "ZEBRA-MC9300-SN88291",
    model = "Zebra MC9300"
)

val caStream = BinduClient.fetchCACertificate("https://${config.gatewayHost}:8080", config.caUrl)
val client = BinduClient(context, config.serverUri, config.deviceId, config.clientToken)
client.connect(caStream)
```

```swift
// iOS (Swift)
BinduClient.enrollWithPIN(gatewayHttpUrl: "https://192.168.1.50:8080", pin: "849-201") { result in
    switch result {
    case .success(let config):
        let client = BinduClient(
            serverHost: config.gatewayHost,
            serverPort: UInt16(config.mqttPort),
            deviceId: config.deviceId,
            authToken: config.clientToken
        )
        // Connect to MQTT...
    case .failure(let error):
        print("Enrollment failed: \(error)")
    }
}
```

---

For the complete architectural guide, consult:
👉 **[`../mobile_and_leaf_client_guide.md`](../mobile_and_leaf_client_guide.md)**
