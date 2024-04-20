package animeList

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/request"
	"golang.org/x/exp/slog"
)

const HOST = "https://graphql.anilist.co"

type Variables map[string]interface{}

type GraphQLRequest struct {
	Variables Variables `json:"variables"`
	Query     string    `json:"query"`
}

type Anilist struct {
	client request.HttpClient
	userId int
}

func NewAnilist(client request.HttpClient, userId int) *Anilist {
	return &Anilist{client: client, userId: userId}
}

func (a Anilist) SearchAnime(title string) ([]Anime, error) {
	query := `query($title: String) {
    Page(perPage: 10) {
      media(search: $title, type: ANIME, sort: SEARCH_MATCH) {
        id
        format
        episodes
        synonyms
        status
        endDate {
          year
          month
          day
        }
        startDate {
          year
          month
          day
        }
        title {
          english
          romaji
        }
        relations {
          edges {
            relationType
          }
          nodes {
            id
            format
            episodes
            endDate {
              year
              month
              day
            }
            startDate {
              year
              month
              day
            }
          }
        }
      }
    }
  }`

	variables := Variables{
		"title": title,
	}

	req, err := buildRequest(query, variables)
	if err != nil {
		slog.Error("Failed to build Anilist request", slog.Any("error", err))
		return nil, err
	}

	response, err := request.MakeRequest[AnimeSearchResponse](a.client, req)
	if err != nil {
		slog.Error("Failed to make Anilist request", slog.Any("error", err))
		return nil, err
	}

	results := make([]Anime, 0, len(response.Data.Page.Media))
	for _, media := range response.Data.Page.Media {
		results = append(results, Anime{
			Id:       media.ID,
			Title:    getTitle(media),
			Format:   media.Format,
			Episodes: media.Episodes,
		})
	}

	return results, nil
}

func getTitle(title AnimeResult) string {
	if title.Title.English != "" {
		return title.Title.English
	}

	return title.Title.Romaji
}

func (a Anilist) GetAnimeList() ([]ListEntry, error) {
	query := `query($user_id: Int) {
    MediaListCollection(userId: $user_id, type: ANIME) {
      lists {
        name
        status
        isCustomList
        entries {
          mediaId
          progress
        }
      }
    }
  }`

	variables := Variables{
		"user_id": a.userId,
	}

	req, err := buildRequest(query, variables)
	if err != nil {
		slog.Error("Failed to build Anilist request", slog.Any("error", err))
		return nil, err
	}

	response, err := request.MakeRequest[AnimeListResponse](a.client, req)
	if err != nil {
		slog.Error("Failed to make Anilist request", slog.Any("error", err))
		return nil, err
	}

	listEntries := make([]ListEntry, 0)

	for _, list := range response.Data.MediaListCollection.Lists {
		if list.Status == "" {
			continue
		}

		for _, entry := range list.Entries {
			listEntries = append(listEntries, ListEntry{
				AnimeId: fmt.Sprint(entry.MediaID),
				Status:  Status(list.Status),
			})
		}
	}

	return listEntries, nil
}

func buildRequest(query string, variables Variables) (*http.Request, error) {
	body := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	requestBody := bytes.NewBuffer(jsonBody)

	req, err := http.NewRequest("POST", HOST, requestBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

type AnimeSearchResponse struct {
	Data struct {
		Page struct {
			Media []AnimeResult `json:"media"`
		} `json:"Page"`
	} `json:"data"`
}

type AnimeResult struct {
	Title struct {
		English string `json:"english"`
		Romaji  string `json:"romaji"`
	} `json:"title"`
	Format    string `json:"format"`
	Status    string `json:"status"`
	Relations struct {
		Edges []struct {
			RelationType string `json:"relationType"`
		} `json:"edges"`
		Nodes []struct {
			Format    string       `json:"format"`
			EndDate   DateResponse `json:"endDate"`
			StartDate DateResponse `json:"startDate"`
			ID        int          `json:"id"`
			Episodes  int          `json:"episodes"`
		} `json:"nodes"`
	} `json:"relations"`
	Synonyms  []string     `json:"synonyms"`
	EndDate   DateResponse `json:"endDate"`
	StartDate DateResponse `json:"startDate"`
	ID        int          `json:"id"`
	Episodes  int          `json:"episodes"`
}

type DateResponse struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type AnimeListResponse struct {
	Data struct {
		MediaListCollection struct {
			Lists []AnilistList `json:"lists"`
		} `json:"MediaListCollection"`
	} `json:"data"`
}

type AnilistList struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Entries []struct {
		MediaID  int `json:"mediaId"`
		Progress int `json:"progress"`
	} `json:"entries"`
	IsCustomList bool `json:"isCustomList"`
}
