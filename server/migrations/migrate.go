package main

import (
	"log"
	"plex-ani-sync/models"
	"plex-ani-sync/services/config"
	"plex-ani-sync/services/database"
)

func main() {
	db := database.Connect()
	err := db.AutoMigrate(&models.Config{})
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print("Migration complete")
	}

	// Populate config record if it doesn't exist
	var cfg []models.Config
	db.Find(&cfg)
	if len(cfg) == 0 {
		defaultConfig := config.GetDefaultConfig()
		db.Create(&defaultConfig)
	}
}
