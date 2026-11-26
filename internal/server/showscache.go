package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/cache"
	"github.com/ReidMason/plex-ani-sync/internal/media"
)

const defaultShowsTTL = 5 * time.Minute

type showsCache struct {
	store  cache.Cache
	ttl    time.Duration
	logger *slog.Logger
}

// get returns the cached shows of a library, or loads and caches them. Cache failures fall back to loading.
func (c *showsCache) get(ctx context.Context, sourceID int64, libraryKey string, load func() ([]media.Show, error)) ([]media.Show, error) {
	key := fmt.Sprintf("shows:%d:%s", sourceID, libraryKey)
	if raw, ok, err := c.store.Get(ctx, key); err != nil {
		c.logger.Warn("shows cache read failed", "error", err)
	} else if ok {
		var shows []media.Show
		if err := json.Unmarshal(raw, &shows); err == nil {
			return shows, nil
		}
	}
	shows, err := load()
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(shows); err == nil {
		if err := c.store.Set(ctx, key, raw, c.ttl); err != nil {
			c.logger.Warn("shows cache write failed", "error", err)
		}
	}
	return shows, nil
}
