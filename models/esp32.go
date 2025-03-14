package models

import "time"

type GeolocationRequest struct {
	WifiAccessPoints []WiFiAccessPoint `json:"wifiAccessPoints"`
}

// WiFiAccessPoint: Structure for Wi-Fi data sent to Google API
type WiFiAccessPoint struct {
	MacAddress     string `json:"macAddress"`
	SignalStrength int    `json:"signalStrength"`
}

// GeolocationResponse: Output from Google Geolocation API
type GeolocationResponse struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

// DeviceLocation: Store device location in the database
type DeviceLocation struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	DeviceID  uint      `json:"user_id" gorm:"not null"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Accuracy  float64   `json:"accuracy"`
	Timestamp time.Time `json:"timestamp"`
}
