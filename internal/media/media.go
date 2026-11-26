package media

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnreachable = errors.New("media source unreachable")
	ErrNoAccess    = errors.New("media source rejected credentials")
)

type Connection struct {
	URL   string
	Token string
}

type Library struct {
	Key   string
	Title string
}

type Episode struct {
	Number        int
	Title         string
	Watched       bool
	LastWatchedAt *time.Time
}

type Season struct {
	Number   int
	Episodes []Episode
}

type Show struct {
	Title   string
	Seasons []Season
}

// Source reads libraries and watch history from one kind of media server.
type Source interface {
	// Libraries returns only libraries that hold shows.
	Libraries(ctx context.Context, c Connection) ([]Library, error)
	// Shows returns every show in a library with all seasons and episodes, including specials.
	Shows(ctx context.Context, c Connection, libraryKey string) ([]Show, error)
}
