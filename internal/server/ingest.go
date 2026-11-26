package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/mappings"
)

const (
	defaultFribbURL      = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json"
	defaultAnimeListsURL = "https://raw.githubusercontent.com/Anime-Lists/anime-lists/master/anime-list.xml"
	originDataset        = "dataset"
	datasetFetchWindow   = 2 * time.Minute
)

type mappingRow struct {
	TVDBID            int  `json:"tvdb_id"`
	TVDBSeason        int  `json:"tvdb_season"`
	FirstEpisode      int  `json:"first_episode"`
	LastEpisode       *int `json:"last_episode"`
	AniListID         int  `json:"anilist_id"`
	EntryFirstEpisode int  `json:"entry_first_episode"`
	EntryLastEpisode  *int `json:"entry_last_episode"`
	AniDBID           int  `json:"anidb_id"`
}

// loadDatasetIfEmpty ingests Fribb and Anime-Lists when no Mappings are stored; a failed download is logged and leaves the table empty.
func loadDatasetIfEmpty(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, fribbURL, animeListsURL string, logger *slog.Logger) error {
	n, err := q.CountMappings(ctx)
	if err != nil || n > 0 {
		return err
	}
	fetchCtx, cancel := context.WithTimeout(ctx, datasetFetchWindow)
	defer cancel()
	rows, err := mappings.FetchFribb(fetchCtx, http.DefaultClient, fribbURL)
	if err != nil {
		logger.Warn("could not load mapping dataset", "error", err)
		return nil
	}
	lists, err := mappings.FetchAnimeLists(fetchCtx, http.DefaultClient, animeListsURL)
	if err != nil {
		logger.Warn("could not load mapping dataset", "error", err)
		return nil
	}
	return replaceDatasetMappings(ctx, pool, q, mappings.Resolve(rows, lists))
}

func replaceDatasetMappings(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, resolved []mappings.Mapping) error {
	rows := make([]mappingRow, 0, len(resolved))
	for _, m := range resolved {
		rows = append(rows, mappingRow{
			TVDBID: m.TVDBID, TVDBSeason: m.Season, FirstEpisode: m.Episodes.First, LastEpisode: m.Episodes.Last,
			AniListID: m.AniListID, EntryFirstEpisode: m.Entry.First, EntryLastEpisode: m.Entry.Last, AniDBID: m.AniDBID,
		})
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := q.WithTx(tx)
	if err := tq.DeleteDatasetMappings(ctx); err != nil {
		return err
	}
	if err := tq.InsertMappings(ctx, dbgen.InsertMappingsParams{Origin: originDataset, Rows: payload}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
