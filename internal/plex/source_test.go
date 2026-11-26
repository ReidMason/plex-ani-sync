package plex_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/media"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
)

func TestSourceLibrariesReturnsOnlyShowLibraries(t *testing.T) {
	srv := plexservertest.New(t)
	srv.Allow("tok")
	srv.AddLibrary("1", "movie", "Movies")
	srv.AddLibrary("2", "show", "Anime")

	got, err := plex.NewServerClient().Source("client-1").Libraries(context.Background(), media.Connection{URL: srv.URL, Token: "tok"})

	if err != nil || len(got) != 1 || got[0] != (media.Library{Key: "2", Title: "Anime"}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestSourceShowsGroupsEpisodesIntoSeasonsAcrossPages(t *testing.T) {
	srv := plexservertest.New(t)
	srv.Allow("tok")
	srv.AddLibrary("2", "show", "Anime")
	for n := 450; n >= 1; n-- {
		srv.AddEpisode("2", plexservertest.Episode{Show: "One Piece", Season: 1, Number: n, Title: "ep"})
	}
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "a", ViewCount: 1, LastViewedAt: 1700000000})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 0, Number: 1, Title: "special"})

	got, err := plex.NewServerClient().Source("client-1").Shows(context.Background(), media.Connection{URL: srv.URL, Token: "tok"}, "2")

	if err != nil || len(got) != 2 {
		t.Fatalf("got %d shows, %v", len(got), err)
	}
	onePiece, frieren := got[0], got[1]
	if len(onePiece.Seasons) != 1 || len(onePiece.Seasons[0].Episodes) != 450 || onePiece.Seasons[0].Episodes[0].Number != 1 {
		t.Errorf("one piece = %d seasons, episodes not all returned in order", len(onePiece.Seasons))
	}
	if len(frieren.Seasons) != 2 || frieren.Seasons[0].Number != 0 || frieren.Seasons[1].Number != 1 {
		t.Fatalf("frieren seasons = %+v", frieren.Seasons)
	}
	watched := frieren.Seasons[1].Episodes[0]
	if !watched.Watched || watched.LastWatchedAt == nil || !watched.LastWatchedAt.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("watched episode = %+v", watched)
	}
	if frieren.Seasons[0].Episodes[0].Watched {
		t.Error("unwatched episode reported as watched")
	}
}

func TestSourceReportsAccessAndReachabilityFailures(t *testing.T) {
	srv := plexservertest.New(t)
	src := plex.NewServerClient().Source("client-1")

	_, err := src.Shows(context.Background(), media.Connection{URL: srv.URL, Token: "bad"}, "2")
	if !errors.Is(err, media.ErrNoAccess) {
		t.Errorf("rejected token err = %v", err)
	}
	_, err = src.Libraries(context.Background(), media.Connection{URL: plexservertest.NewUnreachable(t), Token: "tok"})
	if !errors.Is(err, media.ErrUnreachable) {
		t.Errorf("unreachable err = %v", err)
	}
}
