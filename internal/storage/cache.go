package storage

import (
	"context"
	"errors"
	"time"

	postgresStorage "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
	"github.com/jackc/pgx/v5/pgtype"
)

type Cache interface {
	Get(key string) (string, error)
	Set(key string, value string, duration time.Duration) error
}

func (p Postgres) Get(key string) (string, error) {
	ctx := context.Background()
	result, err := p.queries.GetCache(ctx, key)
	if err != nil {
		return "", err
	}

	currentTime := time.Now().UTC()

	if result.ExpiresAt.Time.Before(currentTime) {
		return "", errors.New("cache entry expired")
	}

	return result.Value, nil
}

func (p Postgres) Set(key string, value string, duration time.Duration) error {
	ctx := context.Background()
	expiresAt := time.Now().Add(duration)

	return p.queries.SetCache(ctx, postgresStorage.SetCacheParams{
		Key:   key,
		Value: value,
		ExpiresAt: pgtype.Timestamp{
			Time:  expiresAt,
			Valid: true,
		},
	})
}
