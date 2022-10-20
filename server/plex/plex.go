package plex

import (
	"plex-ani-sync/config"
	"plex-ani-sync/requesthandler"
	"plex-ani-sync/utils"
)

type IPlexConnection interface {
	GetAllSeries(libraryId string) []Series
	GetAllLibraries() []Library
	GetSeasons(ratingKey string) []Season
}

type Connection struct {
	ConfigHandler  config.IConfigHandler
	RequestHandler requesthandler.IRequestHandler
}

func New(configHandler config.IConfigHandler, requestHandler requesthandler.IRequestHandler) *Connection {
	return &Connection{ConfigHandler: configHandler, RequestHandler: requestHandler}
}

func (pc Connection) GetAllSeries(libraryId string) ([]Series, error) {
	jsonData, err := pc.RequestHandler.MakeRequest("GET", "/library/sections/"+libraryId+"/all")
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
	jsonData, err := pc.RequestHandler.MakeRequest("GET", "/library/sections")
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
	jsonData, err := pc.RequestHandler.MakeRequest("GET", "/library/metadata/"+ratingKey+"/children")
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
	RatingKey string
	Title     string
}
