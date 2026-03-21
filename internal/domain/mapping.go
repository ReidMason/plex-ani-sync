package domain

type TvDbID string

type AniDbID string

type AniListID string

type EpisodeOffset int

type TvDbToAniDbMapping struct {
	TvDbID        TvDbID
	AniDbID       AniDbID
	EpisodeOffset EpisodeOffset
}

type AniDbToListIdMapping struct {
	AniDbID      AniDbID
	AnilistId    AniListID // TODO: Support other anime lists
	EpisodeCount int
}

type Mapping struct {
	TvDbID        TvDbID
	AnilistId     AniListID // TODO: Support other anime lists
	EpisodeOffset EpisodeOffset
	EpisodeCount  int
}

type WatchStatus string

const (
	WatchStatusNotStarted WatchStatus = "not_started"
	WatchStatusInProgress WatchStatus = "in_progress"
	WatchStatusCompleted  WatchStatus = "completed"
)

type AnimeStatus struct {
	AnilistId AniListID
	Status    WatchStatus
}
