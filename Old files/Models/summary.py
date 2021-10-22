from Models.seasonmapping import SeasonMapping
from Models.seriesmapping import SeriesMapping
from anilist import AnilistAnime
from plex import PlexAnime
import requests
from config import EMAIL_NOTIFIER_URL, EMAIL_NOTIFIER_BEARER_TOKEN


class Summary:
    def __init__(self):
        self.updates = []
        self.mapping_errors = []

    def add_to_updates(self, plex_anime: PlexAnime, watch_status: str, episodes_watched: int,
                       anilist_anime: AnilistAnime = None,
                       extra_mapping_info: str = None):
        extra_mapping_info = f"({extra_mapping_info})" if extra_mapping_info is not None else ''
        self.updates.append({
            "title"           : f"{plex_anime.title} - Season {plex_anime.season_number} {extra_mapping_info}",
            "watch_status"    : f"{anilist_anime.status} -> {watch_status}" if anilist_anime is not None else watch_status,
            "episodes_watched": f"{anilist_anime.episodes_watched} -> {episodes_watched}" if anilist_anime is not None else episodes_watched,
        })

    def add_to_mapping_errors(self, mapping: SeriesMapping, season_mapping: SeasonMapping):
        self.mapping_errors.append({
            "title" : f"{mapping.name} - Season {season_mapping.plex_season_number}",
            "tvdbid": mapping.tvdbid
        })

    def serialize(self) -> dict:
        return {
            "updates"       : self.updates,
            "mapping_errors": self.mapping_errors
        }

    def send_notification_email(self):
        if (len(self.updates) + len(self.mapping_errors) == 0):
            return

        headers = {'Authorization': f'Bearer {EMAIL_NOTIFIER_BEARER_TOKEN}',
                   'Content-Type' : 'application/json'}

        requests.post(EMAIL_NOTIFIER_URL, json = self.serialize(), headers = headers, verify = False)


if __name__ == '__main__':
    summary = Summary()
    summary.send_notification_email()
