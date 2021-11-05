from typing import Callable, Generator, List, Optional

from plexapi.video import Season, Show
from config import Config
from plexapi.exceptions import Unauthorized, NotFound
from plexapi.library import ShowSection
from plexapi.myplex import MyPlexPinLogin
from plexapi.server import PlexServer
from models.plex.plexAnime import PlexAnime
from requests.exceptions import MissingSchema, ConnectionError
from services.mappingServices.plexIdToTvdbId import PlexIdToTvdbId

import utils

logger = utils.create_logger("PlexService")


def extract_tvdb_id_from_guid_field(plex_show: Show):
    guid = plex_show.guid.replace("com.plexapp.agents.thetvdb://", "")
    match = guid.rsplit("?", 1)

    try:
        return int(match[0])
    except ValueError or IndexError:
        return None


def extract_tvdb_id_from_guids_field(plex_show: Show):
    # This method takes around 0.3 seconds which is not ideal
    for guid in plex_show.guids:
        if "tvdb://" in guid.id:
            return guid.id.replace("tvdb://", "")


def get_plex_id(plex_show: Show):
    return plex_show.key.lstrip("/library/metadata/")


def get_cached_tvdb_id(plex_show: Show, plex_id_to_tvdb_id: PlexIdToTvdbId):
    plex_id = get_plex_id(plex_show)
    return plex_id_to_tvdb_id.get_tvdb_id(plex_id)


def extract_tvdb_id_from_guid(plex_id_to_tvdb_id: PlexIdToTvdbId, plex_show: Show):
    # First try and get the cached value
    tvdb_id = get_cached_tvdb_id(plex_show, plex_id_to_tvdb_id)

    if tvdb_id is not None:
        return tvdb_id

    # Next try to extract it from the guid field
    tvdb_id = extract_tvdb_id_from_guid_field(plex_show)

    # If that fails try and get it from the guids list
    if tvdb_id is None:
        tvdb_id = extract_tvdb_id_from_guids_field(plex_show)

    # If we are here that means we found a new non cached value
    if tvdb_id is not None:
        plex_id = get_plex_id(plex_show)
        plex_id_to_tvdb_id.add_new_mapping(plex_id, tvdb_id)

    return tvdb_id


class PlexConnectionError(Exception):
    pass


class PlexAuthService:
    def __init__(self, authenticated_callback: Optional[Callable] = None):
        self.plex_pin_login: Optional[MyPlexPinLogin] = MyPlexPinLogin()
        self.authenticated_callback: Optional[Callable] = authenticated_callback

    def generate_pin(self) -> str:
        self.plex_pin_login.run(self.pin_auth_callback)
        return self.plex_pin_login.pin

    def pin_auth_callback(self, token: str) -> None:
        config = Config()
        config.PLEX_TOKEN = token
        config.save()

        # Call the custom callback
        if self.authenticated_callback is not None:
            self.authenticated_callback()


class PlexClient:
    def __init__(self, server_url: str, plex_token: str):
        self.config = Config()
        self.server_url: str = server_url
        self.connection: Optional[PlexServer] = None

        # Credentials require for token authentication
        self.token: Optional[str] = plex_token

    @property
    def can_use_token_auth(self):
        return self.token is not None

    def authenticate(self) -> None:
        if self.server_url is None:
            raise Exception("Missing Plex server url")

        if self.token is None:
            raise Exception("Missing Plex access token")

        try:
            logger.info("Authenticating with Plex token")
            self.connection = PlexServer(self.server_url, self.token)
        except (MissingSchema, ConnectionError):
            logger.error(f"Unable to reach Plex server at {self.server_url}")
            raise PlexConnectionError(f"Unable to reach Plex server at {self.server_url}")
        except Unauthorized:
            logger.error(f"Unauthorized to access Plex server at {self.server_url}")
            raise PlexConnectionError(f"Unauthorized to access Plex server at {self.server_url}")

    def get_library(self, library_name: str) -> Optional[ShowSection]:
        try:
            return self.connection.library.section(library_name)
        except NotFound:
            logger.warning(f"Unable to find Plex library: {library_name}")

    def get_anime_libraries(self) -> List[ShowSection]:
        anime_libraries = [self.get_library(x) for x in self.config.ANIME_LIBRARIES]
        return [x for x in anime_libraries if x is not None]

    def get_media_in_library(self, library: ShowSection):
        return library.all()

    def get_all_anime(self) -> Generator[List[PlexAnime], None, None]:
        logger.info("Finding all anime")
        for library in self.get_anime_libraries():
            for anime in self.get_anime_in_library(library):
                yield anime

    def get_anime_in_library(self, library) -> Generator[List[PlexAnime], None, None]:
        plex_id_to_tvdb_id = PlexIdToTvdbId()

        library_media = self.get_media_in_library(library)
        for anime in library_media:
            tvdb_id = extract_tvdb_id_from_guid(plex_id_to_tvdb_id, anime)

            anime_seasons = self.get_all_seasons_for_anime(anime)
            yield [PlexAnime(x, tvdb_id, anime.year) for x in anime_seasons]

    def get_all_seasons_for_anime(self, anime) -> Generator[Season, None, None]:
        for season in anime.seasons():
            yield season