import logging
import os
import time
import urllib.request
import xml.etree.ElementTree as et
from typing import Optional

import coloredlogs

from driver import Driver
import utils

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


class Mapping:
    def __init__(self):
        self.xml_tvdb_id_to_anidb_id = self.load_tvdb_id_to_anidb_id_xml()
        self.tvdb_id_to_anilist_id = self.load_tvdb_id_to_anilist_id()
        self.anime_offline_database = self.load_anime_offline_database()

        self.driver = None
        # Clear mapping errors
        self.save_mapping_errors({})

    def get_driver(self):
        if self.driver is None:
            self.driver = Driver()

        return self.driver

    # ---- Handling tvdb_id to anidb_id xml file
    def load_tvdb_id_to_anidb_id_xml(self):
        self.update_mapping_xml()
        return et.parse('data/tvdbid_to_anidbid.xml').getroot()

    def download_tvdb_anidb_mapping(self) -> None:
        logger.info("Downloading new XML mapping file")
        if os.path.exists('data/tvdbid_to_anidbid.xml'):
            os.remove('data/tvdbid_to_anidbid.xml')

        urllib.request.urlretrieve(
            'https://raw.githubusercontent.com/ScudLee/anime-lists/master/anime-list-full.xml',
            'data/tvdbid_to_anidbid.xml')

    def update_mapping_xml(self) -> None:
        if not os.path.exists('data/tvdbid_to_anidbid.xml'):
            self.download_tvdb_anidb_mapping()

        mapping_file_age = time.time() - os.path.getctime('data/tvdbid_to_anidbid.xml')
        # Replace if the old file is 7 days old
        if mapping_file_age >= 603_800:
            self.download_tvdb_anidb_mapping()

    # ---- Handling anime-offline-database file
    def load_anime_offline_database(self):
        self.update_mapping_xml()
        return utils.load_json('data/anime-offline-database.json').get('data')

    def update_anime_offline_database(self):
        if not os.path.exists('data/anime-offline-database.json'):
            self.download_anime_offline_database()

        file_age = time.time() - os.path.getctime('data/anime-offline-database.json')
        # Replace if the old file is 7 days old
        if file_age >= 603_800:
            self.download_anime_offline_database()

    def download_anime_offline_database(self):
        logger.info("Downloading new anime-offline-database file")
        if os.path.exists('data/anime-offline-database.json'):
            os.remove('data/anime-offline-database.json')

        url = 'https://raw.githubusercontent.com/manami-project/anime-offline-database/master/anime-offline-database.json'
        urllib.request.urlretrieve(url, 'data/anime-offline-database.json')

    # ---- Handling tvdb_id to anilist_id file
    def load_tvdb_id_to_anilist_id(self) -> dict:
        logger.debug("Loading tvdb_id to anilist_id")
        return utils.load_json('data/tvdbid_to_anilistid.json')

    def save_tvdb_id_to_anilist_id(self):
        utils.save_json(self.tvdb_id_to_anilist_id, 'data/tvdbid_to_anilistid.json')

    # ---- End of file handlers
    def get_anilist_id(self, tvdb_id: str, title: str, season: str):
        # Check if mapping already exists
        anilist_id = self.tvdb_id_to_anilist_id.get(tvdb_id, {}).get(season)
        if anilist_id is not None:
            return anilist_id

        # Create a new mapping
        return self.create_tvdb_id_to_anilist_id_mapping(tvdb_id, title, season)

    def create_tvdb_id_to_anilist_id_mapping(self, tvdb_id: str, title: str, season: str):
        logger.warning(f"Creating new anime mapping for {title} Season {season}")
        anilist_id = None
        if (anidb_id := self.get_anidb_id_from_tvdb_id(tvdb_id, season)) is not None:
            anilist_id = self.get_anilist_id_from_aod(anidb_id)

        self.tvdb_id_to_anilist_id[tvdb_id] = {**self.tvdb_id_to_anilist_id.get(tvdb_id, {}), **{season: anilist_id}}
        self.save_tvdb_id_to_anilist_id()
        # self.remove_solved_mapping_errors()
        return anilist_id

    def get_anidb_id_from_tvdb_id(self, tvdb_id: str, season: str) -> Optional[str]:
        for anime in list(self.xml_tvdb_id_to_anidb_id):
            if anime.get('tvdbid') == tvdb_id and anime.get('defaulttvdbseason') == season:
                return anime.get('anidbid')
        return None

    def get_anilist_id_from_aod(self, anidb_id: str):
        for anime in self.anime_offline_database:
            if f'https://anidb.net/anime/{anidb_id}' in anime.get('sources'):
                for source in anime.get('sources'):
                    if source.startswith('https://anilist.co/anime/'):
                        return source.rsplit('/')[-1]
        return None

    def add_to_mapping_errors(self, anime):
        logger.debug(f"Adding {anime.title} Season {anime.season_number} to mapping errors")
        mapping_errors = utils.load_json('data/mapping_errors.json')
        if anime.tvdb_id not in mapping_errors:
            mapping_errors[anime.tvdb_id] = {'title'  : anime.title,
                                             'seasons': []}

        if anime.season_number not in mapping_errors[anime.tvdb_id]['seasons']:
            mapping_errors[anime.tvdb_id]['seasons'].append(anime.season_number)

        self.save_mapping_errors(mapping_errors)

    def save_mapping_errors(self, data):
        utils.save_json(data, 'data/mapping_errors.json')
