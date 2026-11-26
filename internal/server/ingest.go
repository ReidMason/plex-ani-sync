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
	originCustom       = "custom"
	defaultFribbURL    = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json"
	originDataset      = "dataset"
	datasetFetchWindow = 2 * time.Minute
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

type linkRow struct {
	TMDBID int `json:"tmdb_id"`
	TVDBID int `json:"tvdb_id"`
}

// loadDatasetIfEmpty ingests Fribb when no Mappings are stored; a failed download is logged and leaves the table empty.
func loadDatasetIfEmpty(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, fribbURL string, custom []mappings.Mapping, logger *slog.Logger) error {
	if err := replaceMappings(ctx, pool, q, originCustom, custom, nil); err != nil {
		return err
	}
	n, err := q.CountMappings(ctx)
	if err != nil || n > int64(len(custom)) {
		return err
	}
	fetchCtx, cancel := context.WithTimeout(ctx, datasetFetchWindow)
	defer cancel()
	data, err := mappings.FetchFribb(fetchCtx, http.DefaultClient, fribbURL)
	if err != nil {
		logger.Warn("could not load mapping dataset", "error", err)
		return nil
	}
	return replaceMappings(ctx, pool, q, originDataset, mappings.ResolveAround(data.Rows, custom), &data.Links)
}

func replaceMappings(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, origin string, resolved []mappings.Mapping, idLinks *[]mappings.IDLink) error {
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
	var linkPayload []byte
	if idLinks != nil {
		links := make([]linkRow, 0, len(*idLinks))
		for _, l := range *idLinks {
			links = append(links, linkRow{TMDBID: l.TMDBID, TVDBID: l.TVDBID})
		}
		if linkPayload, err = json.Marshal(links); err != nil {
			return err
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := q.WithTx(tx)
	if origin == originCustom {
		err = tq.DeleteCustomMappings(ctx)
	} else {
		err = tq.DeleteDatasetMappings(ctx)
	}
	if err != nil {
		return err
	}
	if err := tq.InsertMappings(ctx, dbgen.InsertMappingsParams{Origin: origin, Rows: payload}); err != nil {
		return err
	}
	if idLinks != nil {
		if err := tq.DeleteTMDBLinks(ctx); err != nil {
			return err
		}
		if err := tq.InsertTMDBLinks(ctx, linkPayload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
