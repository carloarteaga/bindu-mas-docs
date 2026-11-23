package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	filePath := "public/console/docs/system-email-bpr-tutorial/attachments/Bindu-AUM_AIM-HIM-KLIM.png"
	apiURL := "http://localhost:38080/api/v1/media/upload-url?provider=minio_local&bucket=raw&key=Bindu-AUM_AIM-HIM-KLIM.png"

	fmt.Println("1. Reading image file...")
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Println("2. Requesting presigned upload URL from Bindu API...")
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.SetBasicAuth("console", "StrongPass123") // Replace with actual credentials

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("API request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("API error (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		fmt.Printf("Failed to parse JSON response: %v\n", err)
		return
	}

	details, _ := res["details"].(map[string]any)
	uploadURL, _ := details["url"].(string)
	if uploadURL == "" {
		fmt.Printf("No upload URL returned in response: %+v\n", res)
		return
	}

	fmt.Println("3. Uploading image to MinIO via presigned PUT URL...")
	putReq, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileBytes))
	if err != nil {
		fmt.Printf("Error creating PUT request: %v\n", err)
		return
	}
	putReq.Header.Set("Content-Type", "image/png")
	putReq.ContentLength = int64(len(fileBytes))

	putResp, err := client.Do(putReq)
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(putResp.Body)
		fmt.Printf("Upload error (%d): %s\n", putResp.StatusCode, string(body))
		return
	}

	fmt.Println("Success! Image uploaded to MinIO.")
	fmt.Println("Out-of-band Media Storage URI: minio_local://raw/Bindu-AUM_AIM-HIM-KLIM.png")
}
