import os

ANILIST_API_BASE_URL = "https://graphql.anilist.co"
TIME_TO_RUN = os.getenv("TIME_TO_RUN")

# Configure user settings
USER = {
    # Login using username and password (slower)
    "plex_username"         : os.getenv('PLEX_USERNAME'),
    "plex_password"         : os.getenv('PLEX_PASSWORD'),
    "plex_server_name"      : os.getenv('PLEX_SERVER_NAME'),
    # Authenticate using a token (faster)
    "plex_server_url"       : os.getenv('PLEX_SERVER_URL'),
    "plex_token"            : os.getenv('PLEX_TOKEN'),

    # Anilist authentication
    "anilist_token"         : os.getenv('ANILIST_TOKEN'),

    # Plex config
    "plex_anime_libraries"  : os.getenv('PLEX_ANIME_LIBRARIES', '').split(','),
    # Days of no updates for anime to be marked as PAUSED
    "paused_days_threshold" : 14,
    # Days of no updates for anime to be marked as DROPPED
    "dropped_days_threshold": 31
}
