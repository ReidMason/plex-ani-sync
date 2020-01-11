import logging

import coloredlogs

from anilist import Anilist
from plexConnection import PlexConnection
from config import Config

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)

config = Config()


# logger.debug('Debug')  # Green
# logger.info('Debug')  # Red
# logger.warning('Debug')  # Orange

def start_sync():
    logger.debug("Sync started!")
    plex_connection = PlexConnection(config.server_url, config.server_token)
    plex_anime = plex_connection.get_anime(config.libraries[0])

    # Check anime that are out of sync with anilist
    logger.debug("Checking for any required updates")
    for anime in plex_anime:
        if anime.update_required():
            anime.update_on_anilist()

    # Go through the list and mark any shows that have all their episodes watched as completed
    logger.debug("Fixing leftover completed shows")
    anilist = Anilist(config.anilist_access_token, config.anilist_username)
    for id, data in anilist.user_list.items():
        if data.get('progress') == data.get('media').get('episodes') and data.get('status') != 'COMPLETED':
            anilist.update_series(id, data.get('progress'), 'COMPLETED')

    logger.debug("Sync complete!")
