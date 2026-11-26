package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
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
	client  anilist.Client
	queries *dbgen.Queries
	clock   func() time.Time
	ttl     time.Duration
	logger  *slog.Logger
}

type cachedMedia struct {
	AniListID    int    `json:"anilist_id"`
	Episodes     *int   `json:"episodes"`
	TitleRomaji  string `json:"title_romaji"`
	TitleEnglish string `json:"title_english"`
}

// lookup returns cached metadata for the Entries, refetching missing or stale ones. When AniList can't be reached
// it falls back to whatever is cached, so a failure only removes the signal.
func (c *aniListCache) lookup(ctx context.Context, ids []int) map[int]anilist.Media {
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
	var stale []int
	fresh := map[int]bool{}
	for _, row := range rows {
		id := int(row.AnilistID)
		out[id] = anilist.Media{ID: id, Episodes: optionalInt(row.Episodes), TitleRomaji: row.TitleRomaji, TitleEnglish: row.TitleEnglish}
		fresh[id] = now.Sub(row.FetchedAt.Time) < c.ttl
	}
	for _, id := range ids {
		if !fresh[id] {
			stale = append(stale, id)
		}
	}
	if len(stale) == 0 {
		return out
	}
	fetchCtx, cancel := context.WithTimeout(ctx, aniListFetchWindow)
	defer cancel()
	fetched, err := c.client.Fetch(fetchCtx, slices.Compact(slices.Sorted(slices.Values(stale))))
	if err != nil {
		c.logger.Warn("could not fetch anilist metadata", "error", err)
		return out
	}
	c.store(ctx, fetched, now)
	for _, m := range fetched {
		out[m.ID] = m
	}
	return out
}

func (c *aniListCache) store(ctx context.Context, media []anilist.Media, now time.Time) {
	if len(media) == 0 {
		return
	}
	rows := make([]cachedMedia, len(media))
	for i, m := range media {
		rows[i] = cachedMedia{AniListID: m.ID, Episodes: m.Episodes, TitleRomaji: m.TitleRomaji, TitleEnglish: m.TitleEnglish}
	}
	payload, err := json.Marshal(rows)
	if err == nil {
		err = c.queries.UpsertAniListMedia(ctx, dbgen.UpsertAniListMediaParams{FetchedAt: pgtype.Timestamptz{Time: now, Valid: true}, Rows: payload})
	}
	if err != nil {
		c.logger.Warn("could not cache anilist metadata", "error", err)
	}
}
