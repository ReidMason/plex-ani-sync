from typing import List, Optional

import pytest
from config import Config

from services.animeListServices.anilistService import AnilistService, AnilistUser
from models.animeList.animeList import AnimeList
from models.animeList.animeListAnime import AnimeListAnime

mock_lists = [
    {"name": "Planning"},
    {"name": "Completed"},
    {"name": "Dropped"},
    {"name": "Paused"},
    {"name": "Default"},
]

mock_anime_update_data = AnimeListAnime("1", "Cowbow Bebop", "Cowboy Bebop", 0, 10, 2)

config = Config()
token = config.ANILIST_TOKEN
invalid_token = "invalidToken123"


class TestGetAnimeList:
    @pytest.fixture(scope="class")
    def anime_list(self):
        anilist_service = AnilistService(token)
        return anilist_service.get_anime_list()

    @pytest.fixture(scope="class")
    def anime_list_invalid_token(self):
        anilist_service = AnilistService(invalid_token)
        return anilist_service.get_anime_list()

    def test_anime_list_type_returned_is_anime_list(self, anime_list: AnimeList):
        assert isinstance(anime_list, AnimeList)

    def test_anime_list_name_is_set(self, anime_list: AnimeList):
        assert anime_list.list_name is not None

    def test_anime_list_in_anime_list_object_is_a_list(self, anime_list: AnimeList):
        assert isinstance(anime_list.anime_list, list)

    def test_anime_list_is_populated_with_anime(self, anime_list: AnimeList):
        assert len(anime_list.anime_list) > 0

    def test_anime_in_anime_list_is_anime_list_anime(self, anime_list: AnimeList):
        assert isinstance(anime_list.anime_list[0], AnimeListAnime)

    def test_anime_in_anime_list_has_required_fields(self, anime_list: AnimeList):
        assert anime_list.anime_list[0].anime_id is not None
        assert anime_list.anime_list[0].title is not None
        assert anime_list.anime_list[0].romaji_title is not None
        assert anime_list.anime_list[0].watched_episodes is not None
        assert anime_list.anime_list[0].watch_status is not None
        assert anime_list.anime_list[0].total_episodes is not None

    def test_invali_token(self, anime_list_invalid_token: AnimeList):
        assert len(anime_list_invalid_token.anime_list) == 0


class TestFilterInvalidLists:
    @pytest.fixture(scope="class")
    def filtered_list(self):
        anilist_service = AnilistService(token)
        return anilist_service.filter_invalid_lists(mock_lists)

    def test_filtered_list_contains_correct_values(self, filtered_list: List[dict]):
        assert filtered_list == mock_lists[:-1]


class TestGetUser:
    @pytest.fixture(scope="class")
    def user(self):
        anilist_service = AnilistService(token)
        return anilist_service.get_user()

    @pytest.fixture(scope="class")
    def user_invalid_token(self):
        anilist_service = AnilistService(invalid_token)
        return anilist_service.get_user()

    def test_user_is_anilist_user(self, user: AnilistUser):
        assert isinstance(user, AnilistUser) is True

    def test_invalid_token_user_id_is_correct(self, user_invalid_token: AnilistUser):
        assert user_invalid_token.user_id == ""

    def test_invalid_token_user_name_is_correct(self, user_invalid_token: AnilistUser):
        assert user_invalid_token.name == ""


class TestUpdateAnime:
    @pytest.fixture(scope="class")
    def class_wrapper(self):
        yield

        # Change anime status to something that doesn't match mock data
        anilist_service = AnilistService(token)
        anilist_service.update_anime(
            mock_anime_update_data.anime_id,
            mock_anime_update_data.watched_episodes - 1,
            5,
        )

    @pytest.fixture(scope="class")
    def anilist_service(self):
        return AnilistService(token)

    @pytest.fixture(scope="class")
    def updated(self, anilist_service: AnilistService):
        return anilist_service.update_anime(
            mock_anime_update_data.anime_id,
            mock_anime_update_data.watched_episodes,
            mock_anime_update_data.watch_status,
        )

    @pytest.fixture(scope="class")
    def anilist_service_invalid_token(self):
        return AnilistService(invalid_token)

    @pytest.fixture(scope="class")
    def updated_invalid_token(self, anilist_service_invalid_token: AnilistService):
        return anilist_service_invalid_token.update_anime(
            mock_anime_update_data.anime_id,
            mock_anime_update_data.watched_episodes,
            mock_anime_update_data.watch_status,
        )

    @pytest.fixture(scope="class")
    def updated_anime(
        self, anilist_service: AnilistService
    ) -> Optional[AnimeListAnime]:
        anime_list = anilist_service.get_anime_list()
        return next(
            (
                x
                for x in anime_list.anime_list
                if x.anime_id == mock_anime_update_data.anime_id
            ),
            None,
        )

    def test_update_success_is_returned(self, updated: bool):
        assert updated is True

    def test_updated_anime_is_on_anime_list(self, updated_anime: AnimeListAnime):
        assert updated_anime is not None

    def test_updated_anime_has_updated_watch_status(
        self, updated_anime: AnimeListAnime
    ):
        assert updated_anime.watch_status == mock_anime_update_data.watch_status

    def test_updated_anime_has_updated_watched_episodes(
        self, updated_anime: AnimeListAnime
    ):
        assert updated_anime.watched_episodes == mock_anime_update_data.watched_episodes

    def test_updating_anime_with_invalid_token_failure_is_returned(
        self, updated_invalid_token: bool
    ):
        assert updated_invalid_token is False
