package utils

import "fyp/models"

// CheckAbnormality determines whether the sensor data is abnormal.
func CheckAbnormality(data models.SensorData) bool {
	return data.Temperature < 20 || data.Temperature > 50 ||
		data.Humidity < 30 || data.Humidity > 90 ||
		data.SoilMoisture < 5 || data.SoilMoisture > 95
}

// GetAbnormalType returns a string describing which sensor reading is abnormal.
func GetAbnormalType(record models.SensorData) string {
	if record.Temperature < 20 || record.Temperature > 50 {
		return "Temperature"
	}
	if record.Humidity < 30 || record.Humidity > 90 {
		return "Humidity"
	}
	if record.SoilMoisture < 5 || record.SoilMoisture > 95 {
		return "Soil Moisture"
	}
	return "Unknown"
}
