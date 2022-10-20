package config

type MockConfigHandler struct {
	MockConfig Config
}

var _ IConfigHandler = (*MockConfigHandler)(nil)

func (ch MockConfigHandler) GetConfig() (Config, error) {
	return ch.MockConfig, nil
}

func NewMock(mockConfig Config) *MockConfigHandler {
	return &MockConfigHandler{MockConfig: mockConfig}
}

func GetDefaultConfig() Config {
	return Config{
		Plex: PlexConfig{
			BaseUrl: "http://testing",
			Token:   "testToken123",
		},
		Sync: SyncConfig{
			DaysUntilPaused:  14,
			DaysUntilDropped: 31,
		},
	}
}
