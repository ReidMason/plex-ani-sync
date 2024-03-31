package animeList

type AnimeList interface {
	GetAnimeList() ([]ListEntry, error)
}

type Status string

const (
	Current   Status = "CURRENT"
	Planning  Status = "PLANNING"
	Completed Status = "COMPLETED"
	Dropped   Status = "DROPPED"
	Paused    Status = "PAUSED"
)

type ListEntry struct {
	AnimeId string
	Status  Status
}
