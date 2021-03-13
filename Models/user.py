from typing import Any, List, Optional, Union


class User:
    def __init__(self, user_data: dict):
        self.user_data = user_data
        # Required for plex token authentication
        self.plex_token: str = self.load_string_property_from_user_data('plex_token')
        self.plex_server_url: str = self.load_string_property_from_user_data('plex_server_url')

        # Required for plex username authentication
        self.plex_username: str = self.load_string_property_from_user_data('plex_username')
        self.plex_password: str = self.load_string_property_from_user_data('plex_password')
        self.plex_server_name: str = self.load_string_property_from_user_data('plex_server_name')

        # Anilist authentication
        self.anilist_token: str = self.load_string_property_from_user_data('anilist_token')

        self.paused_days_threshold: int = self.load_string_property_from_user_data('paused_days_threshold')
        self.dropped_days_threshold: int = self.load_string_property_from_user_data('dropped_days_threshold')

        self.plex_anime_libraries: List[str] = user_data.get('plex_anime_libraries')

    def load_string_property_from_user_data(self, property_name: str) -> any:
        prop_value: any = self.user_data.get(property_name, "")
        # Make value None if the string is empty or if the key doesn't exist
        if isinstance(prop_value, str):
            prop_value = prop_value.strip() if len(prop_value.strip()) > 0 else None

        return prop_value

    @property
    def can_use_plex_token_auth(self) -> bool:
        return self.plex_token is not None and self.plex_server_url is not None
