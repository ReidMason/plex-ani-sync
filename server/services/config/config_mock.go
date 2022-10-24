package config

import (
	"plex-ani-sync/models"
)

type MockConfigHandler struct {
	MockConfig models.Config
}

var _ IConfigHandler = (*MockConfigHandler)(nil)

func (ch MockConfigHandler) GetConfig() (models.Config, error) {
	return ch.MockConfig, nil
}

func (ch MockConfigHandler) SaveConfig(newConfig models.Config) error {
	return nil
}
func NewMock(mockConfig models.Config) *MockConfigHandler {
	return &MockConfigHandler{MockConfig: mockConfig}
}
