import json
import os
from typing import List, Union

from config import Config


def ensure_required_directories_exist() -> bool:
    """ Creates required directories if they don't already exist

    :return: Whether any directories were created
    """
    config = Config()
    return any([create_directory_path(path) for path in config.REQUIRED_DIRECTORIES])


def save_json(filepath: str, data: Union[dict, List[dict]]) -> None:
    ensure_required_directories_exist()
    with open(filepath, 'w') as f:
        json.dump(data, f)


def load_json(filepath: str, default_data: Union[dict, List[dict]] = None) -> Union[dict, List[dict]]:
    ensure_required_directories_exist()
    if not os.path.exists(filepath):
        save_json(filepath, default_data if default_data is not None else {})

    with open(filepath, 'r') as f:
        return json.load(f)


def create_directory_path(path: str) -> bool:
    """ Recursively creates a directory.

    :param path: The directory path to create
    :return: Whether a new directory was created
    """

    if not os.path.exists(path):
        os.makedirs(path)
        return True

    return False
