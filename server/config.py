import os
import json
from json import JSONDecodeError
from typing import List


class Config:
    def __init__(self) -> None:
        # Required paths
        self._DATA_PATH = "data"
        self._MAPPING_PATH = os.path.join(self._DATA_PATH, "mapping")
        self._LOGS_PATH = os.path.join(self._DATA_PATH, "logs")

        self.PLEX_SERVER_URL: str = None
        self.PLEX_TOKEN: str = None

        self.ANILIST_TOKEN: str = None

        self.ANIME_LIBRARIES: List[str] = ["Anime"]
        self.MARK_UNWATCHED_EPISODES_AS_PLANNING: bool = False

        self.DAYS_UNTIL_PAUSED: int = 14
        self.DAYS_UNTIL_DROPPED: int = 31
        self.load_config_data()

    @property
    def config_path(self):
        return os.path.join(self._DATA_PATH, "config.json")

    @property
    def REQUIRED_DIRECTORIES(self) -> List[str]:
        return [self._DATA_PATH, self._MAPPING_PATH, self._LOGS_PATH]

    def load_config_data(self):
        # Create the config file if it doesn't exist
        if not os.path.exists(self.config_path):
            with open(self.config_path, 'w') as f:
                json.dump({}, f)

        # Load the data from the config file
        with open(self.config_path, 'r') as f:
            try:
                config_data = json.load(f)
            except JSONDecodeError:
                raise Exception("Error parsing config file")

            fields = [x for x in self.__dict__ if not x.startswith("_")]
            for field in fields:
                data_value = config_data.get(field)
                if data_value is not None:
                    setattr(self, field, data_value)
        self.save()

    def save(self) -> None:
        with open(self.config_path, 'w') as f:
            data = {k: v for k, v in self.__dict__.items() if not k.startswith("_")}
            json.dump(data, f)
