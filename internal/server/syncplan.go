package server

import (
	"strconv"
	"time"
)

const (
	listStatusWatching  = "CURRENT"
	listStatusCompleted = "COMPLETED"
	listStatusRepeating = "REPEATING"
	listStatusPaused    = "PAUSED"
	listStatusDropped   = "DROPPED"

	pausedAfter  = 14 * 24 * time.Hour
	droppedAfter = 30 * 24 * time.Hour
)

type listState struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

type entryFacts struct {
	anilistID   int64
	title       string
	seasons     []int
	watched     int
	lastWatched *time.Time
	total       *int
	now         time.Time
}

type planChange struct {
	AniListID int64      `json:"anilistId"`
	Title     string     `json:"title"`
	Seasons   []int      `json:"seasons"`
	Current   *listState `json:"current"`
	Proposed  listState  `json:"proposed"`
	Reason    string     `json:"reason"`
}

func planEntry(f entryFacts, current *listState) (planChange, bool) {
	if f.watched == 0 {
		return planChange{}, false
	}
	if current != nil && (current.Status == listStatusCompleted || current.Status == listStatusRepeating) {
		return planChange{}, false
	}
	progress := f.watched
	if current != nil {
		progress = max(progress, current.Progress)
	}
	proposed := listState{Status: proposedStatus(f, progress, current), Progress: progress}
	if current != nil && proposed == *current {
		return planChange{}, false
	}
	return planChange{AniListID: f.anilistID, Title: f.title, Seasons: f.seasons, Current: current, Proposed: proposed, Reason: changeReason(f, current, proposed)}, true
}

func changeReason(f entryFacts, current *listState, proposed listState) string {
	switch {
	case current == nil:
		return "Not on your AniList list; " + episodes(f.watched) + " watched in Plex"
	case proposed.Progress > current.Progress:
		return episodes(f.watched) + " watched in Plex, AniList has " + episodes(current.Progress)
	default:
		return "Progress " + strconv.Itoa(proposed.Progress) + " suggests " + proposed.Status + " instead of " + current.Status
	}
}

func proposedStatus(f entryFacts, progress int, current *listState) string {
	if f.total != nil && progress == *f.total {
		return listStatusCompleted
	}
	if f.lastWatched != nil {
		idle := f.now.Sub(*f.lastWatched)
		switch {
		case idle > droppedAfter:
			return listStatusDropped
		case idle > pausedAfter:
			if current != nil && current.Status == listStatusDropped {
				return listStatusDropped
			}
			return listStatusPaused
		}
	}
	return listStatusWatching
}

func episodes(n int) string {
	if n == 1 {
		return "1 episode"
	}
	return strconv.Itoa(n) + " episodes"
}
