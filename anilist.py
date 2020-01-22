import json
import logging
import time
from dataclasses import dataclass
from pprint import pprint
from typing import Optional, Union

import coloredlogs
import requests

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


@dataclass
class Anilist:
    """ This is an interface for Anilist. All of the requests sent to the Anilist api are send from here and as such it
    obtains data from Anilist such as the username and users list.

    access_token: The access token to use for the Anilist api.
    """
    access_token: str

    class InvalidToken(Exception):
        """ A custom error for when the Anilist token is invalid. """
        pass

    def __post_init__(self) -> None:
        """ Creates the username and user_list properties for this class to save from sending requests every time this
        data is required by the module.

        :return: None
        """
        self.username = self.get_username()
        self.user_list = self.fetch_user_list()

    def get_anime(self, anilist_id: str) -> Optional[dict]:
        """ Gets an anime from the users list with a matching anilist id.

        :param anilist_id: The id to use to search the users list.
        :return: The data for the requested anime or None if the anime isn't in the list.
        """
        return self.user_list.get(anilist_id)

    def send_query(self, query: str, variables: dict) -> Union[dict, list]:
        """ Sends a query request to the Anilist api. This is a wrapper for adding the heading and parsing the response
        when sending requests.

        :param query: The actual query to be sent to Anilist using the GraphQl syntax.
        :param variables: Variables to be used in the query conforming to GraphQl syntax.
        :return: The response from the Anilist api in a python readable format.
        """
        url = 'https://graphql.anilist.co'

        headers = {
            'Authorization': 'Bearer ' + self.access_token,
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

    def update_series(self, anilist_id: str, progress: int, status: str) -> bool:
        """ Updates a series on Anilist. This can be used to change the progress or status of a show on Anilist.

        :param anilist_id: The id of the show that needs to be updated.
        :param progress: The current number of watched episodes.
        :param status: The current status be it "completed", "watching" or "plan to watch".
        :return: Whether or not the update was successful.
        """
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

    def fetch_user_list(self) -> dict:
        """ Gets the users list from Anilist and formats it into a dictionary so that all the shows are accessible by
        their ids as keys.

        This will only use default lists such as 'CURRENT', 'PLANNING', 'COMPLETED', 'DROPPED', 'PAUSED', 'REPEATING'.

        :return: A dictionary containing all the shows on the users list.
        """
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
        all_lists = self.send_query(query, variables).get('data').get('MediaListCollection').get('lists')
        for anilist_list in all_lists:
            # Only look at these lists as there may be many other types of lists
            if anilist_list.get('status') in ['CURRENT', 'PLANNING', 'COMPLETED', 'DROPPED', 'PAUSED', 'REPEATING']:
                for anime in anilist_list.get('entries'):
                    anime_list[str(anime.get('media').get('id'))] = anime

        time.sleep(1)
        return anime_list

    def get_username(self) -> str:
        """ Gets the username of the user that owns the token that is currently associated with this instance of the
        Anilist object.

        :return: The users username.
        """
        query = '''
                    query {
                        Viewer {
                            name
                        }
                    }
                    '''

        return self.send_query(query, {}).get('data').get('Viewer').get('name')
