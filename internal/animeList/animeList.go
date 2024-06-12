package animeList

type AnimeList interface {
	GetAnimeList(userId int) ([]ListEntry, error)
	SearchAnime(title string) ([]Anime, error)
	GetAnime(id string) (Anime, error)
	GetAuthToken(clientId, clientSecret, redirectUri, code string) (AuthTokenResponse, error)
}

type AuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
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

type Anime struct {
	Title    string
	Year     int
	Format   string
	Sequel   AnimeRelation
	Prequel  AnimeRelation
	Synonyms []string
	Id       int
	Episodes int
}

type AnimeRelation struct {
	Id string
}
