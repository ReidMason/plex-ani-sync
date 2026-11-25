package config

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	DatabaseURL      string
	PublicBaseURL    string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
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
	var missing []string
	for _, f := range fields {
		v := getenv(f.name)
		if v == "" {
			missing = append(missing, f.name)
		}
		*f.dst = v
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s (see README)", strings.Join(missing, ", "))
	}
	u, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL must be an absolute URL (see README)")
	}
	return cfg, nil
}

func (c Config) RedirectURI() string {
	return strings.TrimRight(c.PublicBaseURL, "/") + "/auth/callback"
}
