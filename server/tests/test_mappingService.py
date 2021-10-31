import pytest
import os
from config import Config
from services.mappingServices.mappingService import MappingService
import shutil

config = Config()


class TestLoadAnimeMapping:
    @pytest.fixture(autouse=True)
    def wrapper(self):
        # Cleanup any existing directories
        if os.path.exists(config._MAPPING_PATH):
            shutil.rmtree(config._MAPPING_PATH)
        yield

    @pytest.fixture(scope="class")
    def mapping_service(self):
        return MappingService()

    def test_mapping_file_is_created(self, mapping_service: MappingService):
        mapping_service.load_mappings()
        assert mapping_service.mappings is not None


class TestSaveAnimeMapping:
    @pytest.fixture(autouse=True)
    def wrapper(self):
        # Cleanup any existing directories
        if os.path.exists(config._MAPPING_PATH):
            shutil.rmtree(config._MAPPING_PATH)
        yield

    @pytest.fixture(scope="class")
    def mapping_service(self):
        return MappingService()

    def test_mapping_file_is_saved(self, mapping_service: MappingService):
        # Make sure the file doesn't exist to start
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        if mapping_file_exists:
            os.remove(mapping_service.anime_list_mapping_path)

        assert mapping_file_exists is False

        mapping_service.save_mapping()
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        assert mapping_file_exists is True


class TestEnsureMappingFileExsits:
    @pytest.fixture(autouse=True)
    def wrapper(self):
        # Cleanup any existing directories
        if os.path.exists(config._MAPPING_PATH):
            shutil.rmtree(config._MAPPING_PATH)
        yield

    @pytest.fixture(scope="class")
    def mapping_service(self):
        return MappingService()

    def test_mapping_file_is_created(self, mapping_service: MappingService):
        # Make sure the file doesn't exist to start
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        if mapping_file_exists:
            os.remove(mapping_service.anime_list_mapping_path)

        assert mapping_file_exists is False

        # Make sure the file exists after the download method has been called
        mapping_service.ensure_mapping_file_exists()
        mapping_file_exists = os.path.exists(mapping_service.anime_list_mapping_path)
        assert mapping_file_exists is True
