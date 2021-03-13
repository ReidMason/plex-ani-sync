import json
import time
from datetime import datetime
from typing import List, Optional

import requests

import utils
from config import ANILIST_API_BASE_URL

logger = utils.create_logger(__name__)


class AnilistAnime:
    def __init__(self, anilist_json: dict):
        media_json = anilist_json.get('media')

        self.title_romaji: str = media_json.get('title').get('romaji')
        self.title: str = media_json.get('title').get('english') or self.title_romaji
        self.id: str = str(media_json.get('id'))
        # When the total episodes aren't known it gives None so convert that to a 0
        self.total_episodes: int = media_json.get('episodes') or 0
        self.episodes_watched: int = anilist_json.get('progress')
        self.status: str = anilist_json.get('status')
        self._last_updated: int = anilist_json.get('updatedAt')

    def get_last_updated(self) -> int:
        return self._last_updated

    @property
    def last_updated(self) -> datetime:
        if self._last_updated is not None:
            return datetime.utcfromtimestamp(self._last_updated)

    @property
    def days_since_last_update(self) -> int:
        if self.last_updated is None:
            return 0
        return abs((self.last_updated - datetime.now()).days)

    def __repr__(self):
        return f"Title: {self.title}\n" \
               f"Id: {self.id}\n" \
               f"Total episodes: {self.total_episodes}\n" \
               f"Episodes watched: {self.episodes_watched}\n" \
               f"Status: {self.status}\n" \
               f"Last updated: {self.last_updated}\n"


class Anilist:
    _username: str = None
    _animelist: Optional[List[AnilistAnime]] = None

    def __init__(self, anilist_token: str):
        self.token = anilist_token

    @property
    def username(self):
        if self._username is None:
            self._username = self.get_username()

        return self._username

    @property
    def anime_list(self):
        if self._animelist is None:
            self._animelist = self.get_animelist()

        return self._animelist

    def get_animelist(self):
        logger.info("Requesting anime list from Anilist")
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
                    updatedAt
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
        all_lists = self.send_graphql_request(query, variables).get('data').get('MediaListCollection').get('lists')
        tracked_lists = [x for x in all_lists if not x.get('isCustomList')]
        animelist = self.animelist_json_to_objects(tracked_lists)
        return animelist

    @staticmethod
    def animelist_json_to_objects(lists_json: List[dict]) -> List[AnilistAnime]:
        animelist = []
        for anilist_list in lists_json:
            anime_in_list = anilist_list.get('entries')
            animelist.extend([AnilistAnime(x) for x in anime_in_list])

        return animelist

    def get_username(self) -> str:
        """ Gets the username of the user that owns the token that is currently associated with this instance of the
        Anilist object.

        :return: The users username.
        """
        logger.info("Requesting username from Anilist")
        query = '''
                query {
                    Viewer {
                        name
                    }
                }
                '''
        response = self.send_graphql_request(query, {})
        return response.get('data').get('Viewer').get('name')

    def send_graphql_request(self, query: str, variables: dict) -> dict:
        headers = {
            'Authorization': 'Bearer ' + self.token,
            'Accept'       : 'application/json',
            'Content-Type' : 'application/json'
        }

        r = requests.post(
            ANILIST_API_BASE_URL,
            headers = headers,
            json = {
                'query'    : query,
                'variables': variables
            })

        # Sleep to avoid rate limiting
        time.sleep(1)

        return json.loads(r.content.decode('utf-8'))

    def get_anime_from_anilistid(self, anilistid: str):
        # Try and get anime from the list
        return next((x for x in self.anime_list if x.id == anilistid), None)

    def reset_anime_list(self):
        self._animelist = None

    def update_anime(self, anilist_id: str, progress: int, status: str) -> bool:
        """ Updates a series on Anilist. This can be used to change the progress or status of a show on Anilist.

        :param anilist_id: The id of the show that needs to be updated.
        :param progress: The current number of watched episodes.
        :param status: The current status be it "completed", "watching" or "plan to watch".
        :return: Whether or not the update was successful.
        """
        logger.info(f"Updating {anilist_id} to {status} episodes watched {progress}")

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
        return self.send_graphql_request(query, variables).get('errors') is None
