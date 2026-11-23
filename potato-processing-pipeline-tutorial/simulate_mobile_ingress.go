//go:build ignore

// ==============================================================================
// 🥔 Mobile Intake Scanner Simulator for Potato Processing Pipeline
// ==============================================================================
// Uploads optical slice images out-of-band to MinIO S3 storage (following the
// Bindu Media Storage Standard) and publishes PotatoBatchIngress to MQTT bindu/in.
// ==============================================================================

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PotatoBatchIngressFields struct {
	LotID                string  `json:"lot_id"`
	Variety              string  `json:"variety"`
	IntakeWeightKg       float64 `json:"intake_weight_kg"`
	AvgTransitTempC      float64 `json:"avg_transit_temp_c"`
	TransitDurationHours float64 `json:"transit_duration_hours"`
	OpticalSliceImageURI string  `json:"optical_slice_image_uri,omitempty"`
	OriginFarm           string  `json:"origin_farm,omitempty"`
	Timestamp            string  `json:"timestamp"`
}

type PotatoBatchIngressMessage struct {
	Type     string                   `json:"$type"`
	ID       string                   `json:"$id"`
	When     string                   `json:"$when"`
	Group    string                   `json:"$group"`
	Producer string                   `json:"$producer"`
	User     string                   `json:"$user"`
	NodeDest string                   `json:"$node-dest"`
	Fields   PotatoBatchIngressFields `json:"fields"`
}

func main() {
	// MQTT Flags
	host := flag.String("host", "192.168.12.160", "MQTT Broker Host")
	port := flag.Int("port", 1885, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	broker := flag.String("broker", "", "Explicit MQTT broker endpoint (e.g. ssl://192.168.12.160:1885)")

	// MinIO / Out-of-band Media Storage Flags
	s3Endpoint := flag.String("s3-endpoint", "http://192.168.12.160:9000", "MinIO / S3 Endpoint URL")
	accessKey := flag.String("access-key", "bindu-core-agent", "MinIO Access Key")
	secretKey := flag.String("secret-key", "bindu-media-123", "MinIO Secret Key")
	provider := flag.String("provider", "minio_local", "Storage Provider ID")
	bucket := flag.String("bucket", "raw", "Storage Bucket Name")
	srcDirFlag := flag.String("src-dir", "public/console/docs/potato-processing-pipeline-tutorial/pictures", "Directory containing source potato PNG images")

	// Potato Inspection Preset Flags
	potatoOption := flag.Int("potato", 1, "Select preset potato profile: 1 = papas-1.png (Cold Damaged CIS), 2 = Papas-2.png (Fresh Harvest)")
	lotID := flag.String("lot-id", "", "Potato Batch / Lot ID (optional, auto-generated if empty)")
	variety := flag.String("variety", "", "Potato cultivar variety")
	temp := flag.Float64("temp", 0.0, "Average transit temperature (°C)")
	hours := flag.Float64("hours", 0.0, "Transit duration (hours)")
	weight := flag.Float64("weight", 0.0, "Intake batch weight (kg)")
	imgPath := flag.String("image", "", "Path to raw sliced potato image")
	nodeDest := flag.String("node-dest", "root", "Target node destination")
	flag.Parse()

	// Handle preset potato selection (1 vs 2)
	var finalLotID, finalVariety, finalImgTarget string
	var finalTemp, finalHours, finalWeight float64

	switch *potatoOption {
	case 1:
		finalLotID = "lot-potato-cold-001"
		finalVariety = "Russet Burbank (Cold-Stressed)"
		finalTemp = 4.8 // < 8.0°C causes Cold-Induced Sweetening (CIS)
		finalHours = 22.0
		finalWeight = 24500.0
		finalImgTarget = "papas-1.png"
	case 2:
		finalLotID = "lot-potato-fresh-002"
		finalVariety = "Atlantic (Optimal Harvest)"
		finalTemp = 11.5 // >= 8.0°C starch remains stable
		finalHours = 14.0
		finalWeight = 28000.0
		finalImgTarget = "Papas-2.png"
	default:
		// Custom mode
		finalLotID = *lotID
		if finalLotID == "" {
			finalLotID = fmt.Sprintf("lot-potato-%d", time.Now().Unix()%10000)
		}
		finalVariety = *variety
		if finalVariety == "" {
			finalVariety = "Russet Burbank"
		}
		finalTemp = *temp
		if finalTemp == 0.0 {
			finalTemp = 4.8
		}
		finalHours = *hours
		if finalHours == 0.0 {
			finalHours = 18.0
		}
		finalWeight = *weight
		if finalWeight == 0.0 {
			finalWeight = 25000.0
		}
		finalImgTarget = "papas-1.png"
	}

	// Override if user explicitly specified flags
	if *lotID != "" {
		finalLotID = *lotID
	}
	if *variety != "" {
		finalVariety = *variety
	}
	if *temp != 0.0 {
		finalTemp = *temp
	}
	if *hours != 0.0 {
		finalHours = *hours
	}
	if *weight != 0.0 {
		finalWeight = *weight
	}
	if *imgPath != "" {
		finalImgTarget = *imgPath
	}

	fmt.Println("=== Bindu Potato Intake Media Storage & Event Publisher ===")
	fmt.Printf("MinIO Endpoint: %s | Provider: %s | Bucket: %s\n", *s3Endpoint, *provider, *bucket)
	fmt.Printf("MQTT Broker:    tcp://%s:%d | User: %s\n", *host, *port, *user)
	fmt.Println("--------------------------------------------------------")

	// 1. Initialize Direct MinIO Client
	u, err := url.Parse(*s3Endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing S3 endpoint URL: %v\n", err)
		os.Exit(1)
	}
	s3Host := u.Host
	if s3Host == "" {
		s3Host = u.Path
	}
	secure := u.Scheme == "https"

	minioClient, err := minio.New(s3Host, &minio.Options{
		Creds:  credentials.NewStaticV4(*accessKey, *secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing MinIO client: %v\n", err)
		os.Exit(1)
	}

	// 2. Ensure bucket exists in MinIO
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, *bucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed checking if bucket %q exists: %v\n", *bucket, err)
	} else if !exists {
		fmt.Printf("Bucket %q does not exist. Creating bucket in MinIO...\n", *bucket)
		err = minioClient.MakeBucket(ctx, *bucket, minio.MakeBucketOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating bucket %q in MinIO: %v\n", *bucket, err)
			os.Exit(1)
		}
		fmt.Printf("Successfully created bucket %q in MinIO.\n", *bucket)
	}

	// 3. Resolve and read slice image file from disk
	var imgBytes []byte
	var resolvedPath string
	candidates := []string{
		finalImgTarget,
		filepath.Join(*srcDirFlag, finalImgTarget),
		filepath.Join(*srcDirFlag, filepath.Base(finalImgTarget)),
		filepath.Join("public/console/docs/potato-processing-pipeline-tutorial/pictures", filepath.Base(finalImgTarget)),
		filepath.Join("pictures", filepath.Base(finalImgTarget)),
	}

	for _, cand := range candidates {
		if cand == "" {
			continue
		}
		if data, readErr := os.ReadFile(cand); readErr == nil && len(data) > 0 {
			imgBytes = data
			resolvedPath = cand
			break
		}
	}

	if len(imgBytes) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Could not read image file %q from candidates %v\n", finalImgTarget, candidates)
		os.Exit(1)
	}

	imgFileName := filepath.Base(resolvedPath)
	if imgFileName == "" || imgFileName == "." {
		imgFileName = "papas-1.png"
	}

	// Standard Semantic S3 Key: PotatoBatchIngress/{producer}/{filename}
	semanticObjectKey := fmt.Sprintf("PotatoBatchIngress/%s/%s", finalLotID, imgFileName)
	mediaURI := fmt.Sprintf("%s://%s/%s", *provider, *bucket, semanticObjectKey)

	// 4. Perform Direct S3 Upload to MinIO
	contentType := "image/png"
	if strings.HasSuffix(strings.ToLower(imgFileName), ".jpg") || strings.HasSuffix(strings.ToLower(imgFileName), ".jpeg") {
		contentType = "image/jpeg"
	}

	fmt.Printf("Uploading %s (from local file %s) to MinIO bucket %q with key %q...\n", imgFileName, resolvedPath, *bucket, semanticObjectKey)
	_, putErr := minioClient.PutObject(ctx, *bucket, semanticObjectKey, bytes.NewReader(imgBytes), int64(len(imgBytes)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if putErr != nil {
		fmt.Fprintf(os.Stderr, "Error uploading %s to MinIO: %v\n", semanticObjectKey, putErr)
		os.Exit(1)
	}
	fmt.Printf("  Successfully uploaded %s to MinIO bucket %s!\n\n", semanticObjectKey, *bucket)

	// 5. Build Envelope conforming strictly to Bindu Ingress Contract & Media-Storage Schema
	now := time.Now().UTC().Format(time.RFC3339)
	msg := PotatoBatchIngressMessage{
		Type:     "PotatoBatchIngress",
		ID:       fmt.Sprintf("scan-%s-%d", finalLotID, time.Now().UnixNano()%1000000),
		When:     now,
		Group:    "mesh",
		Producer: finalLotID, // Partition triplet ($type, $group, $producer) matches unique digital twin lot ID
		User:     *user,
		NodeDest: *nodeDest,
		Fields: PotatoBatchIngressFields{
			LotID:                finalLotID,
			Variety:              finalVariety,
			IntakeWeightKg:       finalWeight,
			AvgTransitTempC:      finalTemp,
			TransitDurationHours: finalHours,
			OpticalSliceImageURI: mediaURI,
			OriginFarm:           "Valley Farms Sector 4",
			Timestamp:            now,
		},
	}

	payload, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=================================================================")
	fmt.Printf("🥔 [MOBILE INTAKE SCANNER] Publishing PotatoBatchIngress for %s\n", finalLotID)
	fmt.Printf("   ├─ Selected Profile: Option %d (%s)\n", *potatoOption, resolvedPath)
	fmt.Printf("   ├─ Cultivar Variety: %s\n", finalVariety)
	fmt.Printf("   ├─ Intake Weight:    %.1f kg\n", finalWeight)
	fmt.Printf("   ├─ Avg Transit Temp: %.1f °C\n", finalTemp)
	fmt.Printf("   ├─ Out-of-Band URI:  %s\n", mediaURI)

	if finalTemp < 8.0 {
		fmt.Printf("   ├─ 🚨 COLD STRESS DETECTED (%.1f °C < 8.0 °C):\n", finalTemp)
		fmt.Println("   │  • Model A Trigger: Cold-Induced Sweetening (CIS) detected.")
		fmt.Println("   │  • Optical AI (llava-phi3 / CEL): High sugar proxy C_RS ≈ 0.41 (> 0.20 threshold).")
		fmt.Println("   │  • Kinetic Scheduling: SystemScheduledEvent timer registered (30s quarantine).")
		fmt.Println("   │  • Modbus PLC Action: Writes Silo Climate Controller Reg 100 = 180 (18.0 °C).")
		fmt.Println("   │  • Awakening Flow: SystemCronJob fires after 30s -> Fryer configured to 178.5 °C.")
	} else {
		fmt.Printf("   ├─ ✅ OPTIMAL STARCH INTAKE (%.1f °C >= 8.0 °C):\n", finalTemp)
		fmt.Println("   │  • Model B & C Trigger: High starch integrity, low reducing sugars.")
		fmt.Println("   │  • Optical AI (llava-phi3 / CEL): Low sugar proxy C_RS ≈ 0.12 (<= 0.20 threshold).")
		fmt.Println("   │  • Direct Fryer Action: Immediate fry approved! No silo quarantine needed.")
		fmt.Println("   │  • Modbus PLC Action: Writes Fryer Oil Temp Reg 200 = 1814 (181.4 °C) & Belt Speed Reg 201 = 35 mm/s.")
	}
	fmt.Println("=================================================================")

	// 6. Connect to MQTT Broker and Publish
	var client mqtt.Client
	var connected bool
	var lastErr error

	brokerEndpoints := []string{}
	if *broker != "" {
		brokerEndpoints = append(brokerEndpoints, *broker)
	} else {
		brokerEndpoints = append(brokerEndpoints,
			fmt.Sprintf("ssl://%s:%d", *host, *port),
			fmt.Sprintf("tcp://%s:%d", *host, *port),
		)
	}

	for _, ep := range brokerEndpoints {
		opts := mqtt.NewClientOptions().
			AddBroker(ep).
			SetClientID(fmt.Sprintf("mobile-scanner-%d", time.Now().UnixNano())).
			SetUsername(*user).
			SetPassword(*pass).
			SetTLSConfig(&tls.Config{InsecureSkipVerify: true}).
			SetCleanSession(true).
			SetConnectTimeout(5 * time.Second)

		c := mqtt.NewClient(opts)
		token := c.Connect()
		if token.WaitTimeout(5*time.Second) && token.Error() == nil {
			client = c
			connected = true
			fmt.Printf("🔌 Connected to MQTT broker (%s)\n", ep)
			break
		}
		lastErr = token.Error()
	}

	if !connected {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to MQTT broker (%v): %v\n", brokerEndpoints, lastErr)
		os.Exit(1)
	}
	defer client.Disconnect(250)

	token := client.Publish("bindu/in", 0, false, payload)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to publish to bindu/in: %v\n", token.Error())
		os.Exit(1)
	}

	fmt.Printf("✅ Event successfully published to bindu/in ($id: %s)!\n\n", msg.ID)
}
