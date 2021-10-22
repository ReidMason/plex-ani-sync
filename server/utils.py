import logging
import re
from models.configuration import Configuration


def create_logger(name: str) -> logging.Logger:
    log_format = '%(asctime)s [%(name)s] %(message)s'
    log_date_format = '%d-%m-%Y %H:%M:%S'
    logging.basicConfig(format=log_format, datefmt=log_date_format)
    logger = logging.getLogger(name)
    logger.setLevel(logging.DEBUG)
    return logger


def remove_non_alphanumerics(text: str) -> str:
    return re.sub(r'[^\w ]', '', text)
