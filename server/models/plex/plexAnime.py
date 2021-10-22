from typing import Optional
from plexapi.video import Season
import utils

from services.mappingServices.plexIdToTvdbId import PlexIdToTvdbId

logger = utils.create_logger(__name__)


class PlexAnime:
    def __init__(self, plex_season: Season, tvdb_id: Optional[int] = None, release_year: Optional[int] = None):
        self.title: str = plex_season.parentTitle
        logger.info(f"Processing: {self.title}")
        self.tvdb_id: Optional[int] = tvdb_id
        self.release_year: Optional[int] = release_year
        self.season_number: str = str(plex_season.seasonNumber)
        self.display_name: str = f'{plex_season.parentTitle} - Season {self.season_number}'
        self.release_year = plex_season.year

        self.plex_season: Season = plex_season
        self._episodes: int = None

    @property
    def episodes(self) -> int:
        if self._episodes is None:
            self._episodes = self.plex_season.episodes()
        return self._episodes

    @property
    def episodes_watched(self) -> int:
        return len([x for x in self.episodes if x is not None and x.isWatched])

    def __repr__(self) -> str:
        return self.display_name
