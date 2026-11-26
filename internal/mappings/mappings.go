package mappings

import (
	"slices"
)

type Range struct {
	First int
	// Last is nil when the range is open-ended.
	Last *int
}

func (r Range) Overlaps(o Range) bool {
	return before(o.First, r.Last) && before(r.First, o.Last)
}

func before(first int, last *int) bool {
	return last == nil || first <= *last
}

type Row struct {
	AniDBID       int
	AniListID     int
	TVDBID        int
	Season        int
	EpisodeOffset int
}

type Mapping struct {
	TVDBID    int
	Season    int
	Episodes  Range
	AniListID int
	Entry     Range
	AniDBID   int
}

type seasonKey struct{ tvdbID, season int }

// Resolve turns dataset rows into Mappings, skipping Specials and incomplete rows. A row's range ends where the
// next row of the same Season begins; on overlap the row with the lowest AniDB ID wins.
func Resolve(rows []Row) []Mapping {
	return ResolveAround(rows, nil)
}

// ResolveAround is Resolve, except any row whose TVDB or Entry range overlaps a Custom Mapping is dropped whole.
func ResolveAround(rows []Row, custom []Mapping) []Mapping {
	usable := slices.DeleteFunc(slices.Clone(rows), func(r Row) bool {
		return r.AniListID <= 0 || r.TVDBID <= 0 || r.Season <= 0 || r.EpisodeOffset < 0
	})
	slices.SortStableFunc(usable, func(a, b Row) int { return a.AniDBID - b.AniDBID })

	offsets := map[seasonKey][]int{}
	for _, r := range usable {
		k := seasonKey{r.TVDBID, r.Season}
		offsets[k] = append(offsets[k], r.EpisodeOffset)
	}
	for k := range offsets {
		slices.Sort(offsets[k])
		offsets[k] = slices.Compact(offsets[k])
	}

	var out []Mapping
	bySeason := map[seasonKey][]Range{}
	byEntry := map[int][]Range{}
	for _, r := range usable {
		k := seasonKey{r.TVDBID, r.Season}
		episodes, entry := rangesOf(r, offsets[k])
		if overlapsCustom(custom, r, episodes, entry) {
			continue
		}
		if slices.ContainsFunc(bySeason[k], episodes.Overlaps) || slices.ContainsFunc(byEntry[r.AniListID], entry.Overlaps) {
			continue
		}
		bySeason[k] = append(bySeason[k], episodes)
		byEntry[r.AniListID] = append(byEntry[r.AniListID], entry)
		out = append(out, Mapping{TVDBID: r.TVDBID, Season: r.Season, Episodes: episodes, AniListID: r.AniListID, Entry: entry, AniDBID: r.AniDBID})
	}
	return out
}

func rangesOf(r Row, seasonOffsets []int) (episodes, entry Range) {
	episodes = Range{First: r.EpisodeOffset + 1}
	entry = Range{First: 1}
	i, _ := slices.BinarySearch(seasonOffsets, r.EpisodeOffset)
	if i+1 < len(seasonOffsets) {
		next := seasonOffsets[i+1]
		episodes.Last = &next
		length := next - r.EpisodeOffset
		entry.Last = &length
	}
	return episodes, entry
}

func overlapsCustom(custom []Mapping, r Row, episodes, entry Range) bool {
	return slices.ContainsFunc(custom, func(c Mapping) bool {
		return (c.TVDBID == r.TVDBID && c.Season == r.Season && c.Episodes.Overlaps(episodes)) ||
			(c.AniListID == r.AniListID && c.Entry.Overlaps(entry))
	})
}
