import json
import logging
import logging.config
import os


def create_logger(name: str) -> logging.Logger:
    log_format = '%(asctime)s [%(name)s] %(message)s'
    log_date_format = '%d-%m-%Y %H:%M:%S'
    logging.basicConfig(format = log_format, datefmt = log_date_format)
    logger = logging.getLogger(name)
    logger.setLevel(logging.DEBUG)
    return logger


def load_json(filepath: str) -> dict:
    if not os.path.exists(filepath):
        save_json({}, filepath)

    with open(filepath, 'r', encoding = 'utf-8') as f:
        return json.load(f)


def save_json(data: dict, filepath: str):
    with open(filepath, 'w') as f:
        json.dump(data, f)


def serialize(obj):
    serialized_types = (str, int, str, float, bool)

    # Type is already serialized so we can just use that
    if type(obj) in serialized_types or obj is None:
        return obj

    # Each item in a list needs to be serialized
    if type(obj) == list:
        return [serialize(x) for x in obj]

    # Each key and value in a dict needs to be serialized
    if type(obj) == dict:
        return {serialize(k): serialize(v) for k, v in obj.items()}

    data = {}
    for prop in obj.__dict__:
        value = getattr(obj, prop)
        data[prop] = serialize(value)

    return data
