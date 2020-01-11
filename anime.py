import logging
from pprint import pprint

import coloredlogs

from mapping import Mapping
from anilist import Anilist
from config import Config

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


class Anime:
    config = Config()
    mapping = Mapping()
    anilist = Anilist(config.anilist_access_token, config.anilist_username)

    def __init__(self, title: str, tvdb_id: str, season_number: str, watched_episodes: int):
        self.title = title
        self.tvdb_id = tvdb_id
        self.season_number = season_number
        self.watched_episodes = watched_episodes
        self.anilist_id = self.get_anilist_id()

        self.total_episodes = self.get_total_episodes()
        self.anilist_progress = (Anime.anilist.get_anime(self.anilist_id) or {}).get('progress')
        self.anilist_status = (Anime.anilist.get_anime(self.anilist_id) or {}).get('status')

        self.status = self.equate_watch_status()

    def get_anilist_id(self):
        anilist_id = Anime.mapping.get_anilist_id(self.tvdb_id, self.title, self.season_number)
        if anilist_id is None:
            Anime.mapping.add_to_mapping_errors(self)
        return anilist_id

    def get_total_episodes(self):
        anime = Anime.anilist.get_anime(self.anilist_id)
        if anime is None:
            return None

        return anime.get('media', {}).get('episodes')

    def equate_watch_status(self):
        if self.total_episodes is not None and self.watched_episodes >= self.total_episodes:
            return 'COMPLETED'

        elif self.watched_episodes == 0:
            return 'PLANNING'

        else:
            return 'CURRENT'

    def update_required(self):
        # If there is no anilist id then there was no mapping so don't try and update
        # If the series is already listed as completed don't update it
        if self.anilist_id is None or self.anilist_status == 'COMPLETED':
            return False

        # If it is dropped or paused don't update unless episodes watched has increased
        if self.anilist_status in ['DROPPED', 'PAUSED'] and self.watched_episodes <= self.anilist_progress:
            return False

        status = self.status != self.anilist_status
        episodes = self.anilist_progress is None or self.watched_episodes > self.anilist_progress

        return status or episodes

    def update_on_anilist(self):
        logger.info(f"Updating {self.title} Season {self.season_number} on Anilist")
        successful = Anime.anilist.update_series(self.anilist_id, self.watched_episodes, self.status, log = False)
        if not successful:
            logger.info(f"Update failed. Anilist id for {self.title} Season {self.season_number} invalid")
            Anime.mapping.add_to_mapping_errors(self)

    def __repr__(self):
        return f"Title: {self.title}\n  " \
               f"tvdb_id: {self.tvdb_id}\n  " \
               f"anilist_id: {self.anilist_id}\n  " \
               f"season: {self.season_number}\n  " \
               f"Watched episodes: {self.watched_episodes}\n  " \
               f"Total episodes: {self.total_episodes}\n  " \
               f"Anilist progress: {self.anilist_progress}\n  " \
               f"Anilist status: {self.anilist_status}\n  " \
               f"Status: {self.status}"
