package animeList

type AnimeList interface {
	GetCurrentUser(token string) (User, error)
	GetAnimeList(userId string) ([]ListEntry, error)
	SearchAnime(title string) ([]Anime, error)
	GetAnime(id AnimeId) (Anime, error)
	GetAuthToken(clientId, clientSecret, redirectUri, code string) (AuthTokenResponse, error)
}

type User struct {
	Id string
}

type AuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type AnimeId (string)
type Status string

const (
	Current   Status = "CURRENT"
	Planning  Status = "PLANNING"
	Completed Status = "COMPLETED"
	Dropped   Status = "DROPPED"
	Paused    Status = "PAUSED"
)

type ListEntry struct {
	AnimeId         AnimeId
	Status          Status
	WatchedEpisodes int
	TotalEpisodes   int
	Title           string
}

type Anime struct {
	Title    string
	Year     int
	Format   string
	Sequel   AnimeRelation
	Prequel  AnimeRelation
	Synonyms []string
	Id       AnimeId
	Episodes int
}

type AnimeRelation struct {
	AnimeId AnimeId
}
