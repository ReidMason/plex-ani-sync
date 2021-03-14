from datetime import datetime
from typing import List, Optional


class SeasonMapping:
    def __init__(self, season_data: dict = None):
        if season_data is None:
            season_data = {}

        self.plex_season_number: str = ''
        self.plex_additional_seasons: List[str] = []
        self.anilistid: str = ''
        self.ignore: bool = False
        self.episode_start: Optional[int] = None
        self.season_length: Optional[int] = None
        self.unix_last_updated: Optional[int] = None

        for prop in self.__dict__:
            setattr(self, prop, season_data.get(prop, getattr(self, prop)))

    @property
    def last_updated_is_none(self) -> bool:
        return self.unix_last_updated is None

    @property
    def last_updated(self) -> datetime:
        if self.unix_last_updated is not None:
            return datetime.utcfromtimestamp(self.unix_last_updated)

    @property
    def days_since_last_update(self) -> int:
        if self.last_updated is None:
            return 0
        return abs((datetime.now() - self.last_updated).days)

    @property
    def has_custom_range(self) -> bool:
        return None not in (self.episode_start, self.season_length)

    @property
    def episode_end(self) -> int:
        return int(self.episode_start) + int(self.season_length) - 1

    @property
    def all_seasons(self) -> List[str]:
        seasons = [self.plex_season_number]
        seasons.extend(self.plex_additional_seasons)
        return [str(x) for x in seasons]
