package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fyp/models"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	mu        sync.RWMutex
	aiEnabled bool
	plantAI   string
)

// AIRequestData represents the structure for the AI API request
type AIRequestData struct {
	PlantName         string `json:"plant_name"`
	Timestamp         string `json:"timestamp"`
	Temperature       string `json:"temperature"`
	Humidity          string `json:"humidity"`
	TempRolling3      string `json:"temp_rolling_3"`
	HumidityRolling3  string `json:"humidity_rolling_3"`
	TempRolling24     string `json:"temp_rolling_24"`
	HumidityRolling24 string `json:"humidity_rolling_24"`
	TempLag1          string `json:"temp_lag_1"`
	HumidityLag1      string `json:"humidity_lag_1"`
}

// calculateRollingAverage calculates the average of the last n records
func calculateRollingAverage(values []float32, n int) float32 {
	if len(values) == 0 {
		return 0
	}

	// Take the last n values (or all if less than n)
	start := 0
	if len(values) > n {
		start = len(values) - n
	}

	var sum float32
	count := 0
	for i := start; i < len(values); i++ {
		sum += values[i]
		count++
	}

	if count == 0 {
		return 0
	}
	return sum / float32(count)
}

// getHistoricalData retrieves historical sensor data for calculating features
func getHistoricalData(db *gorm.DB, userID uint, currentTime time.Time) ([]models.SensorData, error) {
	var sensorData []models.SensorData

	// Get data from the last 24 hours, ordered by timestamp
	twentyFourHoursAgo := currentTime.Add(-24 * time.Hour)

	err := db.Where("user_id = ? AND timestamp >= ? AND timestamp < ?",
		userID, twentyFourHoursAgo, currentTime).
		Order("timestamp ASC").
		Find(&sensorData).Error

	return sensorData, err
}

// calculateFeatures calculates rolling averages and lag features from historical data
func calculateFeatures(historicalData []models.SensorData, currentTemp, currentHumidity float32) (AIRequestData, error) {
	var tempValues, humidityValues []float32

	// Extract temperature and humidity values
	for _, data := range historicalData {
		tempValues = append(tempValues, data.Temperature)
		humidityValues = append(humidityValues, data.Humidity)
	}

	// Calculate rolling averages
	// For 3-hour rolling average, assuming data points are hourly
	tempRolling3 := calculateRollingAverage(tempValues, 3)
	humidityRolling3 := calculateRollingAverage(humidityValues, 3)

	// For 24-hour rolling average
	tempRolling24 := calculateRollingAverage(tempValues, 24)
	humidityRolling24 := calculateRollingAverage(humidityValues, 24)

	// Calculate lag features (previous values)
	var tempLag1, humidityLag1 float32
	if len(tempValues) > 0 {
		tempLag1 = tempValues[len(tempValues)-1]
		humidityLag1 = humidityValues[len(humidityValues)-1]
	} else {
		// If no historical data, use current values as fallback
		tempLag1 = currentTemp
		humidityLag1 = currentHumidity
	}

	features := AIRequestData{
		Temperature:       fmt.Sprintf("%.1f", currentTemp),
		Humidity:          fmt.Sprintf("%.1f", currentHumidity),
		TempRolling3:      fmt.Sprintf("%.1f", tempRolling3),
		HumidityRolling3:  fmt.Sprintf("%.1f", humidityRolling3),
		TempRolling24:     fmt.Sprintf("%.1f", tempRolling24),
		HumidityRolling24: fmt.Sprintf("%.1f", humidityRolling24),
		TempLag1:          fmt.Sprintf("%.1f", tempLag1),
		HumidityLag1:      fmt.Sprintf("%.1f", humidityLag1),
	}

	return features, nil
}

// GetPredictedSoilMoisture calls the AI API to predict soil moisture with enhanced features
func GetPredictedSoilMoisture(db *gorm.DB, userID uint, plant string, timestamp string, temperature, humidity float32) (string, float64, error) {
	apiURL := os.Getenv("AI_URL")

	// Parse timestamp to get the current time
	currentTime, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	// Get historical data for feature calculation
	historicalData, err := getHistoricalData(db, userID, currentTime)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get historical data: %v", err)
	}

	// Calculate features
	features, err := calculateFeatures(historicalData, temperature, humidity)
	if err != nil {
		return "", 0, fmt.Errorf("failed to calculate features: %v", err)
	}

	// Set plant name and timestamp
	features.PlantName = plant
	features.Timestamp = timestamp

	// Create the request body
	requestBody, err := json.Marshal(features)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Log the request for debugging
	fmt.Printf("AI API Request URL: %s\n", apiURL)
	fmt.Printf("AI API Request Body (Historical): %s\n", string(requestBody))

	// Make the HTTP POST request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	type PredictionResponse struct {
		PredictedSoilMoisture float64 `json:"predicted_soil_moisture"`
		Timestamp             string  `json:"timestamp"`
	}

	var response PredictionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, err
	}

	// Return the predicted soil moisture
	return response.Timestamp, response.PredictedSoilMoisture, nil
}

// GetPredictedSoilMoistureSimple - fallback function for when you don't have historical data
func GetPredictedSoilMoistureSimple(plant string, timestamp string, temperature, humidity float32) (string, float64, error) {
	apiURL := os.Getenv("AI_URL")

	// Create request with current values as fallback for missing features
	requestData := AIRequestData{
		PlantName:         plant,
		Timestamp:         timestamp,
		Temperature:       fmt.Sprintf("%.1f", temperature),
		Humidity:          fmt.Sprintf("%.1f", humidity),
		TempRolling3:      fmt.Sprintf("%.1f", temperature), // Use current as fallback
		HumidityRolling3:  fmt.Sprintf("%.1f", humidity),    // Use current as fallback
		TempRolling24:     fmt.Sprintf("%.1f", temperature), // Use current as fallback
		HumidityRolling24: fmt.Sprintf("%.1f", humidity),    // Use current as fallback
		TempLag1:          fmt.Sprintf("%.1f", temperature), // Use current as fallback
		HumidityLag1:      fmt.Sprintf("%.1f", humidity),    // Use current as fallback
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return "", 0, err
	}

	// Log the request for debugging
	fmt.Printf("AI API Request URL: %s\n", apiURL)
	fmt.Printf("AI API Request Body: %s\n", string(requestBody))

	// Make the HTTP POST request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var response map[string]float64
	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, err
	}

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
