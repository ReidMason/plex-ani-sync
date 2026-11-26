package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/mappings"
)

const (
	originCustom         = "custom"
	defaultFribbURL      = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json"
	defaultAnimeListsURL = "https://raw.githubusercontent.com/Anime-Lists/anime-lists/master/anime-list.xml"
	originDataset        = "dataset"
	datasetFetchWindow   = 2 * time.Minute

	defaultDatasetRefreshInterval = 24 * time.Hour
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

// datasetSync keeps the stored Mappings in step with the datasets and Custom Mappings.
type datasetSync struct {
	pool          *pgxpool.Pool
	queries       *dbgen.Queries
	fribbURL      string
	animeListsURL string
	custom        []mappings.Mapping
	logger        *slog.Logger
}

// loadAtStartup stores the Custom Mappings and ingests the datasets when none are stored or the Custom Mappings changed, so stale overlapping dataset rows go. A failed download is logged and leaves the stored rows untouched.
func (d datasetSync) loadAtStartup(ctx context.Context) error {
	changed, err := d.customChanged(ctx)
	if err != nil {
		return err
	}
	if err := d.replaceCustom(ctx); err != nil {
		return err
	}
	n, err := d.queries.CountMappings(ctx)
	if err != nil || (!changed && n > int64(len(d.custom))) {
		return err
	}
	if err := d.refresh(ctx); err != nil {
		d.logger.Warn("could not load mapping dataset", "error", err)
	}
	return nil
}

// refreshEvery reloads the datasets on each tick until ctx ends; a failed refresh is logged and leaves the stored rows untouched.
func (d datasetSync) refreshEvery(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.refresh(ctx); err != nil && ctx.Err() == nil {
				d.logger.Warn("could not refresh mapping dataset", "error", err)
			}
		}
	}
}

func (d datasetSync) customChanged(ctx context.Context) (bool, error) {
	stored, err := d.queries.ListCustomMappings(ctx)
	if err != nil {
		return false, err
	}
	storedRows := make([]mappingRow, 0, len(stored))
	for _, r := range stored {
		storedRows = append(storedRows, mappingRow{
			TVDBID: int(r.TvdbID), TVDBSeason: int(r.TvdbSeason), FirstEpisode: int(r.FirstEpisode), LastEpisode: optionalInt(r.LastEpisode),
			AniListID: int(r.AnilistID), EntryFirstEpisode: int(r.EntryFirstEpisode), EntryLastEpisode: optionalInt(r.EntryLastEpisode),
		})
	}
	return !reflect.DeepEqual(storedRows, toMappingRows(d.custom)), nil
}

// refresh downloads every dataset first, then swaps all dataset rows in one transaction.
func (d datasetSync) refresh(ctx context.Context) error {
	fetchCtx, cancel := context.WithTimeout(ctx, datasetFetchWindow)
	defer cancel()
	data, err := mappings.FetchFribb(fetchCtx, http.DefaultClient, d.fribbURL)
	if err != nil {
		return err
	}
	lists, err := mappings.FetchAnimeLists(fetchCtx, http.DefaultClient, d.animeListsURL)
	if err != nil {
		return err
	}
	return d.replaceDataset(ctx, mappings.Resolve(data.Rows, lists, d.custom), data.Links)
}

func (d datasetSync) replaceCustom(ctx context.Context) error {
	return d.inTx(ctx, func(tq *dbgen.Queries) error {
		if err := tq.DeleteCustomMappings(ctx); err != nil {
			return err
		}
		return insertMappings(ctx, tq, originCustom, d.custom)
	})
}

func (d datasetSync) replaceDataset(ctx context.Context, resolved []mappings.Mapping, idLinks []mappings.IDLink) error {
	links := make([]linkRow, 0, len(idLinks))
	for _, l := range idLinks {
		links = append(links, linkRow{TMDBID: l.TMDBID, TVDBID: l.TVDBID})
	}
	linkPayload, err := json.Marshal(links)
	if err != nil {
		return err
	}
	return d.inTx(ctx, func(tq *dbgen.Queries) error {
		if err := tq.DeleteDatasetMappings(ctx); err != nil {
			return err
		}
		if err := insertMappings(ctx, tq, originDataset, resolved); err != nil {
			return err
		}
		if err := tq.DeleteTMDBLinks(ctx); err != nil {
			return err
		}
		return tq.InsertTMDBLinks(ctx, linkPayload)
	})
}

func (d datasetSync) inTx(ctx context.Context, fn func(*dbgen.Queries) error) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(d.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func toMappingRows(resolved []mappings.Mapping) []mappingRow {
	rows := make([]mappingRow, 0, len(resolved))
	for _, m := range resolved {
		rows = append(rows, mappingRow{
			TVDBID: m.TVDBID, TVDBSeason: m.Season, FirstEpisode: m.Episodes.First, LastEpisode: m.Episodes.Last,
			AniListID: m.AniListID, EntryFirstEpisode: m.Entry.First, EntryLastEpisode: m.Entry.Last, AniDBID: m.AniDBID,
		})
	}
	return rows
}

func insertMappings(ctx context.Context, tq *dbgen.Queries, origin string, resolved []mappings.Mapping) error {
	payload, err := json.Marshal(toMappingRows(resolved))
	if err != nil {
		return err
	}
	return tq.InsertMappings(ctx, dbgen.InsertMappingsParams{Origin: origin, Rows: payload})
}
