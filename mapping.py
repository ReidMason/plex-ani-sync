import logging
import os
import time
import urllib.request
import xml.etree.ElementTree as et
from typing import Optional

import coloredlogs

import utils

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


def load_tvdb_id_to_anidb_id_xml() -> et.Element:
    """ Get an up to date version of the mapping from tvdb to anidb.

    :return: The up to date tvdb to anidb mapping file.
    """
    download_url = 'https://raw.githubusercontent.com/ScudLee/anime-lists/master/anime-list-full.xml'
    filepath = 'data/tvdbid_to_anidbid.xml'
    # self.update_mapping_xml()
    update_mapping_file(filepath, download_url)
    return et.parse(filepath).getroot()


def load_anime_offline_database() -> dict:
    """ Get an up to date version of the mapping anime offline database mapping file.

    :return: The up to date anime offline database mapping file.
    """
    download_url = 'https://raw.githubusercontent.com/manami-project/anime-offline-database/master/anime-offline-database.json'
    filepath = 'data/anime-offline-database.json'
    update_mapping_file(filepath, download_url)
    # self.update_anime_offline_database()
    return utils.load_json(filepath).get('data')


def update_mapping_file(filepath: str, download_url: str) -> None:
    """ Re-download a mapping file if it is in need of being updated.

    :param filepath: The file path to the mapping file.
    :param download_url: The download url for the mapping file.
    :return: None
    """
    logger.info("Updating mapping file")
    if not os.path.exists(filepath):
        download_mapping_file(filepath, download_url)

    file_age = time.time() - os.path.getctime(filepath)
    # Replace if the old file is 7 days old
    if file_age >= 603_800:
        download_mapping_file(filepath, download_url)


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


def load_tvdb_id_to_anilist_id() -> dict:
    """ Get the tvdb to anilist mapping file.

    :return: The tvdb to anilist mapping file
    """
    logger.debug("Loading tvdb_id to anilist_id")
    return utils.load_json('data/tvdbid_to_anilistid.json')


class Mapping:
    """ A class that handles mapping show ids from different sources so that we can convert between the two. """

    # Load the mapping files for use.s
    xml_tvdb_id_to_anidb_id = load_tvdb_id_to_anidb_id_xml()
    tvdb_id_to_anilist_id = load_tvdb_id_to_anilist_id()
    anime_offline_database = load_anime_offline_database()

    def save_tvdb_id_to_anilist_id(self):
        """ Save the tvdbid to anilist mapping file. """
        utils.save_json(self.tvdb_id_to_anilist_id, 'data/tvdbid_to_anilistid.json')

    def get_anilist_id(self, tvdb_id: str, title: str, season: str) -> Optional[str]:
        """ Get the anilist id from a provided tvdb id, title and season number.

        :param tvdb_id: The tvdb id of the show you want to target.
        :param title: The title of the show you want to target.
        :param season: The season number of the show you want to target.
        :return: The anilist id of the show or None if a corresponding id wasn't found.
        """
        # Check if mapping already exists
        anilist_id = self.tvdb_id_to_anilist_id.get(tvdb_id, {}).get(season)
        if anilist_id is not None:
            return anilist_id

        # Create a new mapping
        return self.create_tvdb_id_to_anilist_id_mapping(tvdb_id, title, season)

    def create_tvdb_id_to_anilist_id_mapping(self, tvdb_id: str, title: str, season: str) -> Optional[str]:
        """ Creates abd saves a tvdb to anilist mapping using the current mapping files.

        :param tvdb_id: The tvdb id of the show you want to create the mapping for.
        :param title: The title for the show you want to create the mapping for.
        :param season: The season number for the show you want to create the mapping for.
        :return: The anilist id that has been mapped or None if it was unable to be mapped.
        """
        logger.warning(f"Creating new anime mapping for {title} Season {season}")
        anilist_id = None
        if (anidb_id := self.get_anidb_id_from_tvdb_id(tvdb_id, season)) is not None:
            anilist_id = self.get_anilist_id_from_aod(anidb_id)

        self.tvdb_id_to_anilist_id[tvdb_id] = {**self.tvdb_id_to_anilist_id.get(tvdb_id, {}), **{season: anilist_id}}
        self.save_tvdb_id_to_anilist_id()

        return anilist_id

    def get_anidb_id_from_tvdb_id(self, tvdb_id: str, season: str) -> Optional[str]:
        """ Gets the anidb id from the tvdb id from the current mapping files.

        :param tvdb_id: The tvdb id of the show you want to target.
        :param season: The season number of the show you want to target.
        :return: The anidb id for the targeted show or None if a mapping wasn't found.
        """
        for anime in list(self.xml_tvdb_id_to_anidb_id):
            if anime.get('tvdbid') == tvdb_id and anime.get('defaulttvdbseason') == season:
                return anime.get('anidbid')
        return None

    def get_anilist_id_from_aod(self, anidb_id: str) -> Optional[str]:
        """ Finds the anilist id from a given anidb id using the anime offline database.

        :param anidb_id: The anidb id of the show you want to target.
        :return: The anilist id for the targeted show or None if a mapping wasn't found.
        """
        for anime in self.anime_offline_database:
            if f'https://anidb.net/anime/{anidb_id}' in anime.get('sources'):
                for source in anime.get('sources'):
                    if source.startswith('https://anilist.co/anime/'):
                        return source.rsplit('/')[-1]
        return None

    def add_to_mapping_errors(self, anime) -> None:
        """ Adds an anime to the mapping errors file to be manually added later.

        :param anime: The anime that has a mapping error.
        :return: None
        """
        logger.debug(f"Adding {anime.title} Season {anime.season_number} to mapping errors")
        mapping_errors = utils.load_json('data/mapping_errors.json')
        if anime.tvdb_id not in mapping_errors:
            mapping_errors[anime.tvdb_id] = {'title'  : anime.title,
                                             'seasons': []}

        if anime.season_number not in mapping_errors[anime.tvdb_id]['seasons']:
            mapping_errors[anime.tvdb_id]['seasons'].append(anime.season_number)

        self.save_mapping_errors(mapping_errors)

    def save_mapping_errors(self, mapping_errors: dict) -> None:
        """ Saves the mapping errors file.

        :param mapping_errors: The mapping errors data to be saved.
        :return: None
        """
        utils.save_json(mapping_errors, 'data/mapping_errors.json')
