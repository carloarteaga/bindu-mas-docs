package io.bindu.mobile.sdk

import android.content.Context
import android.location.Location
import org.eclipse.paho.client.mqttv3.*
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence
import org.json.JSONObject
import java.io.InputStream
import java.security.KeyStore
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import java.text.SimpleDateFormat
import java.util.*
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManagerFactory

/**
 * Sovereign Bindu Leaf Client for Android (Kotlin).
 * 
 * Implements the canonical SEPN envelope contract, automated TLS handshake with 
 * cluster CA trust, bottom-up ingress on 'bindu/in', and native Geo-Locagram GPS spatial projection.
 *
 * Requirements (build.gradle.kts):
 * implementation("org.eclipse.paho:org.eclipse.paho.client.mqttv3:1.2.5")
 */
class BinduClient(
    private val context: Context,
    private val serverUri: String, // e.g., "ssl://root.bindu.lan:8883" or "wss://root.bindu.lan:8084"
    private val deviceId: String,  // Unique hardware producer ID (e.g., "handheld-s26-01")
    private val authToken: String  // Auth token from accessControlConfig.brs
) : MqttCallbackExtended {

    private var mqttClient: MqttAsyncClient? = null
    private val isoFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", Locale.US).apply {
        timeZone = TimeZone.getTimeZone("UTC")
    }

    var onDirectiveReceived: ((type: String, payload: JSONObject) -> Unit)? = null
    var onConnectionStateChanged: ((connected: Boolean) -> Unit)? = null

    /**
     * Establishes a secure TLS connection to the Bindu Root or Edge Gateway.
     * @param caCertificateStream InputStream of the cluster CA certificate (downloaded from /api/v1/ca.crt)
     */
    fun connect(caCertificateStream: InputStream) {
        try {
            mqttClient = MqttAsyncClient(serverUri, deviceId, MemoryPersistence())
            mqttClient?.setCallback(this)

            val options = MqttConnectOptions().apply {
                isAutomaticReconnect = true
                isCleanSession = false
                connectionTimeout = 15
                keepAliveInterval = 30
                userName = deviceId
                password = authToken.toCharArray()
                socketFactory = createSocketFactory(caCertificateStream)
            }

            mqttClient?.connect(options, null, object : IMqttActionListener {
                override fun onSuccess(asyncActionToken: IMqttToken?) {
                    subscribeToDirectives()
                }

                override fun onFailure(asyncActionToken: IMqttToken?, exception: Throwable?) {
                    onConnectionStateChanged?.invoke(false)
                }
            })
        } catch (e: Exception) {
            e.printStackTrace()
            onConnectionStateChanged?.invoke(false)
        }
    }

    /**
     * Publishes a strongly-typed domain event to 'bindu/in' with real-time GPS spatial coordinates.
     */
    fun publishEvent(
        messageType: String,
        fields: JSONObject,
        location: Location? = null,
        group: String = "default"
    ) {
        val envelope = JSONObject().apply {
            put("$id", UUID.randomUUID().toString())
            put("$type", messageType)
            put("$producer", deviceId)
            put("$when", isoFormat.format(Date()))
            put("$group", group)
            location?.let {
                val whereObj = JSONObject().apply {
                    put("lat", it.latitude)
                    put("lon", it.longitude)
                    put("alt", it.altitude)
                }
                put("$where", whereObj)
            }
            put("fields", fields)
        }

        val message = MqttMessage(envelope.toString().toByteArray(Charsets.UTF_8)).apply {
            qos = 1
        }
        mqttClient?.publish("bindu/in", message)
    }

    /**
     * Emits a canonical device heartbeat and presence state for monitoring in the Console.
     */
    fun publishHeartbeat(batteryPct: Int, networkType: String, location: Location? = null) {
        val fields = JSONObject().apply {
            put("battery_percent", batteryPct)
            put("network_type", networkType)
            put("status", "ACTIVE")
            put("platform", "Android")
        }
        publishEvent("DeviceHeartbeat", fields, location)
    }

    private fun subscribeToDirectives() {
        val directiveTopic = "hh/dispatch/$deviceId"
        mqttClient?.subscribe(directiveTopic, 1)
    }

    override fun connectComplete(reconnect: Boolean, serverURI: String?) {
        onConnectionStateChanged?.invoke(true)
        if (reconnect) {
            subscribeToDirectives()
        }
    }

    override fun connectionLost(cause: Throwable?) {
        onConnectionStateChanged?.invoke(false)
    }

    override fun messageArrived(topic: String?, message: MqttMessage?) {
        message?.let {
            try {
                val json = JSONObject(String(it.payload, Charsets.UTF_8))
                val type = json.optString("$type", "UnknownDirective")
                onDirectiveReceived?.invoke(type, json)
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }

    override fun deliveryComplete(token: IMqttDeliveryToken?) {}

    private fun createSocketFactory(caStream: InputStream): javax.net.ssl.SSLSocketFactory {
        val cf = CertificateFactory.getInstance("X.509")
        val caCert = cf.generateCertificate(caStream) as X509Certificate

        val keyStore = KeyStore.getInstance(KeyStore.getDefaultType()).apply {
            load(null, null)
            setCertificateEntry("bindu-cluster-ca", caCert)
        }

        val tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm()).apply {
            init(keyStore)
        }

        val sslContext = SSLContext.getInstance("TLSv1.3").apply {
            init(null, tmf.trustManagers, null)
        }
        return sslContext.socketFactory
    }

    fun disconnect() {
        try {
            mqttClient?.disconnect()
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    /**
     * Standardized Enrollment Configuration returned by both QR Code and PIN onboarding flows.
     */
    data class EnrollmentConfig(
        val version: Int,
        val clusterName: String,
        val gatewayHost: String,
        val mqttPort: Int,
        val wsPort: Int,
        val caUrl: String,
        val deviceId: String,
        val clientToken: String,
        val defaultRole: String
    ) {
        val serverUri: String
            get() = "ssl://$gatewayHost:$mqttPort"
    }

    companion object {
        /**
         * Parses the QR Code JSON payload scanned by devices equipped with a camera.
         */
        fun parseEnrollmentQR(qrJsonString: String): EnrollmentConfig {
            val json = JSONObject(qrJsonString)
            return EnrollmentConfig(
                version = json.optInt("version", 1),
                clusterName = json.optString("cluster_name", "bindu-cluster"),
                gatewayHost = json.getString("gateway_host"),
                mqttPort = json.optInt("mqtt_port", 1885),
                wsPort = json.optInt("ws_port", 8084),
                caUrl = json.optString("ca_url", "/api/v1/ca.crt"),
                deviceId = json.getString("device_id"),
                clientToken = json.getString("client_token"),
                defaultRole = json.optString("default_role", "leaf_client")
            )
        }

        /**
         * Enrolls a camera-less or restricted device using a 6-digit OTP PIN via HTTP POST /api/v1/devices/enroll.
         */
        fun enrollWithPIN(
            gatewayHttpUrl: String, // e.g. "https://192.168.1.50:8080"
            pin: String,
            hardwareId: String? = null,
            model: String? = null
        ): EnrollmentConfig {
            val cleanPin = pin.replace("-", "").replace(" ", "").trim()
            val endpoint = "$gatewayHttpUrl/api/v1/devices/enroll"
            val url = java.net.URL(endpoint)
            val conn = url.openConnection() as java.net.HttpURLConnection
            conn.requestMethod = "POST"
            conn.setRequestProperty("Content-Type", "application/json")
            conn.doOutput = true

            val body = JSONObject().apply {
                put("pin", cleanPin)
                hardwareId?.let { put("device_hardware_id", it) }
                model?.let { put("device_model", it) }
            }

            conn.outputStream.use { os ->
                os.write(body.toString().toByteArray(Charsets.UTF_8))
            }

            if (conn.responseCode != 200) {
                val err = conn.errorStream?.bufferedReader()?.use { it.readText() } ?: ""
                throw IllegalStateException("Enrollment failed (HTTP ${conn.responseCode}): $err")
            }

            val resp = conn.inputStream.bufferedReader().use { it.readText() }
            val json = JSONObject(resp)
            if (!json.optBoolean("success", false)) {
                throw IllegalStateException("Enrollment failed: ${json.optString("error")}")
            }
            val config = json.getJSONObject("config")
            return parseEnrollmentQR(config.toString())
        }

        /**
         * Helper to download the Cluster Root CA certificate directly from the gateway.
         */
        fun fetchCACertificate(gatewayHttpUrl: String, caPath: String = "/api/v1/ca.crt"): InputStream {
            val url = java.net.URL("$gatewayHttpUrl$caPath")
            return url.openStream()
        }
    }
}
