package domain

type MediaHostAnimeTitle string

type MediaHostAnime struct {
	ID      TvDbID
	Title   MediaHostAnimeTitle
	Seasons []MediaHostSeason
}

type MediaHostSeason struct {
	Number   MediaHostSeasonNumber
	Episodes []MediaHostEpisode
}

type MediaHostSeasonNumber int
type MediaHostEpisode struct {
	Number  MediaHostEpisodeNumber
	Watched bool
}

type MediaHostEpisodeNumber int
