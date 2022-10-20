package requesthandler

type MockRequestHandler struct {
	ResponseData string
	ResponseError error
}

func NewMock(responseData string, responseError error) *MockRequestHandler {
	return &MockRequestHandler{ResponseData: responseData, ResponseError: responseError}
}

var _ IRequestHandler = (*MockRequestHandler)(nil)

func (rh MockRequestHandler) MakeRequest(method, endpoint string) (string, error) {
	return rh.ResponseData, rh.ResponseError
}

type MockResponseData struct {
	LibrariesResponse, SeriesResponse, SeasonsResponse string
}

func GetMockResponseData() MockResponseData {
	return MockResponseData {
		LibrariesResponse: `{
"MediaContainer": {
"size": 2,
"allowSync": false,
"title1": "Plex Library",
"Directory": [
{
"allowSync": true,
"art": "/:/resources/movie-fanart.jpg",
"composite": "/library/sections/3/composite/1630318148",
"filters": true,
"refreshing": false,
"thumb": "/:/resources/movie.png",
"key": "3",
"type": "movie",
"title": "Test library",
"agent": "tv.plex.agents.movie",
"scanner": "Plex Movie",
"language": "en-US",
"uuid": "b2d679ea-85a6-41e5-bf4e-a1c450403f92",
"updatedAt": 1630318121,
"createdAt": 1560115682,
"scannedAt": 1630318148,
"content": true,
"directory": true,
"contentChangedAt": 1970287,
"hidden": 0,
"Location": [
{
"id": 3,
"path": "/data/Anime movies"
}
]
},
{
"allowSync": true,
"art": "/:/resources/show-fanart.jpg",
"composite": "/library/sections/1/composite/1661819643",
"filters": true,
"refreshing": false,
"thumb": "/:/resources/show.png",
"key": "1",
"type": "show",
"title": "Test library title",
"agent": "tv.plex.agents.series",
"scanner": "Plex TV Series",
"language": "en-US",
"uuid": "ab4411fa-4061-4dc6-b5fb-1bd94cb7c209",
"updatedAt": 1660522114,
"createdAt": 1560112075,
"scannedAt": 1661819643,
"content": true,
"directory": true,
"contentChangedAt": 2416993,
"hidden": 0,
"Location": [
{
"id": 1,
"path": "/data/Anime"
}
]
}
]
}
}`,
SeasonsResponse: `{
"MediaContainer": {
"size": 1,
"allowSync": true,
"art": "/library/metadata/24581/art/1613085732",
"banner": "/library/metadata/24581/banner/1613085732",
"identifier": "com.plexapp.plugins.library",
"key": "24581",
"librarySectionID": 1,
"librarySectionTitle": "Anime",
"librarySectionUUID": "ab4411fa-4061-4dc6-b5fb-1bd94cb7c209",
"mediaTagPrefix": "/system/bundle/media/flags/",
"mediaTagVersion": 1660767531,
"nocache": true,
"parentIndex": 1,
"parentTitle": "The \"Hentai\" Prince and the Stony Cat.",
"parentYear": 2013,
"summary": "The story centers around a second-year high school boy named Yokodera Youto. Youto is always thinking about his carnal desires, but no one acknowledges him as a pervert. He learns about a cat statue that supposedly grants wishes. The boy goes to pray that he will be able to express his lustful thoughts whenever and wherever he wants. At the statue, Youto encounters Tsutsukakushi Tsukiko, a girl from his high school with her own wish - that she would not display her real intentions so readily.",
"thumb": "/library/metadata/24581/thumb/1613085732",
"title1": "Anime",
"title2": "The \"Hentai\" Prince and the Stony Cat.",
"viewGroup": "season",
"viewMode": 65593,
"Metadata": [
{
"ratingKey": "24582",
"key": "/library/metadata/24582/children",
"parentRatingKey": "24581",
"guid": "com.plexapp.agents.thetvdb://264047/1?lang=en",
"parentGuid": "com.plexapp.agents.thetvdb://264047?lang=en",
"parentStudio": "Tokyo MX",
"type": "season",
"title": "Season 1",
"parentKey": "/library/metadata/24581",
"parentTitle": "The \"Hentai\" Prince and the Stony Cat.",
"summary": "",
"index": 1,
"parentIndex": 1,
"viewCount": 3,
"skipCount": 1,
"lastViewedAt": 1665332857,
"parentYear": 2013,
"thumb": "/library/metadata/24582/thumb/1613085732",
"art": "/library/metadata/24581/art/1613085732",
"parentThumb": "/library/metadata/24581/thumb/1613085732",
"leafCount": 12,
"viewedLeafCount": 1,
"addedAt": 1613085708,
"updatedAt": 1613085732
}
]
}
}
`,
SeriesResponse: `{
"MediaContainer": {
"size": 565,
"allowSync": true,
"art": "/:/resources/show-fanart.jpg",
"identifier": "com.plexapp.plugins.library",
"librarySectionID": 1,
"librarySectionTitle": "Anime",
"librarySectionUUID": "ab4411fa-4061-4dc6-b5fb-1bd94cb7c209",
"mediaTagPrefix": "/system/bundle/media/flags/",
"mediaTagVersion": 1660767531,
"nocache": true,
"thumb": "/:/resources/show.png",
"title1": "Anime",
"title2": "All Shows",
"viewGroup": "show",
"viewMode": 65592,
"Metadata": [
{
"ratingKey": "24581",
"key": "/library/metadata/24581/children",
"skipChildren": true,
"guid": "com.plexapp.agents.thetvdb://264047?lang=en",
"studio": "Tokyo MX",
"type": "show",
"title": "Test anime title",
"titleSort": "Test anime short title",
"summary": "The story centers around a second-year high school boy named Yokodera Youto. Youto is always thinking about his carnal desires, but no one acknowledges him as a pervert. He learns about a cat statue that supposedly grants wishes. The boy goes to pray that he will be able to express his lustful thoughts whenever and wherever he wants. At the statue, Youto encounters Tsutsukakushi Tsukiko, a girl from his high school with her own wish - that she would not display her real intentions so readily.",
"index": 1,
"rating": 8.5,
"viewCount": 3,
"skipCount": 1,
"lastViewedAt": 1665332857,
"year": 2013,
"thumb": "/library/metadata/24581/thumb/1613085732",
"art": "/library/metadata/24581/art/1613085732",
"banner": "/library/metadata/24581/banner/1613085732",
"duration": 1500000,
"originallyAvailableAt": "2013-04-13",
"leafCount": 12,
"viewedLeafCount": 1,
"childCount": 1,
"addedAt": 1613085708,
"updatedAt": 1613085732,
"Genre": [
{
"tag": "Anime"
},
{
"tag": "Comedy"
}
],
"Role": [
{
"tag": "Yūki Kaji"
},
{
"tag": "Aki Toyosaki"
},
{
"tag": "Aya Suzaki"
}
]
}
]
}
}`,
	}
}