from typing import Mapping
from datetime import datetime
from models.anilist.anime import Anime


class TvdbToAnilistMapping:
    def __init__(self, tvdb_id: int, anilist_id: int, season_number: int) -> None:
        self.season_number: str = season_number
        self.tvdb_id: int = tvdb_id
        self.anilist_id: int = anilist_id

        self.english_title: str = None
        self.romaji_title: str = None

    @property
    def title(self):
        return self.english_title or self.romaji_title

    def load_attributes_from_json(self, data: dict):
        fields = [x for x in self.__dict__]
        for field in fields:
            data_value = data.get(field)
            if data_value is not None:
                setattr(self, field, data_value)

    def add_anime_attributes(self, anime: Anime):
        self.english_title = anime.english_title
        self.romaji_title = anime.romaji_title

    @property
    def release_date(self):
        return datetime(self.release_year, self.release_month, self.release_day)

    def serialize(self):
        return self.__dict__
