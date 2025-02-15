package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

// GetPredictedSoilMoisture calls the AI API to predict soil moisture based on temperature and humidity.
func GetPredictedSoilMoisture(temperature, humidity float32) (float64, error) {
	apiURL := os.Getenv("AI_URL")
	requestBody, _ := json.Marshal(map[string]float32{
		"temperature": temperature,
		"humidity":    humidity,
	})

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response map[string]float64
	json.Unmarshal(body, &response)

	return response["predicted_soil_moisture"], nil
}
