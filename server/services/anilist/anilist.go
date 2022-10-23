package anilist

import (
	"log"
	"net/http"
	"plex-ani-sync/services/requesthandler"
)

type AnimeListService interface{}

type AnilistService struct {
	RequestHandler requesthandler.IRequestHandler
}

var _ AnimeListService = (*AnilistService)(nil)

func New(requestHandler requesthandler.IRequestHandler) *AnilistService {
	return &AnilistService{RequestHandler: requestHandler}
}

func (as AnilistService) GetAccessToken() {
	response, _ := as.makeRequest("GET", "/api/v2/oauth/token")
	log.Print(response)
}

func (as AnilistService) makeRequest(method, endpoint string) (string, error) {
	url, err := requesthandler.BuildUrl("https://anilist.co", endpoint, []requesthandler.QueryParam{})
	if err != nil {
		return "", err
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	response, err := as.RequestHandler.MakeRequest(method, url, headers)
	if err != nil {
		return "", err
	}

	return response, nil
}
