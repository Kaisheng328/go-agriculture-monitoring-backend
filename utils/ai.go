package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

var (
	mu        sync.RWMutex
	aiEnabled bool
	plantAI   string
)

// GetPredictedSoilMoisture calls the AI API to predict soil moisture based on temperature and humidity.
func GetPredictedSoilMoisture(plant string, timestamp string, temperature, humidity float32) (string, float64, error) {
	apiURL := os.Getenv("AI_URL")

	// Create a map with temperature, humidity, and timestamp
	requestBody, _ := json.Marshal(map[string]interface{}{
		"plant_name":  plant,
		"timestamp":   timestamp,
		"temperature": temperature,
		"humidity":    humidity,
	})

	// Make the HTTP POST request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, _ := ioutil.ReadAll(resp.Body)
	var response map[string]float64

	// Unmarshal the response into a map
	json.Unmarshal(body, &response)

	// Return the predicted soil moisture
	return timestamp, response["predicted_soil_moisture"], nil
}
func SetGlobalAIEnabled(enabled bool, plant string) {
	mu.Lock()
	defer mu.Unlock()
	aiEnabled = enabled
	plantAI = plant
}
func IsGlobalAIEnabled() (bool, string) {
	mu.RLock()
	defer mu.RUnlock()
	return aiEnabled, plantAI
}
