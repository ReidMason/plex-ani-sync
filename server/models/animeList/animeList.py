from typing import List

from models.animeList.animeListAnime import AnimeListAnime


class AnimeList:
    def __init__(self, list_name: str, anime_list: List[AnimeListAnime] = None):
        self.list_name: str = str(list_name)
        self.anime_list: List[AnimeListAnime] = anime_list if anime_list is not None else []

    def get_anime(self, anilist_id: int):
        return next((x for x in self.anime_list if str(x.anime_id) == str(anilist_id)), None)
