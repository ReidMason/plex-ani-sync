import json
import time
from typing import List, Optional

import requests
from models.anilist.anime import Anime

import utils
from models.anilist.anilistTokenResponse import AnilistTokenResponse
from services.animeListServices.IAnimeListService import IAnimeListService
from models.animeList.animeList import AnimeList
from models.animeList.animeListAnime import AnimeListAnime
from models.twoWayMapping import TwoWayMapping

logger = utils.create_logger(__name__)

watch_status_mapping: TwoWayMapping = TwoWayMapping()
watch_status_mapping["completed"] = 1
watch_status_mapping["current"] = 2
watch_status_mapping["planning"] = 3
watch_status_mapping["paused"] = 4
watch_status_mapping["dropped"] = 5


class AnilistAuth:
    def __init__(self, authorization_code: str):
        self.authorization_code: str = authorization_code

    def get_token(self) -> Optional[AnilistTokenResponse]:
        payload = {
            'grant_type': 'authorization_code',
            'client_id': '***REMOVED***',
            'client_secret': '***REMOVED***',
            'redirect_uri': '',
            'code': self.authorization_code,
        }

        r = requests.post("https://anilist.co/api/v2/oauth/token", json=payload)
        response = r.json()

        # Check for invalid response
        if response.get('token_type') != "Bearer":
            return None

        return AnilistTokenResponse(response)


class AnilistUser:
    def __init__(self, user_id: str, name: str):
        self.user_id = user_id
        self.name = name


class AnilistService(IAnimeListService):
    def __init__(self, token: str):
        self.base_url = 'https://graphql.anilist.co/'
        self._user_id = None
        self.token = token

    @property
    def user_id(self) -> str:
        # Request user from anilist to get the user_id
        if self._user_id is None:
            self._user_id = self.get_user().user_id

        return self._user_id

    def send_graphql_request(self, query: str, variables: dict) -> Optional[dict]:
        """ Sends a grapghql request to the Anilist api

        :param query: The query of the request
        :param variables: The variables used in the request
        :return: The request response data
        """
        headers = {
            'Authorization': 'Bearer ' + self.token,
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        }

        try:
            r = requests.post(
                self.base_url,
                headers=headers,
                json={
                    'query': query,
                    'variables': variables
                })

            # Sleep to avoid rate limiting
            time.sleep(1)
        except requests.ConnectionError:
            logger.error("Unable to connect to Anilist api")
            return None

        # Connection errors
        if r.status_code in [404, 405]:
            logger.error("Anilist api bad request")
            return None

        # Authentication errors
        if r.status_code in [401, 400]:
            logger.error("Unable to authenticate with Anilist api")
            return None

        return json.loads(r.content.decode('utf-8'))

    def get_anime_list(self) -> AnimeList:
        """ Gets animelist data from Anilist

        :return: AnimeList object of all anime from AniList
        """
        logger.info("Requesting anime list from Anilist")
        query = '''
            query ($userid: Int) {
            MediaListCollection(userId: $userid, type: ANIME) {
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
            'userid': self.user_id
        }
        response = self.send_graphql_request(query, variables)

        all_lists = response.get('data').get('MediaListCollection').get('lists') if response is not None else []

        valid_lists = self.filter_invalid_lists(all_lists)

        # Pull out just the list of anime from each anime list dictionary so we have a list of anime lists
        list_of_anime_lists = [x.get('entries') for x in valid_lists]
        # Then flatten that list of lists into one list of all the anime
        all_anime = [anime for sublist in list_of_anime_lists for anime in sublist]

        return self.convert_anilist_anime_list_to_anime_list_object(all_anime)

    @classmethod
    def filter_invalid_lists(cls, anime_lists: List[dict]) -> List[dict]:
        """ Remove invalid lists from Anilist response

        :param anime_lists: Anime lists from Anilist
        :return: Valid anime lists
        """
        valid_list_names = ["watching", "planning", "completed", "dropped", "paused"]
        return [x for x in anime_lists if x.get('name').lower() in valid_list_names]

    @classmethod
    def convert_anilist_anime_list_to_anime_list_object(cls, raw_anime_list: List[dict]) -> AnimeList:
        """ Converst the anilist anime list into an AnimeList object

        :param raw_anime_list: A list of anime abtained form Anilist
        :return: Converted AnimeList object
        """
        anime_list = AnimeList("Anime List")
        for anime in raw_anime_list:
            media = anime.get('media')
            anime_list.anime_list.append(AnimeListAnime(
                anime_id=media.get('id'),
                title=media.get('title').get('english'),
                romaji_title=media.get('title').get('romaji'),
                watch_status=watch_status_mapping.get(anime.get('status').lower(), 0),
                total_episodes=media.get('episodes'),
                watched_episodes=anime.get('progress'),
            ))

        return anime_list

    def get_user(self) -> AnilistUser:
        """ Gets the user details of the user that owns the token that is currently associated with this instance of the
        Anilist object.

        :return: User information
        """
        logger.info("Requesting username from Anilist")
        query = '''
                query {
                    Viewer {
                        id
                        name
                    }
                }
                '''
        response = self.send_graphql_request(query, {})

        if response is None:
            return AnilistUser("", "")

            # Extract relevant data from the response
        viewer = response.get('data').get('Viewer')
        return AnilistUser(
            user_id=viewer.get('id'),
            name=viewer.get('name')
        )

    def update_anime(self, anime_id: str, watched_episodes: int, status: int) -> bool:
        """ Updates a series on Anilist. This can be used to change the progress or status of a show on Anilist.

        :param anime_id: The id of the show that needs to be updated.
        :param watched_episodes: The current number of watched episodes.
        :param status: The current status number.
        :return: Whether or not the update was successful.
        """
        logger.info(f"Updating {anime_id} to {status} episodes watched {watched_episodes}")

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
            'mediaId': anime_id,
            'status': watch_status_mapping.get(status, "").upper(),
            'progress': watched_episodes
        }

        # If there were no errors so the update was successful
        response = self.send_graphql_request(query, variables)

        if response is None:
            return False

        return response.get('errors') is None

    def get_anime_prequels(self, anime_id: int, anime: Anime = None) -> Anime:
        if anime is None:
            anime = self.get_anime(anime_id)

        if anime is None:
            return None

        logger.debug(f"Getting prequels for {anime.id}")

        # If the anime had a prequel but wasn't a valid prequel we need to go passed this and try the prequel before that
        # This can happen when there is an ova or movie between two seasons
        if anime.prequel is None and anime.supposed_prequel is not None:
            # We need to keep going through the prequels until there aren't any more
            supposed_prequel = self.get_anime(anime.supposed_prequel.id)
            iterations = 0
            while supposed_prequel is not None and supposed_prequel.supposed_prequel is not None and iterations < 5:
                iterations += 1
                if supposed_prequel.supposed_prequel.format == anime.format:
                    anime.prequel = self.get_anime_prequels(supposed_prequel.supposed_prequel.id)
                    break

                supposed_prequel = self.get_anime(supposed_prequel.supposed_prequel.id)

        elif anime.prequel is not None and anime.prequel.title is None:
            anime.prequel = self.get_anime_prequels(anime.prequel.id)

        if anime.prequel is not None:
            anime.prequel.sequel = anime

        return anime

    def search_for_anime(self, anime_name: str, release_year: Optional[int] = None):
        query = '''
            query ($anime_name: String) {
                Page(perPage: 5) {
                    media(search: $anime_name, type: ANIME) {
                        id
                        title {
                            romaji
                            english
                        }
                    }
                }
            }
        '''

        logger.info(f"Searching for anime {anime_name}")

        variables = {'anime_name': anime_name}
        response = self.send_graphql_request(query, variables)

        if response is not None:
            anime_data = response.get('data').get('Media')
            return Anime(anime_data)

        return None

    def get_anime_sequels(self, anime_id: int, anime: Anime = None) -> Anime:
        if anime is None:
            anime = self.get_anime(anime_id)

        if anime is None:
            return None

        logger.debug(f"Getting sequels for {anime.id}")

        # If the anime had a sequel but wasn't a valid sequel we need to go passed this and try the sequel after that
        # This can happen when there is an ova or movie between two seasons
        if anime.sequel is None and anime.supposed_sequel is not None:
            # We need to keep going through the sequels until there aren't any more
            supposed_sequel = self.get_anime(anime.supposed_sequel.id)
            iterations = 0
            while supposed_sequel is not None and supposed_sequel.supposed_sequel is not None and iterations < 5:
                iterations += 1
                if supposed_sequel.supposed_sequel.format == anime.format:
                    anime.sequel = self.get_anime_sequels(supposed_sequel.supposed_sequel.id)
                    break

                supposed_sequel = self.get_anime(supposed_sequel.supposed_sequel.id)

        elif anime.sequel is not None and anime.sequel.title is None:
            anime.sequel = self.get_anime_sequels(anime.sequel.id)

        if anime.sequel is not None:
            anime.sequel.prequel = anime

        return anime

    def get_anime_with_seasons(self, anime_id: int) -> Anime:
        """ Tries to get all seasons on an anime """
        anime = self.get_anime_sequels(anime_id)
        anime = self.get_anime_prequels(anime_id, anime)

        # if anime.prequel is not None and anime.prequel.title is None:
        #     anime.prequel = self.get_anime_prequels(anime.prequel.id)
        #     anime.prequel.sequel = anime

        # if (anime.sequel is not None and anime.sequel.title is None) or anime.supposed_sequel is not None:
        #     anime.sequel = self.get_anime_sequels((anime.sequel or anime.supposed_sequel).id)
        #     anime.sequel.prequel = anime

        return anime

    def get_anime(self, anime_id: int) -> Optional[Anime]:
        query = '''
            query ($id: Int) {
                Media(id: $id, type: ANIME) {
                    id
                    format
                    episodes
                    endDate {
                        year
                        month
                        day
                    }
                    startDate {
                        year
                        month
                        day
                    }
                    title {
                        english
                        romaji
                    }
                    relations {
                        edges {
                            relationType
                        }
                        nodes {
                            id
                            format
                            endDate {
                                year
                                month
                                day
                            }
                            startDate {
                                year
                                month
                                day
                            }
                        }
                    }
                }
            }
            '''

        logger.info(f"Getting anime {anime_id}")

        variables = {'id': anime_id}
        response = self.send_graphql_request(query, variables)

        if response is not None:
            anime_data = response.get('data').get('Media')
            return Anime(anime_data)

        return None
