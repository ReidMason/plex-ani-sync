import os
import json
from json import JSONDecodeError
from typing import List, Optional

live = os.environ.get("IS_LIVE", "false").lower() == "true"


class Config:
    def __init__(self) -> None:
        # Required paths
        self._DATA_PATH = "/data" if live else "data"
        self._MAPPING_PATH = os.path.join(self._DATA_PATH, "mapping")
        self._LOGS_PATH = os.path.join(self._DATA_PATH, "logs")

        self.PLEX_SERVER_URL: Optional[str] = None
        self.PLEX_TOKEN: Optional[str] = None

        self.ANILIST_TOKEN: Optional[str] = None

        self.ANIME_LIBRARIES: List[str] = ["Anime"]
        self.MARK_UNWATCHED_EPISODES_AS_PLANNING: bool = False

        self.DAYS_UNTIL_PAUSED: int = 14
        self.DAYS_UNTIL_DROPPED: int = 31

        self.SYNC_CRONTIME: str = "0 19 * * *"
        self.SYNC_SCHEDULE_ENABLED: bool = True

        self.DATE_FORMAT: str = "%d-%m-%Y %H:%M:%S"
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

            # Try and load the config data a few times before giving up
            attempts = 0
            while attempts < 10:
                try:
                    config_data = json.load(f)
                    break
                except JSONDecodeError:
                    attempts += 1
                    if attempts > 9:
                        raise Exception("Error parsing config file")

            fields = [x for x in self.__dict__ if not x.startswith("_")]
            for field in fields:
                data_value = config_data.get(field)
                if data_value is not None:
                    setattr(self, field, data_value)

            # If the config file is missing values we need to save in order to add them
            config_keys = set([x for x in self.__dict__.keys() if not x.startswith("_")])
            config_data_keys = set(config_data.keys())
            if config_keys != config_data_keys:
                self.save()

    def save(self) -> None:
        with open(self.config_path, 'w') as f:
            data = {k: v for k, v in self.__dict__.items() if not k.startswith("_")}
            json.dump(data, f)
