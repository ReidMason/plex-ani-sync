package config

import (
	"plex-ani-sync/filehandler"
)

type IConfigHandler interface {
	GetConfig() (Config, error)
}

type ConfigHandler struct{}

var _ IConfigHandler = (*ConfigHandler)(nil)

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func (ch ConfigHandler) GetConfig() (Config, error) {
	fh := filehandler.New[Config]()
	return fh.LoadJsonFile("data/config.json")
}

type Config struct {
	Plex PlexConfig
	Sync SyncConfig
}

type PlexConfig struct {
	BaseUrl string
	Token   string
}

type SyncConfig struct {
	DaysUntilPaused  int
	DaysUntilDropped int
}
