package commander

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	commanderURL    string
	commanderAPIKey string
	instanceID      string
	client          *http.Client
)

func init() {
	commanderURL = os.Getenv("COMMANDER_URL")
	commanderAPIKey = os.Getenv("COMMANDER_API_KEY")
	instanceID = os.Getenv("WORKBENCH_ID")

	if instanceID == "" {
		// Generate a unique instance ID if not provided
		instanceID = fmt.Sprintf("wb-%d", time.Now().Unix())
	}

	client = &http.Client{
		Timeout: 10 * time.Second,
	}

	// Register with Commander if configured
	if commanderURL != "" && commanderAPIKey != "" {
		go registerWithCommander()
		go startHeartbeat()
	}
}

// registerWithCommander registers this workbench instance with Commander
func registerWithCommander() {
	time.Sleep(5 * time.Second) // Wait for services to start

	payload := map[string]any{
		"instance_id": instanceID,
		"name":        os.Getenv("HOSTNAME"),
		"version":     "1.0.0",
		"ip":          getLocalIP(),
		"api_key":     commanderAPIKey,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", commanderURL+"/api/v1/register", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Commander: Failed to create registration request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", commanderAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Commander: Failed to register: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("Commander: Successfully registered with Commander")
	} else {
		log.Printf("Commander: Registration failed with status %d", resp.StatusCode)
	}
}

// startHeartbeat sends periodic heartbeats to Commander
func startHeartbeat() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sendHeartbeat()
	}
}

// sendHeartbeat sends system metrics to Commander
func sendHeartbeat() {
	payload := map[string]any{
		"cpu":    getCPUUsage(),
		"memory": getMemoryUsage(),
		"disk":   getDiskUsage(),
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", commanderURL+"/api/v1/heartbeat", bytes.NewBuffer(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", commanderAPIKey)

	client.Do(req)
}

// CompleteAI sends an AI completion request to Commander
func CompleteAI(prompt string) (string, error) {
	if commanderURL == "" || commanderAPIKey == "" {
		return "", fmt.Errorf("Commander not configured")
	}

	payload := map[string]any{
		"prompt": prompt,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", commanderURL+"/api/v1/ai/complete", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", commanderAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("AI request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Text, nil
}

// IsConfigured returns whether Commander is configured
func IsConfigured() bool {
	return commanderURL != "" && commanderAPIKey != ""
}

// Helper functions for system metrics
func getCPUUsage() float64 {
	// Simplified - in production use proper monitoring
	return 25.0
}

func getMemoryUsage() float64 {
	// Simplified - in production use proper monitoring
	return 50.0
}

func getDiskUsage() float64 {
	// Simplified - in production use proper monitoring
	return 30.0
}

func getLocalIP() string {
	hostname, _ := os.Hostname()
	return hostname
}
