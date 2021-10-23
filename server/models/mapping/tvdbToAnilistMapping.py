from typing import Mapping
from datetime import datetime
from models.anilist.anime import Anime


class TvdbToAnilistMapping:
    def __init__(self, tvdb_id: int, anilist_id: int, season_number: int, title: str = None) -> None:
        self.season_number: str = season_number
        self.tvdb_id: int = tvdb_id
        self.anilist_id: int = anilist_id
        self.title: str = title

    def load_attributes_from_json(self, data: dict):
        fields = [x for x in self.__dict__]
        for field in fields:
            data_value = data.get(field)
            if data_value is not None:
                setattr(self, field, data_value)

    def serialize(self):
        return self.__dict__
