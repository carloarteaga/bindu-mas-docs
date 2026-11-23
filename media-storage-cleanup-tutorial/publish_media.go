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
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type genericMessage struct {
	Type     string         `json:"$type"`
	ID       string         `json:"$id"`
	Group    string         `json:"$group"`
	Producer string         `json:"$producer"`
	User     string         `json:"$user"`
	When     string         `json:"$when"`
	Fields   map[string]any `json:"fields"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1885, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")

	s3Endpoint := flag.String("s3-endpoint", "http://127.0.0.1:9000", "MinIO / S3 Endpoint URL")
	accessKey := flag.String("access-key", "bindu-core-agent", "MinIO Access Key")
	secretKey := flag.String("secret-key", "bindu-media-123", "MinIO Secret Key")
	provider := flag.String("provider", "minio_local", "Storage Provider ID")
	bucket := flag.String("bucket", "raw", "Storage Bucket")
	srcDirFlag := flag.String("src-dir", "public/console/docs/media-storage-cleanup-tutorial/pictures", "Directory containing source plant JPEG images")
	flag.Parse()

	fmt.Println("=== Bindu Media Storage Cleanup Simulator: Publisher ===")
	fmt.Printf("MinIO Endpoint: %s | Provider: %s | Bucket: %s\n", *s3Endpoint, *provider, *bucket)
	fmt.Printf("MQTT Broker:    ssl://%s:%d | User: %s\n", *host, *port, *user)
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
		fmt.Fprintf(os.Stderr, "Warning: Failed to check if bucket %q exists: %v\n", *bucket, err)
	} else if !exists {
		fmt.Printf("Bucket %q does not exist. Creating bucket in MinIO...\n", *bucket)
		err = minioClient.MakeBucket(ctx, *bucket, minio.MakeBucketOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating bucket %q in MinIO: %v\n", *bucket, err)
			os.Exit(1)
		}
		fmt.Printf("Successfully created bucket %q in MinIO.\n", *bucket)
	}

	// Local source images mapping
	plantImages := []struct {
		targetKey      string
		fileCandidates []string
	}{
		{targetKey: "plant-1.jpeg", fileCandidates: []string{"plant-1.jpeg"}},
		{targetKey: "plant-2.jpeg", fileCandidates: []string{"plant-2.jpeg"}},
		{targetKey: "plant-3.jpeg", fileCandidates: []string{"plant-3.jpeg"}},
	}

	uploadedImages := make(map[string]bool)

	// 3. Perform Direct S3 Upload for each image
	for _, img := range plantImages {
		var imgBytes []byte
		var readErr error
		var matchedFile string

		for _, candidate := range img.fileCandidates {
			path := filepath.Join(*srcDirFlag, candidate)
			imgBytes, readErr = os.ReadFile(path)
			if readErr == nil {
				matchedFile = candidate
				break
			}
		}

		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not read source file for %s in %s. Skipping upload.\n", img.targetKey, *srcDirFlag)
			continue
		}

		producerID := strings.TrimSuffix(img.targetKey, ".jpeg")
		semanticObjectKey := fmt.Sprintf("PatioMonitorMedia/%s/%s", producerID, img.targetKey)

		fmt.Printf("Uploading %s (from local file %s) to MinIO bucket %q with key %q...\n", img.targetKey, matchedFile, *bucket, semanticObjectKey)
		_, err := minioClient.PutObject(ctx, *bucket, semanticObjectKey, bytes.NewReader(imgBytes), int64(len(imgBytes)), minio.PutObjectOptions{
			ContentType: "image/jpeg",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error uploading %s to MinIO: %v\n", semanticObjectKey, err)
			continue
		}

		fmt.Printf("  Successfully uploaded %s to MinIO bucket %s!\n", semanticObjectKey, *bucket)
		uploadedImages[img.targetKey] = true
	}

	// 4. Connect to MQTT and Publish PatioMonitorMedia Events
	fmt.Println("\nConnecting to MQTT broker...")
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-media-cleanup-pub-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)

	fmt.Println("\nPublishing PatioMonitorMedia events to MQTT...")
	publishedCount := 0
	for _, img := range plantImages {
		imgName := img.targetKey
		producerID := strings.TrimSuffix(imgName, ".jpeg")
		semanticObjectKey := fmt.Sprintf("PatioMonitorMedia/%s/%s", producerID, imgName)
		mediaURI := fmt.Sprintf("%s://%s/%s", *provider, *bucket, semanticObjectKey)
		msgID := fmt.Sprintf("patio-event-%d-%s", time.Now().UnixNano(), uuid.NewString()[:6])

		msg := genericMessage{
			Type:     "PatioMonitorMedia",
			ID:       msgID,
			Group:    "mesh",
			Producer: producerID,
			User:     "admin",
			When:     time.Now().UTC().Format(time.RFC3339),
			Fields: map[string]any{
				"patio_view": mediaURI,
			},
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal message for %s: %v\n", producerID, err)
			continue
		}

		token := client.Publish("bindu/in", 1, false, payload)
		token.Wait()
		if token.Error() != nil {
			fmt.Fprintf(os.Stderr, "Failed to publish PatioMonitorMedia for %s: %v\n", producerID, token.Error())
		} else {
			fmt.Printf("  Published PatioMonitorMedia (Producer: %s, URI: %s, Message ID: %s)\n", producerID, mediaURI, msgID)
			publishedCount++
		}
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("--------------------------------------------------------")
	fmt.Printf("Finished! %d media events published to MQTT.\n", publishedCount)
}
