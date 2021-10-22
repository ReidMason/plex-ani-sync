from typing import List, Optional, Union

import plexapi.exceptions
from plexapi.library import MovieSection, ShowSection
from plexapi.myplex import MyPlexAccount
from plexapi.server import PlexServer
from plexapi.video import Episode, Movie, Season, Show

import utils
from Models.user import User

logger = utils.create_logger(__name__)


class PlexAnime:
    def __init__(self, plex_series: Union[Season, Movie] = None):
        if plex_series is None:
            return

        plex_show: Show = plex_series.show()
        self.title: str = str(plex_show.title)
        self.tvdbid: str = str(plex_show.guid.rsplit('/')[-1].split('?')[0])
        self.season_number: str = str(plex_series.seasonNumber)
        self.display_name: str = f'{self.title} - Season {self.season_number}'
        self.episodes: List[Episode] = plex_series.episodes()

    @property
    def episodes_watched(self) -> int:
        return len([x for x in self.episodes if x is not None and x.isWatched])

    def find_episode_index(self, episode_number: int):
        try:
            return [x.index for x in self.episodes].index(episode_number)
        except ValueError:
            logger.error(
                f"Unable to find episode number {episode_number} in {self.title} Sesaon {self.season_number}")
            return None


class Plex:
    def __init__(self, user: User) -> None:
        self.user = user
        self.plex_connection: PlexServer = self.create_plex_connection()
        logger.info("Plex connection established")
        self.all_anime: List[PlexAnime] = self.get_all_anime()

    def create_plex_connection(self) -> PlexServer:
        # Authenticate using token
        if self.user.can_use_plex_token_auth:
            return self.authenticate_with_token()

        # Authenticate using username and password
        return self.authenticate_with_credentials()

    def authenticate_with_token(self) -> Optional[PlexServer]:
        try:
            logger.info("Authenticating with Plex token")
            return PlexServer(self.user.plex_server_url, self.user.plex_token)
        except ConnectionError:
            logger.error(f"Unable to reach Plex server at {self.user.plex_server_url}")
            raise Exception

    def authenticate_with_credentials(self) -> Optional[PlexServer]:
        try:
            logger.info("Authenticating with Plex credentials")
            account = MyPlexAccount(self.user.plex_username, self.user.plex_password)
            return account.resource(self.user.plex_server_name).connect()
        except ConnectionError:
            logger.error(f"Unable to reach Plex server {self.user.plex_server_name}")
            raise Exception

    def get_library(self, library_name: str) -> Union[MovieSection, ShowSection]:
        try:
            return self.plex_connection.library.section(library_name)
        except plexapi.exceptions.NotFound:
            logger.warning(f"Unable to find Plex library: {library_name}")

    def get_anime_libraries(self):
        anime_libraries = [self.get_library(x) for x in self.user.plex_anime_libraries]
        return [x for x in anime_libraries if x is not None]

    def get_media_in_library(self, library: ShowSection):
        return library.all()

    def get_all_anime(self) -> List[PlexAnime]:
        logger.info("Finding all anime")
        all_anime = []
        for library in self.get_anime_libraries():
            library_media = self.get_media_in_library(library)
            for anime in library_media:
                # logger.info(f"Processing library {library.title} - {i + 1}/{len(library_media)} {anime.title}")
                if isinstance(anime, Show):
                    # Each season is a different anime on anilist to split them up
                    for season in anime.seasons():
                        all_anime.append(PlexAnime(season))

        return all_anime

    def find_anime(self, tvdbid: str, season_number: str) -> Optional[PlexAnime]:
        return next((x for x in self.all_anime if x.tvdbid == tvdbid and x.season_number == season_number), None)

    def find_anime_by_tvdbid(self, tvdbid: str) -> List[PlexAnime]:
        return [x for x in self.all_anime if x.tvdbid == tvdbid]
