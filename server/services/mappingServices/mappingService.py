from typing import List
import os
from config import ANILIST_TOKEN, MAPPING_PATH
import os
from models.anilist.anime import Anime
from models.plex.plexAnime import PlexAnime
from services.animeListServices.anilistService import AnilistService
from services.mappingServices.fribbAnimeMapping import FribbAnimeMapping
from fileManager import save_json, load_json
from models.mapping.tvdbToAnilistMapping import TvdbToAnilistMapping
import utils

logger = utils.create_logger(__name__)


class MappingService:
    def __init__(self) -> None:
        self.fribb_anime_mapping: FribbAnimeMapping = FribbAnimeMapping()
        self.anime_list_mapping_path = os.path.join(MAPPING_PATH, "anime-mapping.json")
        self.mappings: List[TvdbToAnilistMapping] = None
        self.load_mappings()
        self.check_for_new_fribbs_mappings()

    def find_new_anilist_mapping(self, anime: PlexAnime):
        anilist_service = AnilistService(ANILIST_TOKEN)

        # First see if there is an entry for another season of the show that we can use as a reference
        logger.debug(f"Checking if there's another season entry for {anime.display_name}")
        mappings_with_tvdb_id = self.find_mappings_with_tvdb_id(anime.tvdb_id)
        if len(mappings_with_tvdb_id) > 0:
            similar_anilist_id = mappings_with_tvdb_id[0].anilist_id
            new_mapping_added = self.create_anilist_season_mapping(anime, similar_anilist_id)
            if (new_mapping_added):
                return

        # Failing that check to see if the mapping exists in fribbs mapping
        logger.debug(f"Checking if fribbs mapping for {anime.display_name}")
        anilist_ids = self.get_anilist_ids_from_fribbs_mapping(anime.tvdb_id)
        chosen_anilist_id = None
        for anilist_id in anilist_ids:
            anime_data = anilist_service.get_anime(anilist_id)
            valid_anilist = anime_data is not None and anime_data.start_date is not None
            if valid_anilist and anime_data.start_date.year == anime.release_year:
                chosen_anilist_id = anilist_id
                break

        if chosen_anilist_id is not None:
            new_mapping_added = self.create_anilist_season_mapping(anime, chosen_anilist_id)
            if new_mapping_added:
                return

        # Try searching for the animes name
        logger.debug(f"Doing anilist search for for {anime.display_name}")
        search_results = anilist_service.search_for_anime(anime.title)
        search_results = [x for x in search_results if x.start_date is not None and x.start_date.year == anime.release_year]
        if len(search_results) > 0:
            result = search_results[0]
            new_mapping_added = self.create_anilist_season_mapping(anime, result.id)
            if new_mapping_added:
                return

    def create_anilist_season_mapping(self, anime: PlexAnime, anilist_id: int) -> bool:
        anilist_service = AnilistService(ANILIST_TOKEN)
        obtained_anime = anilist_service.get_anime_with_seasons(anilist_id)

        if obtained_anime is None:
            return False

        all_seasons: List[Anime] = obtained_anime.all_seasons

        # The seasons on plex are more than the seasons found on anilist
        if len(all_seasons) < int(anime.season_number):
            return False

        season = all_seasons[int(anime.season_number) - 1]
        new_mapping_added = self.add_new_mapping(anime, season)

        # Try and match the season using the name
        if not new_mapping_added:
            for index, season in enumerate(all_seasons):
                # Find the season with the matching title
                # We are going to use that as the starting season instead of the actual first season
                if season.title == anime.title:
                    season = all_seasons[index + int(anime.season_number) - 1]
                    return self.add_new_mapping(anime, season)

        return new_mapping_added

    def get_anilist_ids_from_fribbs_mapping(self, tvdb_id: int):
        return sorted(self.fribb_anime_mapping.get_anilist_ids(tvdb_id))

    def add_new_mapping(self, plex_anime: PlexAnime, anime: Anime) -> bool:
        existing_mapping = self.find_mapping_by_anilist_id(anime.id)
        if existing_mapping is not None:
            return False

        mapping = TvdbToAnilistMapping(plex_anime.tvdb_id, anime.id, plex_anime.season_number, plex_anime.title)
        self.mappings.append(mapping)
        self.save_mapping()
        return True

    def find_mapping_by_anilist_id(self, anilist_id: int):
        return next((x for x in self.mappings if x.anilist_id == anilist_id), None)

    def find_mapping_by_tvdb_id(self, tvdb_id: int, season_number: str):
        return next((x for x in self.mappings if str(x.tvdb_id) == str(tvdb_id) and str(x.season_number) == str(season_number)), None)

    def find_mappings_with_tvdb_id(self, tvdb_id: int):
        return [x for x in self.mappings if x.tvdb_id == tvdb_id]

    def check_for_new_fribbs_mappings(self) -> None:
        self.fribb_anime_mapping.ensure_anime_list_mapping_up_to_date()

    def load_mappings(self) -> None:
        self.ensure_mapping_file_exists()
        self.mappings = [self.create_mapping_from_json(x) for x in load_json(self.anime_list_mapping_path)]

    def create_mapping_from_json(self, mapping_json: dict):
        anilist_id = mapping_json.get('anilist_id')
        tvdb_id = mapping_json.get('tvdb_id')
        season_number = mapping_json.get('season_number')
        title = mapping_json.get('title')

        mapping = TvdbToAnilistMapping(tvdb_id, anilist_id, season_number, title)
        mapping.load_attributes_from_json(mapping_json)
        return mapping

    def ensure_mapping_file_exists(self) -> None:
        if not os.path.exists(self.anime_list_mapping_path):
            self.save_mapping()

    def save_mapping(self):
        data = [x.serialize() for x in self.mappings] if self.mappings is not None else []

        save_json(self.anime_list_mapping_path, data)
