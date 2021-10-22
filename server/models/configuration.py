from typing import Optional

import fileManager

CONFIG_PATH = "data/config.json"


class Configuration:
    def __init__(self):
        self.plex_token: Optional[str] = None
        self.anilist_token: Optional[str] = None
        self.populate_values_from_config()

    def populate_values_from_config(self):
        config_data = self.load_from_file()
        for attr, value in self.__dict__.items():
            config_value = config_data.get(attr)
            if config_value is not None:
                setattr(self, attr, config_data.get(attr, value))

    def load_from_file(self) -> dict:
        return fileManager.load_json(CONFIG_PATH, self.__dict__)

    def save(self) -> None:
        fileManager.save_json(CONFIG_PATH, self.__dict__)
