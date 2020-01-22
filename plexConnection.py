import logging
from typing import List

import coloredlogs
from plexapi.video import Show as plexapiShow
from plexapi.server import PlexServer

from anime import Anime

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)

# Stop utllib3 spamming the logs
logging.getLogger('urllib3.connectionpool').setLevel(logging.WARNING)


class PlexConnection(PlexServer):
    class PlexServerUnreachable(Exception):
        pass

    class InvalidPlexToken(Exception):
        pass

    def __init__(self, server_url: str, server_token: str) -> None:
        """ Connects to plex server with the given url and token.

        :param server_url: The url to the target plex server.
        :param server_token: The token for the target server.
        :return: None
        """
        logger.warning("Connecting to plex server")
        super().__init__(server_url, server_token)
        logger.debug("Plex connection established")

    def get_shows(self, library: str) -> List[plexapiShow]:
        """ Gets all the shows in a given library.

        :param library: The name of the target library.
        :return: A list of Show objects from the target library.
        """
        logger.debug(f"Getting shows for library {library}")

        # If the library doesn't exist return empty list
        if library not in [x.title for x in self.library.sections()]:
            return []

        return self.library.section(library).all()

    def get_anime(self, library: str) -> List[Anime]:
        """ Loads all the shows and seasons in a library into a list of anime objects.

        :param library: The Plex library to look through.
        :return: A list of Anime objects representing the shows in the targeted library.
        """
        anime = []
        for show in self.get_shows(library):
            tvdb_id = show.guid.rsplit('/')[-1].split('?')[0]
            for season in [x for x in show.seasons() if x.title.lower() != 'specials']:
                watched_episodes = len([x for x in season.episodes() if x.isWatched])
                anime.append(Anime(show.title, tvdb_id, str(season.seasonNumber), watched_episodes))

        return anime
