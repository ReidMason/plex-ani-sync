package storage

import (
	"context"
	"database/sql"
	"time"

	sqlite3Storage "github.com/ReidMason/plex-ani-sync/internal/storage/sqlite3"
)

type Cache interface {
	GetCache(key string) (*string, error)
	SetCache(key string, value string, duration time.Duration) error
}

func (s Sqlite) GetCache(key string) (*string, error) {
	ctx := context.Background()
	result, err := s.queries.GetCache(ctx, key)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	expiresAt, err := parseIso8601Time(result.ExpiresAt)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now().UTC()

	if expiresAt.Before(currentTime) {
		return nil, nil
	}

	return &result.Value, nil
}

func (s Sqlite) SetCache(key string, value string, duration time.Duration) error {
	ctx := context.Background()
	expiresAt := time.Now().Add(duration)

	return s.queries.SetCache(ctx, sqlite3Storage.SetCacheParams{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}
