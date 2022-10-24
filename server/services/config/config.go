package config

import (
	"plex-ani-sync/models"

	"gorm.io/gorm"
)

type IConfigHandler interface {
	GetConfig() (models.Config, error)
	SaveConfig(newConfig models.Config) error
}

type ConfigHandler struct {
	Db *gorm.DB
}

var _ IConfigHandler = (*ConfigHandler)(nil)

func NewConfigHandler(db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{Db: db}
}

func (ch ConfigHandler) GetConfig() (models.Config, error) {
	var config models.Config
	result := ch.Db.First(&config)

	if result.Error != nil {
		return models.Config{}, result.Error
	}

	return config, nil
}

func (ch ConfigHandler) SaveConfig(newConfig models.Config) error {
	var currentConfig models.Config
	ch.Db.Find(&currentConfig)

	// Make sure the ID stays the same
	newConfig.ID = currentConfig.ID
	response := ch.Db.Model(&currentConfig).Updates(newConfig)

	return response.Error
}

func GetDefaultConfig() models.Config {
	return models.Config{
		PlexBaseUrl:          "http://localhost:32400",
		PlexToken:            "testToken123",
		SyncDaysUntilPaused:  14,
		SyncDaysUntilDropped: 31,
	}
}
