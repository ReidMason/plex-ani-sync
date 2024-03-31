package mediaHost

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/ReidMason/plex-ani-sync/internal/request"
)

type Plex struct {
	client  request.HttpClient
	hostUrl *url.URL
	token   string
}

func NewPlex() *Plex {
	return &Plex{}
}

func (p *Plex) Initialize(token, host string, client request.HttpClient) error {
	hostUrl, err := url.Parse(host)
	if err != nil {
		return err
	}

	p.token = token
	p.hostUrl = hostUrl
	p.client = client

	return nil
}

func buildRequest(method, url, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("X-Plex-Token", token)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("accept", "application/json")

	return req, nil
}

func (p Plex) buildHostUrl(path string) (string, error) {
	return url.JoinPath(p.hostUrl.String(), path)
}

func (p Plex) GetSeries(libraryKey string) ([]PlexSeries, error) {
	url, err := p.buildHostUrl(fmt.Sprintf("/library/sections/%s/all", libraryKey))
	if err != nil {
		return nil, err
	}

	req, err := buildRequest("GET", url, p.token)
	if err != nil {
		return nil, err
	}

	response, err := request.MakeRequest[PlexResponse[MetadataMediaContainer[[]PlexSeries]]](p.client, req)
	if err != nil {
		return nil, err
	}

	return response.MediaContainer.Metadata, nil
}

func (p Plex) GetSeasons(seriesKey string) ([]PlexSeason, error) {
	url, err := p.buildHostUrl(fmt.Sprintf("/library/metadata/%s/children", seriesKey))
	if err != nil {
		return nil, err
	}

	req, err := buildRequest("GET", url, p.token)
	if err != nil {
		return nil, err
	}

	response, err := request.MakeRequest[PlexResponse[MetadataMediaContainer[[]PlexSeason]]](p.client, req)
	if err != nil {
		return nil, err
	}

	return response.MediaContainer.Metadata, nil
}

func (p Plex) GetEpisodes(seasonKey string) ([]PlexEpisode, error) {
	url, err := p.buildHostUrl(fmt.Sprintf("/library/metadata/%s/children", seasonKey))
	if err != nil {
		return nil, err
	}

	req, err := buildRequest("GET", url, p.token)
	if err != nil {
		return nil, err
	}

	response, err := request.MakeRequest[PlexResponse[MetadataMediaContainer[[]PlexEpisode]]](p.client, req)
	if err != nil {
		return nil, err
	}

	return response.MediaContainer.Metadata, nil
}

func (p Plex) GetLibraries() ([]Library, error) {
	url, err := p.buildHostUrl("library/sections")
	if err != nil {
		return nil, err
	}

	req, err := buildRequest("GET", url, p.token)
	if err != nil {
		return nil, err
	}

	response, err := request.MakeRequest[PlexResponse[DirectoryMediaContainer[[]Library]]](p.client, req)
	if err != nil {
		return nil, err
	}

	return response.MediaContainer.Directory, nil
}

func (p Plex) GetCurrentUser() (PlexUser, error) {
	var plexUser PlexUser
	req, err := buildRequest("GET", "https://plex.tv/api/v2/user", p.token)
	if err != nil {
		log.Println("Failed to build request for GetCurrentUser", err)
		return plexUser, err
	}

	return request.MakeRequest[PlexUser](p.client, req)
}

type PlexResponse[T any] struct {
	MediaContainer T `json:"MediaContainer"`
}

// TODO: Add the rest of the fields
type PlexSeries struct {
	RatingKey       string `json:"ratingKey"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	TitleSort       string `json:"titleSort"`
	ViewCount       int    `json:"viewCount"`
	LastViewedAt    int    `json:"lastViewedAt"`
	LeafCount       int    `json:"leafCount"`
	ViewedLeafCount int    `json:"viewedLeafCount"`
	ChildCount      int    `json:"childCount"`
}

type PlexSeason struct {
	ParentTitle     string `json:"parentTitle"`
	ParentRatingKey string `json:"parentRatingKey"`
	Summary         string `json:"summary"`
	Guid            string `json:"guid"`
	ParentGuid      string `json:"parentGuid"`
	ParentStudio    string `json:"parentStudio"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	RatingKey       string `json:"ratingKey"`
	ParentKey       string `json:"parentKey"`
	ParentThumb     string `json:"parentThumb"`
	Key             string `json:"key"`
	TitleSort       string `json:"titleSort"`
	Art             string `json:"art"`
	Thumb           string `json:"thumb"`
	Year            int    `json:"year"`
	ParentIndex     int    `json:"parentIndex"`
	UpdatedAt       int    `json:"updatedAt"`
	LeafCount       int    `json:"leafCount"`
	ViewedLeafCount int    `json:"viewedLeafCount"`
	AddedAt         int    `json:"addedAt"`
	Index           int    `json:"index"`
}

type PlexEpisode struct {
	OriginalTitle         string  `json:"originalTitle"`
	Art                   string  `json:"art"`
	ContentRating         string  `json:"contentRating"`
	ParentRatingKey       string  `json:"parentRatingKey"`
	GrandparentRatingKey  string  `json:"grandparentRatingKey"`
	Guid                  string  `json:"guid"`
	ParentGuid            string  `json:"parentGuid"`
	GrandparentGuid       string  `json:"grandparentGuid"`
	Type                  string  `json:"type"`
	Title                 string  `json:"title"`
	GrandparentKey        string  `json:"grandparentKey"`
	ParentKey             string  `json:"parentKey"`
	GrandparentTitle      string  `json:"grandparentTitle"`
	ParentTitle           string  `json:"parentTitle"`
	AudienceRatingImage   string  `json:"audienceRatingImage"`
	RatingKey             string  `json:"ratingKey"`
	Summary               string  `json:"summary"`
	OriginallyAvailableAt string  `json:"originallyAvailableAt"`
	GrandparentArt        string  `json:"grandparentArt"`
	GrandparentThumb      string  `json:"grandparentThumb"`
	Key                   string  `json:"key"`
	Thumb                 string  `json:"thumb"`
	ParentYear            int     `json:"parentYear"`
	AudienceRating        float64 `json:"audienceRating"`
	ParentIndex           int     `json:"parentIndex"`
	Duration              int     `json:"duration"`
	Index                 int     `json:"index"`
	AddedAt               int     `json:"addedAt"`
	UpdatedAt             int     `json:"updatedAt"`
	SkipParent            bool    `json:"skipParent"`
}

type MetadataMediaContainer[T any] struct {
	Metadata  T      `json:"metadata"`
	Title1    string `json:"title1"`
	Size      int    `json:"size"`
	AllowSync bool   `json:"allowSync"`
}

type DirectoryMediaContainer[T any] struct {
	Directory T      `json:"Directory"`
	Title1    string `json:"title1"`
	Size      int    `json:"size"`
	AllowSync bool   `json:"allowSync"`
}

type Library struct {
	Scanner          string     `json:"scanner"`
	Type             string     `json:"type"`
	Art              string     `json:"art"`
	UUID             string     `json:"uuid"`
	Language         string     `json:"language"`
	Thumb            string     `json:"thumb"`
	Key              string     `json:"key"`
	Composite        string     `json:"composite"`
	Title            string     `json:"title"`
	Agent            string     `json:"agent"`
	Locations        []Location `json:"Location"`
	ContentChangedAt int        `json:"contentChangedAt"`
	Hidden           int        `json:"hidden"`
	UpdatedAt        int        `json:"updatedAt"`
	CreatedAt        int        `json:"createdAt"`
	ScannedAt        int        `json:"scannedAt"`
	Refreshing       bool       `json:"refreshing"`
	Directory        bool       `json:"directory"`
	Content          bool       `json:"content"`
	Filters          bool       `json:"filters"`
	AllowSync        bool       `json:"allowSync"`
}

type Location struct {
	Path string `json:"path"`
	ID   int    `json:"id"`
}

type PlexUser struct {
	Locale            *string      `json:"locale"`
	Thumb             string       `json:"thumb"`
	Title             string       `json:"title"`
	Country           string       `json:"country"`
	ScrobbleTypes     string       `json:"scrobbleTypes"`
	FriendlyName      string       `json:"friendlyName"`
	UUID              string       `json:"uuid"`
	MailingListStatus string       `json:"mailingListStatus"`
	AuthToken         string       `json:"authToken"`
	Email             string       `json:"email"`
	Username          string       `json:"username"`
	Subscription      Subscription `json:"subscription"`
	ID                int          `json:"id"`
	JoinedAt          int          `json:"joinedAt"`
	Confirmed         bool         `json:"confirmed"`
	MailingListActive bool         `json:"mailingListActive"`
	Protected         bool         `json:"protected"`
	HasPassword       bool         `json:"hasPassword"`
	EmailOnlyAuth     bool         `json:"emailOnlyAuth"`
}

type Subscription struct {
	SubscribedAt   string `json:"subscribedAt"`
	Status         string `json:"status"`
	PaymentService string `json:"paymentService"`
	Plan           string `json:"plan"`
	Active         bool   `json:"active"`
}
