package models

import "time"

// DeveloperModeSetting stores the state of developer mode
type DeveloperModeSetting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	IsEnabled bool      `json:"is_enabled" gorm:"default:false"`
	StartTime time.Time `json:"start_time"`
}
