from typing import Optional
from config import MAPPING_PATH
from fileManager import load_json, save_json
import os


class PlexIdToTvdbId:
    def __init__(self) -> None:
        self.mapping_path = os.path.join(MAPPING_PATH, "plex_id_to_tvdb_id.json")
        self.mapping: dict = {}
        self.load_mapping()

    def add_new_mapping(self, plex_id: int, tvdb_id: int):
        self.mapping[plex_id] = tvdb_id
        self.save_mapping()

    def get_tvdb_id(self, plex_id: int) -> Optional[int]:
        return self.mapping.get(plex_id)

    def load_mapping(self):
        self.ensure_mapping_file_exists()
        self.mapping = load_json(self.mapping_path)

    def ensure_mapping_file_exists(self) -> None:
        if not os.path.exists(self.mapping_path):
            self.save_mapping()

    def save_mapping(self):
        save_json(self.mapping_path, self.mapping)
