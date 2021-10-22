from typing import List
import os
from config import ANILIST_TOKEN, MAPPING_PATH
import os
from models.anilist.anime import Anime
from models.mapping.fribbsAnilistMapping import FribbsMapping
from models.plex.plexAnime import PlexAnime
from services.animeListServices.anilistService import AnilistService
from services.mappingServices.fribbAnimeMapping import FribbAnimeMapping
from fileManager import save_json, load_json
from models.mapping.tvdbToAnilistMapping import TvdbToAnilistMapping


class MappingService:
    def __init__(self) -> None:
        self.fribb_anime_mapping: FribbAnimeMapping = FribbAnimeMapping()
        self.anime_list_mapping_path = os.path.join(MAPPING_PATH, "anime-mapping.json")
        self.mappings: List[TvdbToAnilistMapping] = None
        self.load_mappings()
        self.check_for_new_fribbs_mappings()

    def find_new_anilist_mapping(self, tvdb_id: int):
        anilistService = AnilistService(ANILIST_TOKEN)

        # First see if there is an entry for another season of the show that we can use as a reference
        mappings_with_tvdb_id = self.find_mappings_with_tvdb_id(tvdb_id)
        if len(mappings_with_tvdb_id) > 0:
            similar_anilist_id = mappings_with_tvdb_id[0].anilist_id

            obtained_anime = anilistService.get_anime_with_seasons(similar_anilist_id)
            all_seasons: List[Anime] = obtained_anime.all_seasons
            for season in all_seasons:
                self.add_new_mapping(tvdb_id, season)
            if len(all_seasons) > 0:
                return

        # Failing that check to see if the mapping exists in fribbs mapping
        anilist_ids = self.get_anilist_ids_from_fribbs_mapping(tvdb_id)

        # No anilist_ids were found
        if len(anilist_ids) == 0:

            return

        anilist_id = anilist_ids[0]
        obtained_anime = anilistService.get_anime_with_seasons(anilist_id)

        if obtained_anime is None:
            return

        all_seasons: List[Anime] = obtained_anime.all_seasons
        for season in all_seasons:
            self.add_new_mapping(tvdb_id, season)

    def get_anilist_ids_from_fribbs_mapping(self, tvdb_id: int):
        return self.fribb_anime_mapping.get_anilist_ids(tvdb_id)

    def add_new_mapping(self, tvdb_id: int, anime: Anime):
        existing_mapping = self.find_mapping_by_anilist_id(anime.id)
        if existing_mapping is None:
            mapping = TvdbToAnilistMapping(tvdb_id, anime.id, anime.season_number)
            mapping.add_anime_attributes(anime)
            self.mappings.append(mapping)
            self.save_mapping()

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

        mapping = TvdbToAnilistMapping(tvdb_id, anilist_id, season_number)
        mapping.load_attributes_from_json(mapping_json)
        return mapping

    def ensure_mapping_file_exists(self) -> None:
        if not os.path.exists(self.anime_list_mapping_path):
            self.save_mapping()

    def save_mapping(self):
        data = [x.serialize() for x in self.mappings] if self.mappings is not None else []

        save_json(self.anime_list_mapping_path, data)
