package mapping

import (
	"fmt"
	"plex-ani-sync/services/anilist"
	"plex-ani-sync/services/plex"
	"plex-ani-sync/services/utils"
	"regexp"
	"strings"
)

type MappingFinder interface {
	GetSeriesAnilistEntries() []anilist.AnilistMedia
}

type AnilistMappingFinder struct {
	anilistService anilist.AnimeListService
}

func NewAnilistMappingFinder(anilistService anilist.AnimeListService) *AnilistMappingFinder {
	return &AnilistMappingFinder{anilistService: anilistService}
}

func (amf AnilistMappingFinder) GetSeriesAnilistEntries(series plex.Series) ([]anilist.AnilistMedia, error) {
	anilistEntries := []anilist.AnilistMedia{}
	prevousSeasonAnilistEntries := []anilist.AnilistMedia{}
	mappedEpisodes := 0
	for _, season := range series.Seasons {
		// Skip specials
		if season.Index == 0 {
			continue
		}

		if len(anilistEntries) > 0 && mappedEpisodes+season.Episodes <= anilistEntries[len(anilistEntries)-1].Episodes {
			mappedEpisodes += season.Episodes
			continue
		} else {
			mappedEpisodes = 0
		}

		seasonAnilistEntries, err := amf.findMappingForSeason(series, season, prevousSeasonAnilistEntries, anilistEntries)
		if err != nil {
			continue
		}

		anilistEntries = append(anilistEntries, seasonAnilistEntries...)
		mappedEpisodes += season.Episodes
		prevousSeasonAnilistEntries = seasonAnilistEntries

		// If it's relasing then it's either ongoing like One Piece or currently airing
		if len(seasonAnilistEntries) == 1 && seasonAnilistEntries[0].Status == "RELEASING" {
			return anilistEntries, nil
		}
	}

	return anilistEntries, nil
}

func (amf AnilistMappingFinder) findMappingForSeason(series plex.Series, season plex.Season, previousSeasonAnilistEntries []anilist.AnilistMedia, anilistEntries []anilist.AnilistMedia) ([]anilist.AnilistMedia, error) {
	seasonAnilistEntries := []anilist.AnilistMedia{}

	if len(previousSeasonAnilistEntries) > 0 {
		previousSeasonFirstAnilistEntry := previousSeasonAnilistEntries[0]
		// We want to select the latest mapped TV entry
		for _, entry := range previousSeasonAnilistEntries {
			if entry.Format == "TV" {
				previousSeasonFirstAnilistEntry = entry
			}
		}

		sequel, err := amf.getAnimeSequel(previousSeasonFirstAnilistEntry, previousSeasonFirstAnilistEntry.Format)
		if err != nil {
			return seasonAnilistEntries, err
		}
		sideStory, err := amf.getAnimeSideStory(previousSeasonFirstAnilistEntry)
		if err != nil {
			return seasonAnilistEntries, err
		}

		// Sequel gets +10 for being a sequel
		sequelScore := getMatchScore(series, season, sequel) + 10
		sideStoryScore := getMatchScore(series, season, sideStory)

		if sequel.ID != 0 && sequelScore >= 10 && sequelScore >= sideStoryScore {
			seasonAnilistEntries = append(seasonAnilistEntries, sequel)
		} else if sideStoryScore > 0 && sideStoryScore > sequelScore {
			seasonAnilistEntries = append(seasonAnilistEntries, sideStory)
		}

	} else {
		searchTerm := season.ParentTitle
		if season.Index > 1 {
			searchTerm += " " + fmt.Sprint(season.Index)
		}

		anilistSearchResults, err := amf.anilistService.SearchForAnime(searchTerm)
		if err != nil {
			return seasonAnilistEntries, err
		}

		bestMatch := findBestMatch(series, season, anilistSearchResults.Data.Page.Media)
		if bestMatch.ID != 0 {
			seasonAnilistEntries = append(seasonAnilistEntries, bestMatch)
		}
	}

	// Check if the season has been fully mapped
	mappedEpisodes := utils.SumSlice(utils.SelectSlice(seasonAnilistEntries, func(x anilist.AnilistMedia) int { return x.Episodes }))
	if mappedEpisodes >= season.Episodes {
		return seasonAnilistEntries, nil
	}

	if len(seasonAnilistEntries) > 0 {
		// Check if a side story or special could complete this season
		mappedEntry := seasonAnilistEntries[len(seasonAnilistEntries)-1]
		// first check side story
		sideStory, err := amf.getAnimeSideStory(mappedEntry)
		mappedEpisodesWithSideStory := sideStory.Episodes + mappedEpisodes
		if err == nil && mappedEpisodesWithSideStory == season.Episodes {
			return append(seasonAnilistEntries, sideStory), nil
		}

		// Then check special
		special, err := amf.getAnimeSpecial(mappedEntry)
		mappedEpisodesWithSpecial := special.Episodes + mappedEpisodes
		if err == nil && mappedEpisodesWithSpecial == season.Episodes {
			return append(seasonAnilistEntries, special), nil
		}
	}

	// We need to start looking for sequels
	for mappedEpisodes < season.Episodes && len(seasonAnilistEntries) > 0 {
		mappedEntry := seasonAnilistEntries[len(seasonAnilistEntries)-1]
		sequel, err := amf.getAnimeSequel(mappedEntry, mappedEntry.Format)
		if err != nil || sequel.ID == 0 {
			break
		}

		seasonAnilistEntries = append(seasonAnilistEntries, sequel)
		mappedEpisodes = utils.SumSlice(utils.SelectSlice(seasonAnilistEntries, func(x anilist.AnilistMedia) int { return x.Episodes }))
	}

	// The sequel may be on the previous anilist entry
	if mappedEpisodes < season.Episodes && len(anilistEntries) >= 2 {
		secondToLastAnilistEntry := anilistEntries[len(anilistEntries)-2]
		sequel, err := amf.getAnimeSequel(secondToLastAnilistEntry, secondToLastAnilistEntry.Format)
		if err != nil || sequel.ID == 0 {
			return seasonAnilistEntries, err
		}

		seasonAnilistEntries = append(seasonAnilistEntries, sequel)
	}

	return seasonAnilistEntries, nil
}

func (amf AnilistMappingFinder) getAnimeSideStory(anime anilist.AnilistMedia) (anilist.AnilistMedia, error) {
	return amf.getAnimeRelation(anime, "SIDE_STORY")
}

func (amf AnilistMappingFinder) getAnimeSpecial(anime anilist.AnilistMedia) (anilist.AnilistMedia, error) {
	sequel, err := amf.getAnimeRelation(anime, "SEQUEL")
	if err != nil || sequel.Format != "SPECIAL" {
		return anilist.AnilistMedia{}, err
	}

	return sequel, nil
}

func (amf AnilistMappingFinder) getAnimeSequel(anime anilist.AnilistMedia, wantedFormat string) (anilist.AnilistMedia, error) {
	relationType := "SEQUEL"
	sequels, err := amf.getAnimeRelations(anime, relationType)
	if err != nil {
		return anilist.AnilistMedia{}, err
	}

	acceptedFormats := []string{wantedFormat}
	// If the current season is OVA or ONA allow a sequel to be TV
	if wantedFormat == "OVA" || wantedFormat == "ONA" {
		acceptedFormats = append(acceptedFormats, "TV")
	}

	for _, sequel := range sequels {
		// The sequels format has to match the original
		// We might want to do some more complex comparisons here
		if utils.SliceContains(acceptedFormats, sequel.Format) {
			sequelData, err := amf.anilistService.GetAnimeDetails(fmt.Sprint(sequel.ID))
			return sequelData.Data.Media, err
		}
	}

	if len(sequels) > 0 {
		firstSequel := sequels[0]
		// If there's a sequel of this sequel find the next sequel of that
		if len(utils.FilterSlice(firstSequel.Relations.Edges, func(x anilist.MediaEdge) bool { return x.RelationType == relationType })) > 0 {
			return amf.getAnimeSequel(firstSequel, wantedFormat)
		}
	}

	return anilist.AnilistMedia{}, nil
}

func (amf AnilistMappingFinder) getAnimeRelation(anime anilist.AnilistMedia, relation string) (anilist.AnilistMedia, error) {
	relations, err := amf.getAnimeRelations(anime, relation)
	if len(relations) > 0 {
		return relations[0], err
	}

	return anilist.AnilistMedia{}, err
}

func (amf AnilistMappingFinder) getAnimeRelations(anime anilist.AnilistMedia, relation string) ([]anilist.AnilistMedia, error) {
	animeEntires := []anilist.AnilistMedia{}
	for i, edge := range anime.Relations.Edges {
		if edge.RelationType != relation {
			continue
		}

		sequelNode := anime.Relations.Nodes[i]

		// Get the sequel data
		sequelData, err := amf.anilistService.GetAnimeDetails(fmt.Sprint(sequelNode.ID))
		if err != nil {
			continue
		}

		animeEntires = append(animeEntires, sequelData.Data.Media)
	}

	return animeEntires, nil
}

func findBestMatch(series plex.Series, season plex.Season, anilistEntries []anilist.AnilistMedia) anilist.AnilistMedia {
	highestMatchScore := 0
	var bestMatch anilist.AnilistMedia
	for _, anilistEntry := range anilistEntries {
		score := getMatchScore(series, season, anilistEntry)
		if score > highestMatchScore || bestMatch.ID == 0 {
			highestMatchScore = score
			bestMatch = anilistEntry
		}
	}

	return bestMatch
}

func getMatchScore(series plex.Series, season plex.Season, anilistEntry anilist.AnilistMedia) int {
	score := 0
	if anilistNameMatchesPlexName(season, anilistEntry) {
		score += 30
	}

	if anilistEntry.Episodes == season.Episodes {
		score += 25
	}

	if series.Year == anilistEntry.StartDate.Year {
		score += 10
	}

	return score
}

func anilistNameMatchesPlexName(season plex.Season, anilistEntry anilist.AnilistMedia) bool {
	plexSeasonNames := getNameVariants(season, true)
	anilistTitles := append(anilistEntry.Synonyms, anilistEntry.Title.English, anilistEntry.Title.Romaji)
	for _, anilistTitle := range anilistTitles {
		anilistTitle := cleanName(anilistTitle)
		for _, plexTitle := range plexSeasonNames {
			if strings.Contains(anilistTitle, plexTitle) {
				return true
			}
		}
	}

	return false
}

func getNameVariants(season plex.Season, includeParentName bool) []string {
	results := []string{
		cleanName(season.ParentTitle + " " + season.Title),
		cleanName(season.ParentTitle + " " + fmt.Sprint(season.Index)),
	}

	if includeParentName || season.Index == 1 {
		results = append(results, cleanName(season.ParentTitle))
	}

	return results
}

func cleanName(name string) string {
	// Remove content in brackets "(TV)"
	m1 := regexp.MustCompile(`\([^)]*\)`)
	name = m1.ReplaceAllString(name, "")

	// Remove any non alphanumeric characters
	m2 := regexp.MustCompile(`[^a-zA-Z0-9]`)
	name = m2.ReplaceAllString(name, "")

	return strings.TrimSpace(strings.ToLower(name))
}
