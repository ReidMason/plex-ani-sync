package config_test

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/config"
)

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":       "postgres://db",
		"PUBLIC_BASE_URL":    "https://plex-ani-sync.example.com/",
		"OIDC_ISSUER_URL":    "https://auth.example.com/application/o/plex-ani-sync/",
		"OIDC_CLIENT_ID":     "client",
		"OIDC_CLIENT_SECRET": "secret",
		"APP_ENCRYPTION_KEY": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)),
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	env := validEnv()
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://db" || cfg.OIDCClientID != "client" || cfg.OIDCClientSecret != "secret" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if got, want := cfg.RedirectURI(), "https://plex-ani-sync.example.com/auth/callback"; got != want {
		t.Errorf("RedirectURI = %q, want %q", got, want)
	}
}

func TestLoadNamesEachMissingVariable(t *testing.T) {
	for name := range validEnv() {
		t.Run(name, func(t *testing.T) {
			env := validEnv()
			delete(env, name)
			_, err := config.Load(func(k string) string { return env[k] })
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "README") {
				t.Errorf("error %q should name %s and reference the README", err, name)
			}
		})
	}
}

func TestLoadRejectsInvalidPublicBaseURL(t *testing.T) {
	env := validEnv()
	env["PUBLIC_BASE_URL"] = "not a url"
	_, err := config.Load(func(k string) string { return env[k] })
	if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL") {
		t.Errorf("err = %v, want one naming PUBLIC_BASE_URL", err)
	}
}

func TestLoadReportsAllMissingVariables(t *testing.T) {
	env := validEnv()
	delete(env, "DATABASE_URL")
	delete(env, "OIDC_CLIENT_ID")
	delete(env, "OIDC_CLIENT_SECRET")
	_, err := config.Load(func(k string) string { return env[k] })
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"DATABASE_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "README"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q should mention %s", err, name)
		}
	}
	if strings.Contains(err.Error(), "PUBLIC_BASE_URL") {
		t.Errorf("error %q names a variable that is set", err)
	}
}

func TestLoadDecodesEncryptionKey(t *testing.T) {
	env := validEnv()
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cfg.EncryptionKey, bytes.Repeat([]byte{7}, 32)) {
		t.Errorf("EncryptionKey = %x", cfg.EncryptionKey)
	}
}

func TestLoadRejectsMalformedEncryptionKey(t *testing.T) {
	for _, bad := range []string{"not base64!!", base64.StdEncoding.EncodeToString([]byte("too short"))} {
		env := validEnv()
		env["APP_ENCRYPTION_KEY"] = bad
		_, err := config.Load(func(k string) string { return env[k] })
		if err == nil || !strings.Contains(err.Error(), "APP_ENCRYPTION_KEY") {
			t.Errorf("%q: err = %v, want one naming APP_ENCRYPTION_KEY", bad, err)
		}
	}
}

func TestLoadDefaultsAniListRateAndAcceptsOverride(t *testing.T) {
	env := validEnv()
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil || cfg.AniListRequestsPerMinute != 30 {
		t.Fatalf("default: rate = %d, err = %v", cfg.AniListRequestsPerMinute, err)
	}

	env["ANILIST_REQUESTS_PER_MINUTE"] = "60"
	cfg, err = config.Load(func(k string) string { return env[k] })
	if err != nil || cfg.AniListRequestsPerMinute != 60 {
		t.Errorf("override: rate = %d, err = %v", cfg.AniListRequestsPerMinute, err)
	}
}

func TestLoadRejectsInvalidAniListRate(t *testing.T) {
	for _, v := range []string{"0", "-5", "fast"} {
		env := validEnv()
		env["ANILIST_REQUESTS_PER_MINUTE"] = v
		_, err := config.Load(func(k string) string { return env[k] })
		if err == nil || !strings.Contains(err.Error(), "ANILIST_REQUESTS_PER_MINUTE") {
			t.Errorf("%q: err = %v", v, err)
		}
	}
}
