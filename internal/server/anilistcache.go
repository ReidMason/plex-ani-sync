package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

const (
	defaultAniListMetadataTTL = 24 * time.Hour
	aniListFetchWindow        = 10 * time.Second
)

type aniListCache struct {
	client    *anilist.Client
	queries   *dbgen.Queries
	clock     func() time.Time
	ttl       time.Duration
	pollEvery time.Duration
	logger    *slog.Logger

	mu      sync.Mutex
	pending map[int]struct{}
	kick    chan struct{}
}

func newAniListCache(client *anilist.Client, queries *dbgen.Queries, clock func() time.Time, ttl, pollEvery time.Duration, logger *slog.Logger) *aniListCache {
	return &aniListCache{client: client, queries: queries, clock: clock, ttl: ttl, pollEvery: pollEvery, logger: logger, pending: map[int]struct{}{}, kick: make(chan struct{}, 1)}
}

type cachedMedia struct {
	AniListID    int    `json:"anilist_id"`
	Episodes     *int   `json:"episodes"`
	TitleRomaji  string `json:"title_romaji"`
	TitleEnglish string `json:"title_english"`
	Missing      bool   `json:"missing"`
}

// mediaFor serves cached rows, including stale ones, and queues stale IDs for the background worker.
func (c *aniListCache) mediaFor(ctx context.Context, ids []int) map[int]anilist.Media {
	out := map[int]anilist.Media{}
	if len(ids) == 0 {
		return out
	}
	wanted := make([]int64, len(ids))
	for i, id := range ids {
		wanted[i] = int64(id)
	}
	rows, err := c.queries.ListAniListMedia(ctx, wanted)
	if err != nil {
		c.logger.Warn("could not read anilist cache", "error", err)
	}
	now := c.clock()
	fresh := map[int]bool{}
	for _, row := range rows {
		id := int(row.AnilistID)
		fresh[id] = now.Sub(row.FetchedAt.Time) < c.ttl
		if row.Missing {
			continue
		}
		out[id] = anilist.Media{ID: id, Episodes: optionalInt(row.Episodes), TitleRomaji: row.TitleRomaji, TitleEnglish: row.TitleEnglish}
	}
	c.enqueue(slices.DeleteFunc(slices.Clone(ids), func(id int) bool { return fresh[id] }))
	return out
}

func (c *aniListCache) enqueue(ids []int) {
	if len(ids) == 0 {
		return
	}
	c.mu.Lock()
	for _, id := range ids {
		c.pending[id] = struct{}{}
	}
	c.mu.Unlock()
	select {
	case c.kick <- struct{}{}:
	default:
	}
}

func (c *aniListCache) nextBatch() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	batch := slices.Sorted(maps.Keys(c.pending))
	return batch[:min(len(batch), anilist.MaxBatch)]
}

func (c *aniListCache) settle(ids []int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.pending, id)
	}
}

// run fetches queued IDs until ctx ends, one batch per slot the client allows.
func (c *aniListCache) run(ctx context.Context) {
	ticker := time.NewTicker(c.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.kick:
		case <-ticker.C:
		}
		c.drain(ctx)
	}
}

func (c *aniListCache) drain(ctx context.Context) {
	for ctx.Err() == nil {
		batch := c.nextBatch()
		if len(batch) == 0 {
			return
		}
		fetchCtx, cancel := context.WithTimeout(ctx, aniListFetchWindow)
		fetched, err := c.client.Fetch(fetchCtx, batch)
		cancel()
		var throttled *anilist.ThrottledError
		if errors.As(err, &throttled) {
			return
		}
		if err != nil {
			c.logger.Warn("could not fetch anilist metadata", "error", err)
			return
		}
		if c.store(ctx, batch, fetched, c.clock()) {
			c.settle(batch)
		}
	}
}

// store caches what AniList returned and remembers requested IDs it didn't know as missing; it reports whether the write succeeded.
func (c *aniListCache) store(ctx context.Context, requested []int, media []anilist.Media, now time.Time) bool {
	rows := make([]cachedMedia, 0, len(requested))
	returned := map[int]bool{}
	for _, m := range media {
		returned[m.ID] = true
		rows = append(rows, cachedMedia{AniListID: m.ID, Episodes: m.Episodes, TitleRomaji: m.TitleRomaji, TitleEnglish: m.TitleEnglish})
	}
	for _, id := range requested {
		if !returned[id] {
			rows = append(rows, cachedMedia{AniListID: id, Missing: true})
		}
	}
	if len(rows) == 0 {
		return true
	}
	payload, err := json.Marshal(rows)
	if err == nil {
		err = c.queries.UpsertAniListMedia(ctx, dbgen.UpsertAniListMediaParams{FetchedAt: pgtype.Timestamptz{Time: now, Valid: true}, Rows: payload})
	}
	if err != nil {
		c.logger.Warn("could not cache anilist metadata", "error", err)
		return false
	}
	return true
}
