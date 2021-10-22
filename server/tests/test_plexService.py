from plexapi.library import ShowSection
from services.plexService import PlexConnectionError, PlexService
from config import PLEX_SERVER_URL, PLEX_TOKEN
import pytest

INVALID_TOKEN = "invalid_token"
INVALID_PLEX_URL = "https://invalidurl"
VALID_LIBRARY_NAME = "anime"
INVALID_LIBRARY_NAME = "invalidLibrary"


def setup_plex_service() -> PlexService:
    return PlexService(PLEX_SERVER_URL, PLEX_TOKEN)


def setup_authenticated_plex_service() -> PlexService:
    plex_service = setup_plex_service()
    plex_service.authenticate()
    return plex_service


class Testauthenticate:
    def test_authentication_with_token(self):
        plex_service = setup_plex_service()
        plex_service.authenticate()

        assert plex_service.connection is not None

    def test_autnetication_with_invalid_token(self):
        plex_service = PlexService(PLEX_SERVER_URL, INVALID_TOKEN)

        with pytest.raises(PlexConnectionError):
            plex_service.authenticate()

        assert plex_service.connection is None

    def test_authentication_with_invalid_server_url(self):
        plex_service = PlexService(INVALID_PLEX_URL, PLEX_TOKEN)

        with pytest.raises(PlexConnectionError):
            plex_service.authenticate()

        assert plex_service.connection is None


class TestGetLibrary:
    @pytest.fixture(scope="class")
    def plex_service(self) -> PlexService:
        return setup_authenticated_plex_service()

    def test_get_existing_library(self, plex_service: PlexService):
        library = plex_service.get_library(VALID_LIBRARY_NAME)

        assert library is not None
        assert isinstance(library, ShowSection)
        assert library.title.lower() == VALID_LIBRARY_NAME.lower()

    def test_get_non_existant_library(self, plex_service: PlexService):
        assert plex_service.get_library(INVALID_LIBRARY_NAME) is None
