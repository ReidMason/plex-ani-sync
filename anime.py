import logging
from dataclasses import dataclass
from pprint import pprint
from typing import Optional

import coloredlogs

from mapping import Mapping
from anilist import Anilist
from config import Config

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


@dataclass
class Anime:
    """ A object representing an anime holding various data values and providing methods for calculating and obtaining
    information about the anime.
    """
    # Class variables
    config = Config()
    mapping = Mapping()
    anilist = Anilist(config.anilist_access_token)

    # Instance variables
    title: str
    tvdb_id: str
    season_number: str
    watched_episodes: int

    def __post_init__(self) -> None:
        """ Defines other instance variables that require more complex assignments.

        :return: None
        """
        self.anilist_id = self.obtain_anilist_id()
        self.total_episodes = self.obtain_total_episodes()
        self.anilist_progress = (Anime.anilist.get_anime(self.anilist_id) or {}).get('progress')
        self.anilist_status = (Anime.anilist.get_anime(self.anilist_id) or {}).get('status')

        self.status = self.equate_watch_status()

    def obtain_anilist_id(self) -> Optional[str]:
        """ Obtains the matching Anilist id from the mapping files.

        :return: The Anilist id for the anime or None if there was no id mapped.
        """
        anilist_id = Anime.mapping.get_anilist_id(self.tvdb_id, self.title, self.season_number)
        if anilist_id is None:
            Anime.mapping.add_to_mapping_errors(self)
        return anilist_id

    def obtain_total_episodes(self) -> Optional[int]:
        """ Obtains the total number of episodes from the Anilist data.

        :return: The total number of episodes
                 or None if the anime isn't already on Anilist or if the total episodes isn't known by Anilist.
        """
        if (anime := Anime.anilist.get_anime(self.anilist_id)) is None:
            return None
        x = anime.get('media', {}).get('episodes')

        return x

    def equate_watch_status(self) -> str:
        """ Using the data available determine the watch status of the anime. Whether it is completed, planning to be
        watched or currently being watched.

        :return: The current watch status of the anime.
        """
        if self.total_episodes is not None and self.watched_episodes >= self.total_episodes:
            return 'COMPLETED'

        elif self.watched_episodes == 0:
            return 'PLANNING'

        else:
            return 'CURRENT'

    def update_required(self) -> bool:
        """ Check whether or not the anime is in need of being updated on Anilist as more episodes have been watched.

        :return: Boolean of whether or not the anime needs to be updated on Anilist.
        """
        # If there is no anilist id then there was no mapping so don't try and update
        # If the series is already listed as completed don't update it
        if self.anilist_id is None or self.anilist_status == 'COMPLETED':
            return False

        # If it is dropped or paused don't update unless episodes watched has increased
        if self.anilist_status in ['DROPPED', 'PAUSED'] and self.watched_episodes <= self.anilist_progress:
            return False

        status = self.status != self.anilist_status
        episodes = self.anilist_progress is None or self.watched_episodes > self.anilist_progress
        needs_update = status or episodes
        return needs_update

    def update_on_anilist(self) -> None:
        """ Update the anime on Anilist. This will update using the information stored in this object.

        :return: None
        """
        logger.info(f"Updating {self.title} Season {self.season_number} on Anilist")
        successful = Anime.anilist.update_series(self.anilist_id, self.watched_episodes, self.status)
        if not successful:
            logger.info(f"Update failed. Anilist id for {self.title} Season {self.season_number} invalid")
            Anime.mapping.add_to_mapping_errors(self)
