package mappings_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/mappings"
)

const animeListFixture = `<?xml version="1.0" encoding="utf-8"?>
<anime-list>
	<anime anidbid="5" tvdbid="72025" defaulttvdbseason="3">
		<mapping-list>
			<mapping anidbseason="0" tvdbseason="0">;1-4;</mapping>
			<mapping anidbseason="1" tvdbseason="1" start="1" end="30"/>
			<mapping anidbseason="1" tvdbseason="2" start="31" end="60" offset="-30"/>
			<mapping anidbseason="1" tmdbseason="3" start="61" end="75" offset="-60"/>
		</mapping-list>
	</anime>
	<anime anidbid="7" tvdbid="movie"><name>Film</name></anime>
</anime-list>`

func TestFetchAnimeListsKeepsOnlyRangeMappingsIntoTVDBSeasons(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(animeListFixture)) }))
	defer srv.Close()

	got, err := mappings.FetchAnimeLists(context.Background(), srv.Client(), srv.URL)

	want := mappings.AnimeLists{5: {{Season: 1, First: 1, Last: 30}, {Season: 2, First: 31, Last: 60, Offset: -30}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}

func TestFetchAnimeListsParsesDocumentRecordedFromTheRealDataset(t *testing.T) {
	srv := serveFile(t, "testdata/anime-list-sample.xml")

	got, err := mappings.FetchAnimeLists(context.Background(), srv.Client(), srv.URL)

	want := mappings.AnimeLists{19: {{Season: 1, First: 1, Last: 12}, {Season: 2, First: 13, Last: 24, Offset: -12}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}
