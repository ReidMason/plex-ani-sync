import os
import time
import urllib.request
from typing import List, Optional

import xml.etree.ElementTree as et
import utils
from Models.seasonmapping import SeasonMapping
from Models.seriesmapping import SeriesMapping
from plex import PlexAnime

tvdbid_to_anilist_filepath = 'data/tvdbid_to_anilistid.json'
logger = utils.create_logger(__name__)


class MappingSuggester:
    def __init__(self):
        self.tvdbid_to_anidbid: dict = self.get_tvdbid_to_anidbid()
        self.anidbid_to_anilistid: dict = self.get_anidbid_to_anilistid()

    def get_mapping_suggestion(self, tvdbid: str, season_number: str) -> Optional[str]:
        anidb_mapping = self.tvdbid_to_anidbid.get(tvdbid)
        if anidb_mapping is None:
            return None

        anidbid = anidb_mapping.get(season_number)
        anilistid = self.anidbid_to_anilistid.get(anidbid)

        return anilistid

    def load_anime_offline_database(self) -> dict:
        """ Get an up to date version of the mapping anime offline database mapping file.

        :return: The up to date anime offline database mapping file.
        """
        logger.info("Loading anime offline database")
        download_url = 'https://raw.githubusercontent.com/manami-project/anime-offline-database/master/anime-offline-database.json'
        filepath = 'data/anime-offline-database.json'
        self.update_mapping_file(filepath, download_url)
        return utils.load_json(filepath).get('data')

    def update_mapping_file(self, filepath: str, download_url: str) -> None:
        """ Re-download a mapping file if it is in need of being updated.

        :param filepath: The file path to the mapping file.
        :param download_url: The download url for the mapping file.
        :return: None
        """
        if not os.path.exists(filepath):
            self.download_mapping_file(filepath, download_url)

        file_age = time.time() - os.path.getmtime(filepath)
        # Replace if the old file is 7 days old
        if file_age >= 603_800:
            logger.info("Downloading new mapping file")
            self.download_mapping_file(filepath, download_url)

    def get_anidbid_to_anilistid(self) -> dict:
        """
        Creates a mapping from anidbid to anilistid
        :return:
        """
        anidbid_to_anilistid = {}
        offline_database = self.load_anime_offline_database()
        for anime in offline_database:
            # Check that the sources has anidb and anilist
            sources = anime.get('sources')
            anidb_url = next((x for x in sources if x.startswith('https://anidb.net/anime/')), None)
            anilist_url = next((x for x in sources if x.startswith('https://anilist.co/anime/')), None)

            # Add these to the dictionary to map the two
            if None not in (anidb_url, anilist_url):
                anidbid = anidb_url.lstrip('https://anidb.net/anime/')
                anilistid = anilist_url.lstrip('https://anilist.co/anime/')

                # Make sure the two values are valid numbers
                if anidbid.isdigit() and anilistid.isdigit():
                    anidbid_to_anilistid[anidbid] = anilistid

        return anidbid_to_anilistid

    def get_tvdbid_to_anidbid(self) -> dict:
        """
        Creates a mapping from tvdbid to anidbid
        :return:
        """
        tvdbid_to_anidbid = {}
        for anime in list(self.load_tvdb_id_to_anidb_id_xml()):
            tvdbid = anime.get('tvdbid')
            season_number = anime.get('defaulttvdbseason')
            anidbid = anime.get('anidbid')
            # We only want shows that have all three of these
            if None not in (tvdbid, season_number, anidbid):
                show = tvdbid_to_anidbid.get(tvdbid, {})
                show[season_number] = anidbid
                tvdbid_to_anidbid[tvdbid] = show

        return tvdbid_to_anidbid

    def load_tvdb_id_to_anidb_id_xml(self) -> et.Element:
        """ Get an up to date version of the mapping from tvdb to anidb.

        :return: The up to date tvdb to anidb mapping file.
        """
        logger.info("Loading anime lists master mapping")
        download_url = 'https://raw.githubusercontent.com/ScudLee/anime-lists/master/anime-list-full.xml'
        filepath = 'data/tvdbid_to_anidbid.xml'

        self.update_mapping_file(filepath, download_url)

        return et.parse(filepath).getroot()

    @staticmethod
    def download_mapping_file(filepath: str, download_url: str) -> None:
        """ Download a mapping file to a specified filepath.

        :param filepath: The file path to save the downloaded mapping file.
        :param download_url: The download url for the mapping file.
        :return: None
        """
        logger.info("Downloading new mapping file")
        if os.path.exists(filepath):
            os.remove(filepath)

        urllib.request.urlretrieve(download_url, filepath)


class Mapper:
    def __init__(self):
        self.mapping = self.load_mapping_file()
        self.mapping_suggester = MappingSuggester()

    def load_mapping_file(self) -> List[SeriesMapping]:
        self.ensure_mapping_file_exists()

        mapping_data = utils.load_json(tvdbid_to_anilist_filepath)
        return [SeriesMapping(x) for x in mapping_data]

    @staticmethod
    def ensure_mapping_file_exists() -> None:
        """
        Makes sure the tvdbid_to_anilistid mapping file exists and if not creates it.
        :return:
        """
        if not os.path.exists(tvdbid_to_anilist_filepath):
            utils.save_json({}, tvdbid_to_anilist_filepath)

    def has_mapping(self, anime: PlexAnime) -> bool:
        series_mapping = self.find_series_mapping(anime.tvdbid)
        if series_mapping is None:
            return False

        season_mapping = series_mapping.find_season_mapping(anime.season_number)
        return season_mapping is not None

    def find_series_mapping(self, tvdbid: str) -> Optional[SeriesMapping]:
        return next((x for x in self.mapping if x.tvdbid == tvdbid), None)

    def get_anilistid_suggestion(self, tvdbid: str, season_number: str):
        return self.mapping_suggester.get_mapping_suggestion(tvdbid, season_number)

    def add_mapping(self, tvdbid: str, title: str, season_mapping: SeasonMapping):
        series_mapping = self.find_series_mapping(tvdbid)

        if series_mapping is None:
            # Create a new mapping for the series
            series_mapping = SeriesMapping()
            series_mapping.tvdbid = tvdbid
            series_mapping.name = title
            self.mapping.append(series_mapping)

        series_mapping.add_season_mapping(season_mapping)
        self.save_mapping()

    def save_mapping(self):
        utils.save_json(utils.serialize(self.mapping), tvdbid_to_anilist_filepath)
