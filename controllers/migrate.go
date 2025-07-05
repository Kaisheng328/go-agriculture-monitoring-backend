package controllers

import (
	"fyp/config"
	"fyp/models"

	"gorm.io/gorm"
)

// MigrateModels runs the database migrations
func MigrateModels(db *gorm.DB) {
	config.DB = db
	db.AutoMigrate(&models.User{}, &models.SensorData{}, &models.DeveloperModeSetting{})
}
