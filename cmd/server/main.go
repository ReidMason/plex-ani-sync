// Package main is the composition root. All dependency wiring happens here;
// no other package is allowed to perform DI.
package main

import (
	"context"
	"fmt"
	"log"
	"myapp/internal/adapter/anilist"
	"myapp/internal/adapter/animelists"
	"myapp/internal/adapter/plex"
	"myapp/internal/app"
	"myapp/internal/port"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const cacheDir = "data"

func envTruthy(v string) bool {
	s := strings.TrimSpace(v)
	return s != "" && !strings.EqualFold(s, "0") && !strings.EqualFold(s, "false")
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("loading .env: %v", err)
	}

	ctx := context.Background()

	mappingRepo := animelists.NewMappingRepository(cacheDir)
	mappingService := app.NewMappingService(mappingRepo)

	var plexRepo port.MediaHostRepository
	if url, token := os.Getenv("PLEX_URL"), os.Getenv("PLEX_TOKEN"); url != "" && token != "" {
		plexRepo = plex.NewRepository(url, token)
	} else {
		log.Println("PLEX_URL or PLEX_TOKEN not set — using mock Plex repository")
		plexRepo = plex.NewMockRepository()
	}

	var anilistRepo port.AnimeListRepository
	anilistMock := false
	if mock := strings.TrimSpace(os.Getenv("ANILIST_MOCK")); mock != "" && !strings.EqualFold(mock, "0") && !strings.EqualFold(mock, "false") {
		anilistMock = true
		if path := strings.TrimSpace(os.Getenv("ANILIST_MOCK_FILE")); path != "" {
			entries, err := anilist.LoadMockEntriesFile(path)
			if err != nil {
				log.Fatalf("ANILIST_MOCK_FILE: %v", err)
			}
			log.Printf("ANILIST_MOCK: loaded %d entries from %s", len(entries), path)
			anilistRepo = anilist.NewMockRepository(entries)
		} else {
			log.Println("ANILIST_MOCK: built-in Gate (TvDB 295222) fixture — pair with Plex mock (no PLEX_* in .env) or set ANILIST_MOCK_FILE for your library")
			anilistRepo = anilist.NewMockRepository(anilist.DefaultMockEntriesPlex295222())
		}
	} else {
		anilistToken := os.Getenv("ANILIST_TOKEN")
		if anilistToken == "" {
			log.Fatal("ANILIST_TOKEN not set — set ANILIST_MOCK=1 to skip the API, or see internal/adapter/anilist/repository.go for a token")
		}
		savePath := strings.TrimSpace(os.Getenv("ANILIST_SAVE_LIST"))
		anilistRepo = anilist.NewRepository(anilistToken, savePath)
	}

	cfg := app.DefaultSyncConfig()
	if v := os.Getenv("PAUSED_AFTER_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			cfg.PausedAfter = time.Duration(d) * 24 * time.Hour
		}
	}
	if v := os.Getenv("DROPPED_AFTER_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			cfg.DroppedAfter = time.Duration(d) * 24 * time.Hour
		}
	}

	syncService := app.NewSyncService(anilistRepo, plexRepo, mappingService, cfg)

	plexStatuses, err := syncService.SyncAnime(ctx)
	if err != nil {
		log.Fatalf("fetching Plex statuses: %v", err)
	}

	results, err := syncService.CompareWithAniList(ctx, plexStatuses)
	if err != nil {
		log.Fatalf("comparing with AniList: %v", err)
	}

	needsUpdate := 0
	for _, r := range results {
		if !r.NeedsAniListChange() {
			continue
		}
		curSt := string(r.AnilistStatus)
		if curSt == "" {
			curSt = "—"
		}
		targetWatched := r.TargetWatchedForAniList()
		tot := r.TotalEpisodes
		if tot <= 0 {
			fmt.Printf("~ %-45s  %s -> %s   %d -> %d\n",
				r.Title, curSt, r.PlexStatus, r.AnilistProgress, targetWatched)
		} else {
			fmt.Printf("~ %-45s  %s -> %s   %d/%d -> %d/%d\n",
				r.Title, curSt, r.PlexStatus, r.AnilistProgress, tot, targetWatched, tot)
		}
		needsUpdate++
	}
	fmt.Printf("\n%d/%d entries need updating\n", needsUpdate, len(results))

	wantApply := envTruthy(os.Getenv("ANILIST_APPLY"))
	if wantApply {
		if anilistMock {
			log.Fatal("ANILIST_APPLY is set but ANILIST_MOCK is enabled — refusing to write (mock list is not the API)")
		}
		if envTruthy(os.Getenv("ANILIST_BACKUP_BEFORE_APPLY")) {
			backupDir := strings.TrimSpace(os.Getenv("ANILIST_BACKUP_DIR"))
			if backupDir == "" {
				backupDir = cacheDir
			}
			entries, err := anilistRepo.GetAnimeList(ctx)
			if err != nil {
				log.Fatalf("AniList backup (fetch list): %v", err)
			}
			backupPath := filepath.Join(backupDir, fmt.Sprintf("anilist-backup-%s.json", time.Now().Format("20060102-150405")))
			if err := anilist.SaveAnimeListEntries(backupPath, entries); err != nil {
				log.Fatalf("AniList backup (write file): %v", err)
			}
			log.Printf("AniList: backup saved (%d entries) → %s", len(entries), backupPath)
		}

		applied, err := syncService.ApplyAniListUpdates(ctx, results)
		log.Printf("AniList: applied %d update(s)", applied)
		if err != nil {
			log.Fatalf("AniList apply: %v", err)
		}
	} else if needsUpdate > 0 {
		log.Println("dry-run: set ANILIST_APPLY=1 to push changes (~ rows). Safety: ANILIST_BACKUP_BEFORE_APPLY=1, ANILIST_SAVE_LIST (fetch snapshot)")
	}
}
