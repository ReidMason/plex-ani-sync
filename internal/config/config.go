package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/secretbox"
)

type Config struct {
	DatabaseURL      string
	PublicBaseURL    string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	EncryptionKey    []byte
}

// Load reads configuration through getenv and names every missing variable, or the first invalid one.
func Load(getenv func(string) string) (Config, error) {
	var cfg Config
	fields := []struct {
		name string
		dst  *string
	}{
		{"DATABASE_URL", &cfg.DatabaseURL},
		{"PUBLIC_BASE_URL", &cfg.PublicBaseURL},
		{"OIDC_ISSUER_URL", &cfg.OIDCIssuerURL},
		{"OIDC_CLIENT_ID", &cfg.OIDCClientID},
		{"OIDC_CLIENT_SECRET", &cfg.OIDCClientSecret},
	}
	encodedKey := getenv("APP_ENCRYPTION_KEY")
	var missing []string
	for _, f := range fields {
		v := getenv(f.name)
		if v == "" {
			missing = append(missing, f.name)
		}
		*f.dst = v
	}
	if encodedKey == "" {
		missing = append(missing, "APP_ENCRYPTION_KEY")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s (see README)", strings.Join(missing, ", "))
	}
	u, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL must be an absolute URL (see README)")
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != secretbox.KeySize {
		return Config{}, fmt.Errorf("APP_ENCRYPTION_KEY must be %d random bytes encoded as base64, e.g. from `openssl rand -base64 32` (see README)", secretbox.KeySize)
	}
	cfg.EncryptionKey = key
	return cfg, nil
}

func (c Config) RedirectURI() string {
	return strings.TrimRight(c.PublicBaseURL, "/") + "/auth/callback"
}
