package domain

type TvDbID string

type AniDbID string

type AniListID string

type EpisodeOffset int

type TvDbToAniDbMapping struct {
	TvDbID  TvDbID
	AniDbID AniDbID
	// TvDbSeason is the TvDB season the episode offset applies to.
	// "a" means absolute numbering across all seasons.
	TvDbSeason    string
	EpisodeOffset EpisodeOffset
}

type AniDbToListIdMapping struct {
	AniDbID      AniDbID
	AnilistId    AniListID // TODO: Support other anime lists
	EpisodeCount int
}

type Mapping struct {
	TvDbID    TvDbID
	AnilistId AniListID // TODO: Support other anime lists
	// TvDbSeason is the TvDB season the episode offset applies to.
	// "a" means absolute numbering across all seasons.
	TvDbSeason    string
	EpisodeOffset EpisodeOffset
	EpisodeCount  int
}

type WatchStatus string

const (
	WatchStatusNotStarted WatchStatus = "not_started"
	WatchStatusInProgress WatchStatus = "in_progress"
	WatchStatusPaused     WatchStatus = "paused"
	WatchStatusDropped    WatchStatus = "dropped"
	WatchStatusCompleted  WatchStatus = "completed"
)

type AnimeStatus struct {
	AnilistId       AniListID
	PlexTitle       string
	Status          WatchStatus
	WatchedEpisodes int
	TotalEpisodes   int
}

type AnimeListEntry struct {
	AnilistId AniListID
	Title     string
	Status    WatchStatus
	Progress  int // episodes marked watched on AniList (API "progress")
}

type SyncResult struct {
	AnilistId           AniListID
	Title               string
	PlexStatus          WatchStatus // target list status from Plex
	AnilistStatus       WatchStatus // current on AniList (empty if not on list)
	AnilistProgress     int         // current watched count on AniList
	PlexWatchedEpisodes int         // target watched count from Plex
	TotalEpisodes       int         // episodes in this mapping / list slice
}
