package mappings_test

import (
	"slices"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/mappings"
)

func ptr(n int) *int { return &n }

func TestRangesOverlap(t *testing.T) {
	cases := []struct {
		name string
		a, b mappings.Range
		want bool
	}{
		{"disjoint", mappings.Range{First: 1, Last: ptr(12)}, mappings.Range{First: 13, Last: ptr(24)}, false},
		{"touching", mappings.Range{First: 1, Last: ptr(12)}, mappings.Range{First: 12, Last: ptr(24)}, true},
		{"contained", mappings.Range{First: 1, Last: ptr(24)}, mappings.Range{First: 5, Last: ptr(6)}, true},
		{"open ends", mappings.Range{First: 1}, mappings.Range{First: 50}, true},
		{"open after closed", mappings.Range{First: 1, Last: ptr(12)}, mappings.Range{First: 13}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.a.Overlaps(c.b); got != c.want {
				t.Errorf("a.Overlaps(b) = %v", got)
			}
			if got := c.b.Overlaps(c.a); got != c.want {
				t.Errorf("b.Overlaps(a) = %v", got)
			}
		})
	}
}

func TestResolveEndsEachRangeWhereTheNextEntryOfTheSeasonBegins(t *testing.T) {
	got := mappings.Resolve([]mappings.Row{
		{AniDBID: 3, AniListID: 303, TVDBID: 1, Season: 1, EpisodeOffset: 24},
		{AniDBID: 1, AniListID: 101, TVDBID: 1, Season: 1},
		{AniDBID: 2, AniListID: 202, TVDBID: 1, Season: 1, EpisodeOffset: 12},
	})

	want := []mappings.Mapping{
		{TVDBID: 1, Season: 1, Episodes: mappings.Range{First: 1, Last: ptr(12)}, AniListID: 101, Entry: mappings.Range{First: 1, Last: ptr(12)}, AniDBID: 1},
		{TVDBID: 1, Season: 1, Episodes: mappings.Range{First: 13, Last: ptr(24)}, AniListID: 202, Entry: mappings.Range{First: 1, Last: ptr(12)}, AniDBID: 2},
		{TVDBID: 1, Season: 1, Episodes: mappings.Range{First: 25}, AniListID: 303, Entry: mappings.Range{First: 1}, AniDBID: 3},
	}
	if !slices.EqualFunc(got, want, equal) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveKeepsTheLowestAniDBIDWhenTVDBRangesOverlap(t *testing.T) {
	got := mappings.Resolve([]mappings.Row{
		{AniDBID: 9, AniListID: 909, TVDBID: 1, Season: 1},
		{AniDBID: 4, AniListID: 404, TVDBID: 1, Season: 1},
	})

	if len(got) != 1 || got[0].AniListID != 404 {
		t.Errorf("got %+v", got)
	}
}

func TestResolveKeepsTheLowestAniDBIDWhenEntryRangesOverlap(t *testing.T) {
	got := mappings.Resolve([]mappings.Row{
		{AniDBID: 9, AniListID: 5, TVDBID: 2, Season: 1},
		{AniDBID: 4, AniListID: 5, TVDBID: 1, Season: 1},
	})

	if len(got) != 1 || got[0].TVDBID != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestResolveSkipsSpecialsAndRowsMissingIDs(t *testing.T) {
	got := mappings.Resolve([]mappings.Row{
		{AniDBID: 1, AniListID: 1, TVDBID: 1, Season: 0},
		{AniDBID: 2, AniListID: 0, TVDBID: 1, Season: 1},
		{AniDBID: 3, AniListID: 3, TVDBID: 0, Season: 1},
	})

	if len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func equal(a, b mappings.Mapping) bool {
	same := func(x, y *int) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return a.TVDBID == b.TVDBID && a.Season == b.Season && a.AniListID == b.AniListID && a.AniDBID == b.AniDBID &&
		a.Episodes.First == b.Episodes.First && same(a.Episodes.Last, b.Episodes.Last) &&
		a.Entry.First == b.Entry.First && same(a.Entry.Last, b.Entry.Last)
}
