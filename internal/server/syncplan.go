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
	watched     int
	lastWatched *time.Time
	total       *int
	now         time.Time
}

type planChange struct {
	AniListID int64      `json:"anilistId"`
	Title     string     `json:"title"`
	Current   *listState `json:"current"`
	Proposed  listState  `json:"proposed"`
	Reason    string     `json:"reason"`
}

// planEntry decides what a sync would change for one Entry; current is nil when the Entry is not on the list.
func planEntry(f entryFacts, current *listState) (planChange, bool) {
	if f.watched == 0 {
		return planChange{}, false
	}
	if current != nil && (current.Status == listStatusCompleted || current.Status == listStatusRepeating) {
		return planChange{}, false
	}
	proposed := listState{Status: proposedStatus(f, f.watched), Progress: f.watched}
	reason := "Not on your AniList list; " + episodes(f.watched) + " watched in Plex"
	if current != nil {
		if f.watched <= current.Progress {
			return planChange{}, false
		}
		reason = episodes(f.watched) + " watched in Plex, AniList has " + episodes(current.Progress)
	}
	return planChange{AniListID: f.anilistID, Title: f.title, Current: current, Proposed: proposed, Reason: reason}, true
}

// proposedStatus is where status rules grow; for now an Entry keeps its status and additions are Watching.
func proposedStatus(f entryFacts, progress int) string {
	if f.total != nil && progress == *f.total {
		return listStatusCompleted
	}
	if f.lastWatched != nil {
		idle := f.now.Sub(*f.lastWatched)
		switch {
		case idle > droppedAfter:
			return listStatusDropped
		case idle > pausedAfter:
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
