package models

import "time"

type SensorData struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null"`
	Timestamp    time.Time `json:"timestamp"`
	Temperature  float32   `json:"temperature"`
	Humidity     float32   `json:"humidity"`
	SoilMoisture float32   `json:"soil_moisture"`
	IsAbnormal   bool      `json:"is_abnormal"`
}
