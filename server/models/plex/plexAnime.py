from typing import List, Optional
from plexapi.video import Episode, Season
import utils
from datetime import datetime
from services.mappingServices.plexIdToTvdbId import PlexIdToTvdbId

logger = utils.create_logger("PlexAnime")


class PlexAnime:
    def __init__(self, plex_season: Season, tvdb_id: Optional[int] = None, release_year: Optional[int] = None):
        self.title: str = plex_season.parentTitle
        self.season_number: str = str(plex_season.seasonNumber)
        self.tvdb_id: Optional[int] = tvdb_id
        self.display_name: str = f'{plex_season.parentTitle} - Season {self.season_number} ({self.tvdb_id})'
        self.release_year: Optional[int] = release_year

        self.plex_season: Season = plex_season
        self._episodes: Optional[List[Episode]] = None
        self._episodes_watched: int = None
        self._last_viewed_at: Optional[datetime] = None

    @property
    def episodes(self) -> List[Episode]:
        if self._episodes is None:
            self._episodes = self.plex_season.episodes()
        return self._episodes

    def set_cached_episodes(self, cached_episodes: List[Episode]):
        self._episodes = cached_episodes
        # We need to reset the other cached values that rely on this as well
        self._last_viewed_at = None
        self._episodes_watched = None

    @property
    def episodes_watched(self) -> int:
        if self._episodes_watched is None:
            self._episodes_watched = len([x for x in self.episodes if x is not None and x.isWatched])
        return self._episodes_watched

    @property
    def last_viewed_at(self) -> Optional[datetime]:
        if self._last_viewed_at is None:
            for episode in self.episodes:
                if self._last_viewed_at is None or (episode.lastViewedAt is not None and self._last_viewed_at < episode.lastViewedAt):
                    self._last_viewed_at = episode.lastViewedAt

        return self._last_viewed_at

    def __repr__(self) -> str:
        return self.display_name
