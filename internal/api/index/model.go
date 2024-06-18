package api

import "github.com/ReidMason/plex-ani-sync/internal/animeList"

type IndexData struct {
	Name      string
	Libraries []string
	ListData  struct {
		Completed int
		Watching  int
		Planning  int
		Dropped   int
		Paused    int
	}
	FullList []animeList.ListEntry
}
