import pytest
import os
from config import Config
from services.mappingServices.mappingService import FribbAnimeMapping
import shutil
import time

config = Config()


class TestDownloadFribbAnimeListMapping:
    @pytest.fixture(autouse=True)
    def wrapper(self):
        # Cleanup any existing directories
        if os.path.exists(config._MAPPING_PATH):
            shutil.rmtree(config._MAPPING_PATH)
        yield

    @pytest.fixture(scope="class")
    def mapping_service(self):
        return FribbAnimeMapping()

    def test_fribb_mapping_file_download(self, mapping_service: FribbAnimeMapping):
        # Make sure the file doesn't exist to start
        mapping_file_exists = os.path.exists(
            mapping_service.anime_list_mapping_path
        )
        assert mapping_file_exists is False

        # Make sure the file exists after the download method has been called
        mapping_service.download_fribb_anime_list_mapping()
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        assert mapping_file_exists is True


class TestFribbMappingFileDownloadRequired:
    @pytest.fixture(autouse=True)
    def wrapper(self):
        # Cleanup any existing directories
        if os.path.exists(config._MAPPING_PATH):
            shutil.rmtree(config._MAPPING_PATH)
        yield

    @pytest.fixture(scope="class")
    def mapping_service(self):
        return FribbAnimeMapping()

    def test_fribb_mapping_file_download_required(self, mapping_service: FribbAnimeMapping):
        # Make sure the file exists to start
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        if not mapping_file_exists:
            mapping_service.download_fribb_anime_list_mapping()

        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        assert mapping_file_exists is True

        # The file has just been downloaded so it shouldn't need updating
        download_required = mapping_service.fribb_anime_list_needs_updating()
        assert download_required == False

        # Set the download threshold low to simulate the file going over the threshold
        old_value = mapping_service.mapping_update_threshold
        mapping_service.mapping_update_threshold = 1
        time.sleep(2)
        download_required = mapping_service.fribb_anime_list_needs_updating()
        # Put the threshold back
        mapping_service.mapping_update_threshold = old_value
        assert download_required == True
