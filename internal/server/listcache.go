package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
	"github.com/ReidMason/plex-ani-sync/internal/cache"
)

const defaultAniListListTTL = 5 * time.Minute

type aniListListCache struct {
	store  cache.Cache
	ttl    time.Duration
	logger *slog.Logger
}

func (c *aniListListCache) get(ctx context.Context, username string, load func() ([]anilist.ListEntry, error)) ([]anilist.ListEntry, error) {
	key := "anilist-list:" + username
	if raw, ok, err := c.store.Get(ctx, key); err != nil {
		c.logger.Warn("anilist list cache read failed", "error", err)
	} else if ok {
		var list []anilist.ListEntry
		if err := json.Unmarshal(raw, &list); err == nil {
			return list, nil
		}
	}
	list, err := load()
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(list); err == nil {
		if err := c.store.Set(ctx, key, raw, c.ttl); err != nil {
			c.logger.Warn("anilist list cache write failed", "error", err)
		}
	}
	return list, nil
}
