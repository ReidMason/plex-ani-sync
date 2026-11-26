package server

import (
	"strings"
	"unicode"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
)

const titleSimilarityFloor = 0.35

// titlesDiffer reports whether the Show's title looks nothing like any title of the Entries it maps to.
// Entries without titles give no signal.
func titlesDiffer(showTitle string, entries []anilist.Media) bool {
	show := normalizeTitle(showTitle)
	compared := false
	for _, e := range entries {
		for _, t := range []string{e.TitleRomaji, e.TitleEnglish} {
			other := normalizeTitle(t)
			if other == "" || show == "" {
				continue
			}
			compared = true
			if titleSimilarity(show, other) >= titleSimilarityFloor {
				return false
			}
		}
	}
	return compared
}

func normalizeTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func titleSimilarity(a, b string) float64 {
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 1
	}
	ga, gb := bigrams(a), bigrams(b)
	shared := 0
	for g, n := range ga {
		shared += min(n, gb[g])
	}
	total := len(a) - 1 + len(b) - 1
	if total <= 0 {
		return 0
	}
	return 2 * float64(shared) / float64(total)
}

func bigrams(s string) map[string]int {
	r := []rune(s)
	out := map[string]int{}
	for i := 0; i+1 < len(r); i++ {
		out[string(r[i:i+2])]++
	}
	return out
}
