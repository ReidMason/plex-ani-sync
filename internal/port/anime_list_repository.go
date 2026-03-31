package port

import (
	"context"
	"myapp/internal/domain"
)

type AnimeListRepository interface {
	GetAnimeList(ctx context.Context) ([]domain.AnimeListEntry, error)
	// SaveAnimeListEntry creates or updates the authenticated user's list row for the anime.
	SaveAnimeListEntry(ctx context.Context, mediaID domain.AniListID, status domain.WatchStatus, progress int) error
}
