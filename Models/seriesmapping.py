from typing import List, Optional

from Models.seasonmapping import SeasonMapping


class SeriesMapping:
    def __init__(self, mapping_data: dict = None):
        if mapping_data is None:
            mapping_data = {}

        self.tvdbid: str = ''
        self.name: str = ''
        self.ignore: bool = False

        for prop in self.__dict__:
            setattr(self, prop, mapping_data.get(prop, getattr(self, prop)))

        self.seasons: List[SeasonMapping] = [SeasonMapping(x) for x in mapping_data.get('seasons', [])]

    @property
    def contains_wildcard_season(self):
        return any([True for x in self.seasons if "*" in x.all_seasons])

    def find_season_mapping(self, season_number: str) -> Optional[SeasonMapping]:
        return next((x for x in self.seasons if x.plex_season_number == season_number), None)

    def add_season_mapping(self, season_mapping: SeasonMapping):
        self.seasons.append(season_mapping)
