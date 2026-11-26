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
	return startsBeforeEnd(o.First, r.Last) && startsBeforeEnd(r.First, o.Last)
}

func startsBeforeEnd(first int, last *int) bool {
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

// Resolve turns dataset rows into Mappings; the lowest AniDB ID wins an overlap and a losing row is dropped whole,
// as is any row overlapping a Custom Mapping.
func Resolve(rows []Row, lists AnimeLists, custom []Mapping) []Mapping {
	usable := slices.DeleteFunc(slices.Clone(rows), func(r Row) bool {
		return r.AniListID <= 0 || r.TVDBID <= 0
	})
	slices.SortStableFunc(usable, func(a, b Row) int { return a.AniDBID - b.AniDBID })

	offsets := map[seasonKey][]int{}
	for _, r := range usable {
		if len(lists[r.AniDBID]) > 0 || !fitsOneSeason(r) {
			continue
		}
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
	add := func(candidates []Mapping) {
		taken := slices.ContainsFunc(candidates, func(m Mapping) bool {
			return slices.ContainsFunc(bySeason[seasonKey{m.TVDBID, m.Season}], m.Episodes.Overlaps) ||
				slices.ContainsFunc(byEntry[m.AniListID], m.Entry.Overlaps)
		})
		if taken {
			return
		}
		for _, m := range candidates {
			k := seasonKey{m.TVDBID, m.Season}
			bySeason[k] = append(bySeason[k], m.Episodes)
			byEntry[m.AniListID] = append(byEntry[m.AniListID], m.Entry)
			out = append(out, m)
		}
	}
	for _, r := range usable {
		if spans := lists[r.AniDBID]; len(spans) > 0 {
			var candidates []Mapping
			for _, sp := range spans {
				candidates = append(candidates, Mapping{
					TVDBID: r.TVDBID, Season: sp.Season,
					Episodes:  Range{First: sp.First + sp.Offset, Last: intPtr(sp.Last + sp.Offset)},
					AniListID: r.AniListID, Entry: Range{First: sp.First, Last: intPtr(sp.Last)}, AniDBID: r.AniDBID,
				})
			}
			if slices.ContainsFunc(candidates, func(m Mapping) bool { return overlapsCustom(custom, m) }) {
				continue
			}
			add(candidates)
			continue
		}
		if !fitsOneSeason(r) {
			continue
		}
		episodes, entry := rangesFromOffsets(r, offsets[seasonKey{r.TVDBID, r.Season}])
		m := Mapping{TVDBID: r.TVDBID, Season: r.Season, Episodes: episodes, AniListID: r.AniListID, Entry: entry, AniDBID: r.AniDBID}
		if !overlapsCustom(custom, m) {
			add([]Mapping{m})
		}
	}
	return out
}

func fitsOneSeason(r Row) bool { return r.Season > 0 && r.EpisodeOffset >= 0 }

func intPtr(n int) *int { return &n }

func rangesFromOffsets(r Row, seasonOffsets []int) (episodes, entry Range) {
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

func overlapsCustom(custom []Mapping, m Mapping) bool {
	return slices.ContainsFunc(custom, func(c Mapping) bool {
		return (c.TVDBID == m.TVDBID && c.Season == m.Season && c.Episodes.Overlaps(m.Episodes)) ||
			(c.AniListID == m.AniListID && c.Entry.Overlaps(m.Entry))
	})
}
