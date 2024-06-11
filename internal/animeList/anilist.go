package animeList

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/request"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

const HOST = "https://graphql.anilist.co"

type Variables map[string]interface{}

type GraphQLRequest struct {
	Variables Variables `json:"variables"`
	Query     string    `json:"query"`
}

type Anilist struct {
	client request.HttpClient
	cache  storage.Cache
	userId int
	log    *slog.Logger
}

func NewAnilist(client request.HttpClient, cache storage.Cache, logger *slog.Logger) *Anilist {
	return &Anilist{client: client, cache: cache, log: logger}
}

func (a Anilist) GetAnime(id string) (Anime, error) {
	query := `query ($anime_id: Int) {
    Media(id: $anime_id, type: ANIME) {
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
  }`

	variables := Variables{
		"anime_id": id,
	}

	var response GetAnimeResponse
	cacheKey := fmt.Sprintf("anilistGetAnime-id:%s", id)
	cachedResult, cachedErr := a.cache.GetCache(cacheKey)
	if cachedErr != nil {
		a.log.Error("Failed to get cache", slog.Any("error", cachedErr))
		cachedResult = nil
	}

	if cachedErr = json.Unmarshal([]byte(*cachedResult), &response); cachedErr != nil {
		a.log.Error("Failed to unmarshal Anilist get anime result", slog.Any("error", cachedErr))
	}

	if cachedErr != nil || cachedResult == nil {
		// Make request
		a.log.Info("Geting Anilist anime", slog.String("id", id))
		req, err := buildRequest(query, variables)
		if err != nil {
			a.log.Error("Failed to build Anilist request", slog.Any("error", err))
			return Anime{}, err
		}

		response, err = request.MakeRequest[GetAnimeResponse](a.client, req)
		if err != nil {
			a.log.Error("Failed to make Anilist request", slog.Any("error", err))
			return Anime{}, err
		}

		// Cache the response
		if resultsString, err := json.Marshal(response); err == nil {
			duration := 10_000 * time.Hour
			if err = a.cache.SetCache(cacheKey, string(resultsString), duration); err != nil {
				a.log.Error("Failed to cache Anilist get anime result", slog.Any("error", err))
			}
		} else {
			a.log.Error("Failed to marshal Anilist get anime result", slog.Any("error", err))
		}
	}

	media := response.Data.Media
	anime := Anime{
		Id:       media.ID,
		Title:    getTitle(media),
		Format:   media.Format,
		Episodes: media.Episodes,
		Synonyms: media.Synonyms,
		Year:     media.StartDate.Year,
	}

	for i, node := range media.Relations.Nodes {
		relation := media.Relations.Edges[i]
		if relation.RelationType == "SEQUEL" && anime.Sequel.Id == "" {
			anime.Sequel = AnimeRelation{
				Id: fmt.Sprint(node.ID),
			}
		}

		if relation.RelationType == "PREQUEL" && anime.Prequel.Id == "" {
			anime.Prequel = AnimeRelation{
				Id: fmt.Sprint(node.ID),
			}
		}
	}

	return anime, nil
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

	var response AnimeSearchResponse
	cacheKey := fmt.Sprintf("anilistSearchAnime-title:%s", title)
	cachedResult, cacheErr := a.cache.GetCache(cacheKey)
	if cacheErr != nil {
		a.log.Error("Failed to get cache", slog.Any("error", cacheErr))
		cachedResult = nil
	}

	if cachedResult != nil {
		if cacheErr = json.Unmarshal([]byte(*cachedResult), &response); cacheErr != nil {
			a.log.Error("Failed to unmarshal Anilist search results", slog.Any("error", cacheErr))
		}
	}

	if cacheErr != nil || cachedResult == nil {
		// Make request
		a.log.Info("Searching Anilist for anime", slog.String("title", title))
		req, err := buildRequest(query, variables)
		if err != nil {
			a.log.Error("Failed to build Anilist request", slog.Any("error", err))
			return nil, err
		}

		response, err = request.MakeRequest[AnimeSearchResponse](a.client, req)
		if err != nil {
			a.log.Error("Failed to make Anilist request", slog.Any("error", err))
			return nil, err
		}

		// Cache the response
		if resultsString, err := json.Marshal(response); err == nil {
			duration := 10_000 * time.Hour
			if err = a.cache.SetCache(cacheKey, string(resultsString), duration); err != nil {
				a.log.Error("Failed to cache Anilist search results", slog.Any("error", err))
			}
		}
	}

	results := make([]Anime, 0, len(response.Data.Page.Media))
	for _, media := range response.Data.Page.Media {
		anime := Anime{
			Id:       media.ID,
			Title:    getTitle(media),
			Format:   media.Format,
			Episodes: media.Episodes,
			Synonyms: media.Synonyms,
			Year:     media.StartDate.Year,
		}

		for i, node := range media.Relations.Nodes {
			relation := media.Relations.Edges[i]
			if relation.RelationType == "SEQUEL" && anime.Sequel.Id == "" {
				anime.Sequel = AnimeRelation{
					Id: fmt.Sprint(node.ID),
				}
			}

			if relation.RelationType == "PREQUEL" && anime.Prequel.Id == "" {
				anime.Prequel = AnimeRelation{
					Id: fmt.Sprint(node.ID),
				}
			}
		}

		results = append(results, anime)
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
		a.log.Error("Failed to build Anilist request", slog.Any("error", err))
		return nil, err
	}

	response, err := request.MakeRequest[AnimeListResponse](a.client, req)
	if err != nil {
		a.log.Error("Failed to make Anilist request", slog.Any("error", err))
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

type GetAnimeResponse struct {
	Data struct {
		Media AnimeResult `json:"Media"`
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
