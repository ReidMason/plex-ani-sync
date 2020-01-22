import json
import os
from typing import Union


def load_json(filepath: str) -> Union[dict, list]:
    """ Loads a json file from a specified file path.

    :param filepath: The file path to load from.
    :return: The contents of the json file.
    """
    if not os.path.exists(filepath):
        save_json({}, filepath)

    with open(filepath, 'r', encoding = 'utf-8') as f:
        return json.load(f)


def save_json(data: Union[list, dict], filepath: str) -> None:
    """ Save a dictionary of list in a json file format.

    :param data: The data to be saved.
    :param filepath: The filepath to save the data.
    :return: None
    """
    with open(filepath, 'w', encoding = 'utf-8') as f:
        json.dump(data, f)
