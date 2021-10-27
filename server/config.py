import os
import json
import os

# Required paths
DATA_PATH = "data"
MAPPING_PATH = os.path.join(DATA_PATH, "mapping")
LOGS_PATH = os.path.join(DATA_PATH, "logs")

REQUIRED_DIRECTORIES = [DATA_PATH, MAPPING_PATH, LOGS_PATH]

with open(os.path.join(DATA_PATH, "config.json"), 'r') as f:
    config_data = json.load(f)


PLEX_SERVER_URL = config_data.get('plex_server_url')
PLEX_TOKEN = config_data.get('plex_token')

ANILIST_TOKEN = config_data.get('anilist_token')

ANIME_LIBRARIES = config_data.get('anime_libraries')
MARK_UNWATCHED_EPISODES_AS_PLANNING = config_data.get('mark_unwatched_episodes_as_planning')
TIME_UNTIL_PAUSED = config_data.get('time_until_paused')
TIME_UNTIL_DROPPED = config_data.get('time_until_dropped')
