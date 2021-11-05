from plexapi.library import ShowSection
from services.plexClient import PlexConnectionError, PlexClient
from config import Config
import pytest

INVALID_TOKEN = "invalid_token"
INVALID_PLEX_URL = "https://invalidurl"
VALID_LIBRARY_NAME = "anime"
INVALID_LIBRARY_NAME = "invalidLibrary"

config = Config()


def setup_plex_service() -> PlexClient:
    return PlexClient(config.PLEX_SERVER_URL, config.PLEX_TOKEN)


def setup_authenticated_plex_service() -> PlexClient:
    plex_service = setup_plex_service()
    plex_service.authenticate()
    return plex_service


class Testauthenticate:
    def test_authentication_with_token(self):
        plex_service = setup_plex_service()
        plex_service.authenticate()

        assert plex_service.connection is not None

    def test_autnetication_with_invalid_token(self):
        plex_service = PlexClient(config.PLEX_SERVER_URL, INVALID_TOKEN)

        with pytest.raises(PlexConnectionError):
            plex_service.authenticate()

        assert plex_service.connection is None

    def test_authentication_with_invalid_server_url(self):
        plex_service = PlexClient(INVALID_PLEX_URL, config.PLEX_TOKEN)

        with pytest.raises(PlexConnectionError):
            plex_service.authenticate()

        assert plex_service.connection is None


class TestGetLibrary:
    @pytest.fixture(scope="class")
    def plex_service(self) -> PlexClient:
        return setup_authenticated_plex_service()

    def test_get_existing_library(self, plex_service: PlexClient):
        library = plex_service.get_library(VALID_LIBRARY_NAME)

        assert library is not None
        assert isinstance(library, ShowSection)
        assert library.title.lower() == VALID_LIBRARY_NAME.lower()

    def test_get_non_existant_library(self, plex_service: PlexClient):
        assert plex_service.get_library(INVALID_LIBRARY_NAME) is None
