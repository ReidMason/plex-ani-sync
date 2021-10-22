import pytest
import fileManager
import os
import shutil

test_main_directory = "mainTestDirectory"
test_sub_directories = ["subTestDirectory1", "subTestDirectory2"]
test_paths = [os.path.join(test_main_directory, x) for x in test_sub_directories]


class TestCreateDirectoryPath:
    @pytest.fixture(autouse = True)
    def wrapper(self):
        yield

        # Cleanup any created directories
        if os.path.exists(test_main_directory):
            shutil.rmtree(test_main_directory)

    def test_directory_path_is_created(self):
        test_path = test_paths[0]
        fileManager.create_directory_path(test_path)
        assert os.path.exists(test_path) is True

    def test_directory_creation_action_is_returned(self):
        test_path = test_paths[0]
        # Directory was created
        directory_created = fileManager.create_directory_path(test_path)
        assert directory_created is True

        # Directory wasn't created
        directory_created = fileManager.create_directory_path(test_path)
        assert directory_created is False


class TestEnsureRequiredDirectoriesExist:
    @pytest.fixture(autouse = True)
    def wrapper(self):
        yield

        # Cleanup any created directories
        if os.path.exists(test_main_directory):
            shutil.rmtree(test_main_directory)

    def test_directory_paths_are_created(self):
        fileManager.REQUIRED_DIRECTORIES = test_paths
        fileManager.ensure_required_directories_exist()

        for path in test_paths:
            assert os.path.exists(path) is True

    def test_directory_creation_action_is_returned(self):
        fileManager.REQUIRED_DIRECTORIES = test_paths
        directory_created = fileManager.ensure_required_directories_exist()
        assert directory_created is True

        directory_created = fileManager.ensure_required_directories_exist()
        assert directory_created is False
