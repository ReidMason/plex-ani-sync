import json
import logging
import time
from pprint import pprint

import coloredlogs
import requests

from config import Config

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)

config = Config()


class Anilist:
    class InvalidToken(Exception):
        pass

    def __init__(self, access_token: str):
        self.access_token = access_token
        self.username = self.get_username()
        self.user_list = self.fetch_user_list()

    def get_anime(self, anilist_id: str):
        return self.user_list.get(anilist_id)

    def send_query(self, query: str, variables: dict):
        url = 'https://graphql.anilist.co'

        headers = {
            'Authorization': 'Bearer ' + config.anilist_access_token,
            'Accept'       : 'application/json',
            'Content-Type' : 'application/json'
        }

        r = requests.post(
            url, headers = headers, json = {
                'query': query, 'variables': variables})

        content = json.loads(r.content.decode('utf-8'))

        # Check for invalid token errors
        errors = content.get('errors')
        if errors is not None and errors[0].get('message') == "Invalid token":
            raise Anilist.InvalidToken("Provided Anilist token is invalid.")

        return content

    def update_series(self, anilist_id: str, progress: int, status: str, log: bool = True) -> bool:
        if log:
            logger.warning(f"Updating {anilist_id} to {status}")

        query = '''
            mutation ($mediaId: Int, $status: MediaListStatus, $progress: Int) {
                SaveMediaListEntry (mediaId: $mediaId, status: $status, progress: $progress) {
                    id
                    status,
                    progress
                }
            }
            '''

        variables = {
            'mediaId' : anilist_id,
            'status'  : status,
            'progress': progress
        }

        # If there were no errors so the update was successful
        return self.send_query(query, variables).get('errors') is None

    def fetch_user_list(self):
        logger.warning("Fetching users lists from anilist")

        query = '''
            query ($username: String) {
            MediaListCollection(userName: $username, type: ANIME) {
                lists {
                name
                status
                isCustomList
                entries {
                    id
                    progress
                    status
                    media{
                        id
                        type
                        status
                        season
                        episodes
                    title {
                        romaji
                        english
                    }
                    }
                }
                }
            }
            }
            '''

        variables = {
            'username': self.username
        }

        anime_list = {}
        data = self.send_query(query, variables).get('data').get('MediaListCollection').get('lists')
        for anilist_list in data:
            if anilist_list.get('status') in ['CURRENT', 'PLANNING', 'COMPLETED', 'DROPPED', 'PAUSED', 'REPEATING']:
                for anime in anilist_list.get('entries'):
                    anime_list[str(anime.get('media').get('id'))] = anime

        time.sleep(1)
        return anime_list

    def get_username(self):
        query = '''
                    query {
                        Viewer {
                            name
                        }
                    }
                    '''

        return self.send_query(query, {}).get('data').get('Viewer').get('name')
