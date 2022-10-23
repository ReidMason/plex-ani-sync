package plex

import (
	"net/http"
	"plex-ani-sync/services/config"
	"plex-ani-sync/services/requesthandler"
	"plex-ani-sync/services/utils"
)

type IPlexConnection interface {
	GetSeries(libraryId string) ([]Series, error)
	GetAllLibraries() ([]Library, error)
	GetSeasons(ratingKey string) ([]Season, error)
}

type Connection struct {
	ConfigHandler  config.IConfigHandler
	RequestHandler requesthandler.IRequestHandler
}

var _ IPlexConnection = (*Connection)(nil)

func New(configHandler config.IConfigHandler, requestHandler requesthandler.IRequestHandler) *Connection {
	return &Connection{ConfigHandler: configHandler, RequestHandler: requestHandler}
}

func (pc Connection) GetSeries(libraryId string) ([]Series, error) {
	jsonData, err := pc.makeRequest("GET", "/library/sections/"+libraryId+"/all")
	if err != nil {
		var series []Series
		return series, err
	}

	series, err := utils.ParseJson[BaseResponse[SeriesMediaContainer]](jsonData)
	if err != nil {
		var series []Series
		return series, err
	}

	return series.MediaContainer.Metadata, nil
}

func (pc Connection) GetAllLibraries() ([]Library, error) {
	jsonData, err := pc.makeRequest("GET", "/library/sections")
	if err != nil {
		var libraries []Library
		return libraries, err
	}

	libraries, err := utils.ParseJson[BaseResponse[LibraryMediaContainer]](jsonData)
	if err != nil {
		var libraries []Library
		return libraries, err
	}

	return libraries.MediaContainer.Directory, nil
}

func (pc Connection) GetSeasons(ratingKey string) ([]Season, error) {
	jsonData, err := pc.makeRequest("GET", "/library/metadata/"+ratingKey+"/children")
	if err != nil {
		var seasons []Season
		return seasons, err
	}

	seasons, err := utils.ParseJson[BaseResponse[SeasonMediaContainer]](jsonData)
	if err != nil {
		var seasons []Season
		return seasons, err
	}

	return seasons.MediaContainer.Metadata, nil
}

func (pc Connection) makeRequest(method, endpoint string) (string, error) {
	cfg, err := pc.ConfigHandler.GetConfig()
	if err != nil {
		return "", err
	}

	queryParams := []requesthandler.QueryParam{
		{
			Key:   "X-Plex-Token",
			Value: cfg.Plex.Token,
		},
	}
	url, err := requesthandler.BuildUrl(cfg.Plex.BaseUrl, endpoint, queryParams)
	if err != nil {
		return "", err
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	response, err := pc.RequestHandler.MakeRequest(method, url, headers)
	if err != nil {
		return "", err
	}

	return response, nil
}

type BaseResponse[T any] struct {
	MediaContainer T
}

type SeriesMediaContainer struct {
	Metadata []Series
}

type LibraryMediaContainer struct {
	Directory []Library
}

type SeasonMediaContainer struct {
	Metadata []Season
}

type Season struct {
	RatingKey       string `json:"ratingKey"`
	Key             string `json:"key"`
	ParentRatingKey string `json:"parentRatingKey"`
	GUID            string `json:"guid"`
	ParentGUID      string `json:"parentGuid"`
	ParentStudio    string `json:"parentStudio"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	ParentKey       string `json:"parentKey"`
	ParentTitle     string `json:"parentTitle"`
	Summary         string `json:"summary"`
	Index           int    `json:"index"`
	ParentIndex     int    `json:"parentIndex"`
	ViewCount       int    `json:"viewCount"`
	SkipCount       int    `json:"skipCount"`
	LastViewedAt    int64  `json:"lastViewedAt"`
	ParentYear      int    `json:"parentYear"`
	Thumb           string `json:"thumb"`
	Art             string `json:"art"`
	ParentThumb     string `json:"parentThumb"`
	Episodes        int    `json:"leafCount"`
	EpisodesWatched int    `json:"viewedLeafCount"`
	AddedAt         int    `json:"addedAt"`
	UpdatedAt       int    `json:"updatedAt"`
}

type Library struct {
	AllowSync        bool       `json:"allowSync"`
	Art              string     `json:"art"`
	Composite        string     `json:"composite"`
	Filters          bool       `json:"filters"`
	Refreshing       bool       `json:"refreshing"`
	Thumb            string     `json:"thumb"`
	Key              string     `json:"key"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Agent            string     `json:"agent"`
	Scanner          string     `json:"scanner"`
	Language         string     `json:"language"`
	UUID             string     `json:"uuid"`
	UpdatedAt        int        `json:"updatedAt"`
	CreatedAt        int        `json:"createdAt"`
	ScannedAt        int        `json:"scannedAt"`
	Content          bool       `json:"content"`
	Directory        bool       `json:"directory"`
	ContentChangedAt int        `json:"contentChangedAt"`
	Hidden           int        `json:"hidden"`
	Location         []Location `json:"Location"`
}

type Location struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

type Series struct {
	RatingKey             string  `json:"ratingKey"`
	Key                   string  `json:"key"`
	SkipChildren          bool    `json:"skipChildren"`
	GUID                  string  `json:"guid"`
	Studio                string  `json:"studio"`
	Type                  string  `json:"type"`
	Title                 string  `json:"title"`
	TitleSort             string  `json:"titleSort"`
	Summary               string  `json:"summary"`
	Index                 int     `json:"index"`
	Rating                float64 `json:"rating"`
	ViewCount             int     `json:"viewCount"`
	SkipCount             int     `json:"skipCount"`
	LastViewedAt          int     `json:"lastViewedAt"`
	Year                  int     `json:"year"`
	Thumb                 string  `json:"thumb"`
	Art                   string  `json:"art"`
	Banner                string  `json:"banner"`
	Duration              int     `json:"duration"`
	OriginallyAvailableAt string  `json:"originallyAvailableAt"`
	LeafCount             int     `json:"leafCount"`
	ViewedLeafCount       int     `json:"viewedLeafCount"`
	ChildCount            int     `json:"childCount"`
	AddedAt               int     `json:"addedAt"`
	UpdatedAt             int     `json:"updatedAt"`
	Genre                 []Genre `json:"Genre"`
	Role                  []Role  `json:"Role"`
}

type Genre struct {
	Tag string `json:"tag"`
}

type Role struct {
	Tag string `json:"tag"`
}
