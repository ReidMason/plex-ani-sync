import os
from typing import List
import urllib.request
import time
from config import MAPPING_PATH
from fileManager import ensure_required_directories_exist, load_json
from models.mapping.fribbsAnilistMapping import FribbsMapping
import utils

logger = utils.create_logger(__name__)


class FribbAnimeMapping:
    def __init__(self, mapping_update_threshold: int = 604800) -> None:
        self.anime_list_mapping_path = os.path.join(MAPPING_PATH, "fribb-anime-list-full.json")
        # How old the fribb mapping file should be before a new one is dowloaded
        self.mapping_update_threshold = mapping_update_threshold

        self.mappings: FribbsMapping = None
        self.load_mapping()

    def extract_mappings(self, mapping_data: List[dict]) -> None:
        self.ensure_anime_list_mapping_up_to_date()
        self.mappings: List[FribbsMapping] = []
        for mapping in mapping_data:
            tvdb_id = mapping.get('thetvdb_id')
            anilist_id = mapping.get('anilist_id')
            if tvdb_id is not None and anilist_id is not None:
                self.mappings.append(FribbsMapping(tvdb_id, anilist_id))

    def get_anilist_ids(self, tvdb_id: int):
        anilist_ids: List[int] = []
        for mapping in self.mappings:
            if str(mapping.tvdb_id) == str(tvdb_id):
                anilist_ids.append(mapping.anilist_id)

        return anilist_ids

    def load_mapping(self):
        """ Updates and loads the downloaded mapping file """
        self.ensure_anime_list_mapping_up_to_date()
        self.extract_mappings(load_json(self.anime_list_mapping_path))

    def ensure_anime_list_mapping_up_to_date(self) -> bool:
        """ Ensures the currently downloaded fribb mapping doesn't exceed the curent mapping update threshold """
        if self.fribb_anime_list_needs_updating():
            self.download_fribb_anime_list_mapping()
            return False

        return True

    def download_fribb_anime_list_mapping(self) -> None:
        """ Downloadds a new version of the fribb anime list mapping file """
        ensure_required_directories_exist()
        logger.info("Downloading new fribbs mapping")

        # Delete the file if it already exists
        if os.path.exists(self.anime_list_mapping_path):
            os.remove(self.anime_list_mapping_path)

        url = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json"
        urllib.request.urlretrieve(url, self.anime_list_mapping_path)

    def fribb_anime_list_needs_updating(self) -> bool:
        """ Checks to see if the fribb anime list mapping file exceeds the current mapping update threshold """
        # If the file doesn't exist we need to download it
        if not os.path.exists(self.anime_list_mapping_path):
            return True

        # Get how long it's been since it was downloaded
        created_time = os.path.getctime(self.anime_list_mapping_path)
        time_since_download = time.time() - created_time

        return time_since_download > self.mapping_update_threshold
