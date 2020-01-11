import json
import logging
import time

import coloredlogs
import requests

from config import Config

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)

config = Config()


class Anilist:

    def __init__(self, access_token: str, username: str):
        self.access_token = access_token
        self.user_list = self.fetch_user_list(username)

    def get_anime(self, anilist_id: str):
        return self.user_list.get(anilist_id)

    def update_series(self, anilist_id: str, progress: int, status: str, log: bool = True) -> bool:
        if log:
            logger.info(f"Updating {anilist_id} to {status}")
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

        url = 'https://graphql.anilist.co'

        headers = {
            'Authorization': 'Bearer ' + self.access_token,
            'Accept'       : 'application/json',
            'Content-Type' : 'application/json'
        }

        data = {'query': query, 'variables': variables}

        r = requests.post(
            url, headers = headers, json = data)

        # If there were no errors so the update was successful
        return json.loads(r.content.decode('utf-8')).get('errors') is None

    def fetch_user_list(self, username: str):
        logger.info("Fetching users lists from anilist")
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
                    repeat
                    media{
                    id
                    type
                    format
                    status
                    source
                    season
                    episodes
                    startDate {
                        year
                        month
                        day
                    }
                    endDate {
                        year
                        month
                        day
                    }
                    title {
                        romaji
                        english
                        native
                    }
                    }
                }
                }
            }
            }
            '''

        variables = {
            'username': username
        }

        url = 'https://graphql.anilist.co'

        headers = {
            'Authorization': 'Bearer ' + config.anilist_access_token,
            'Accept'       : 'application/json',
            'Content-Type' : 'application/json'
        }

        response = requests.post(
            url, headers = headers, json = {
                'query': query, 'variables': variables})

        anime_list = {}
        data = json.loads(response.content).get('data').get('MediaListCollection').get('lists')
        for anilist_list in data:
            if anilist_list.get('status') in ['CURRENT', 'PLANNING', 'COMPLETED', 'DROPPED', 'PAUSED', 'REPEATING']:
                for anime in anilist_list.get('entries'):
                    anime_list[str(anime.get('media').get('id'))] = anime

        time.sleep(1)
        return anime_list
