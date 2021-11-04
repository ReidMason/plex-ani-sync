import abc
from typing import Protocol

from models.animeList.animeList import AnimeList


class IAnimeListService(Protocol):
    @abc.abstractmethod
    def get_anime_list(self) -> AnimeList:
        raise NotImplementedError("Please Implement this method")

    @abc.abstractmethod
    def update_anime(self, anime_id: int, watched_episodes: int, status: int, title: str = None) -> bool:
        raise NotImplementedError("Please Implement this method")
