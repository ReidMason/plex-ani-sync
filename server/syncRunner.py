from datetime import datetime
from typing import List
from config import Config
from models.animeList.animeList import AnimeList
from models.animeList.animeListAnime import AnimeListAnime
from models.mapping.tvdbToAnilistMapping import TvdbToAnilistMapping
from models.plex.plexAnime import PlexAnime
from services.animeListServices.anilistService import AnilistService
from services.mappingServices.mappingService import MappingService
from services.plexService import PlexService
from fileManager import load_json, save_json
import utils
import copy

logger = utils.create_logger("Syncrunner")


class SyncRunner:
    def __init__(self) -> None:
        self.config = Config()

        self.anilist_service = AnilistService(self.config.ANILIST_TOKEN)
        self.mapping_service = MappingService()

        self.plex_service = PlexService(self.config.PLEX_SERVER_URL, self.config.PLEX_TOKEN)

        # Save and load failed mappings
        self.failed_tvdb_id_mappings = load_json("failed_tvdb_id_mappings.json", [])

    def create_mapping_for_anime_series(self, series: List[PlexAnime]):
        for anime in series:
            # Skip specials seasons
            if anime.season_number == "0" or any([x.get("tvdb_id") == anime.tvdb_id and x.get("season_number") == anime.season_number for x in self.failed_tvdb_id_mappings]):
                continue

            # Try and get the mapping if not create it
            mapping = self.mapping_service.get_mapping_by_tvdb_id(anime.tvdb_id, anime.season_number)

            # If we didn't find one we should check for the wildcard listing
            if len(mapping) == 0:
                mapping = self.mapping_service.get_mapping_by_tvdb_id(anime.tvdb_id, "*")

            # If it's still none we need to try and add a mapping
            if len(mapping) == 0:
                self.mapping_service.find_new_anilist_mapping(anime)
                mapping = self.mapping_service.get_mapping_by_tvdb_id(anime.tvdb_id, anime.season_number)

            # Mapping could not be found or created
            if len(mapping) == 0:
                self.failed_tvdb_id_mappings.append({
                    "title": anime.title,
                    "season_number": anime.season_number,
                    "tvdb_id": anime.tvdb_id
                })
                save_json("failed_tvdb_id_mappings.json", self.failed_tvdb_id_mappings)
                print(f"Falied to find mapping for {anime.display_name}")
                continue

    def get_watch_status(self, plex_anime: PlexAnime, mapping: TvdbToAnilistMapping, anime_list_anime: AnimeListAnime):
        # Don't allow changes to completed series
        if anime_list_anime is not None and anime_list_anime.watch_status == 1:
            return

        # We need the total number of episodes so either get it from the anime list object or request the anime
        if anime_list_anime is None:
            anilist_anime = self.anilist_service.get_anime(mapping.anilist_id)
            if anilist_anime is None:
                logger.error(f'Unable to find anime "{plex_anime.display_name}" - AnilistId: {mapping.anilist_id}')
                return

            total_episodes = anilist_anime.episodes
        else:
            total_episodes = anime_list_anime.total_episodes

        last_viewed_at = plex_anime.last_viewed_at
        if last_viewed_at is not None:
            days_since_last_viewing = utils.ensure_number_is_positive(
                (last_viewed_at - datetime.now()).total_seconds()) / 86400
        else:
            days_since_last_viewing = None

        watch_episodes_unchanged = plex_anime.episodes_watched == anime_list_anime.watched_episodes if anime_list_anime is not None else True

        # Completed
        if plex_anime.episodes_watched > 0 and total_episodes != None and plex_anime.episodes_watched >= total_episodes:
            return 1

        # Dropped
        # Needs to be updated depending on how long it's been since the last update and if the current status is Paused or Current
        if watch_episodes_unchanged and plex_anime.episodes_watched > 0 and days_since_last_viewing is not None and days_since_last_viewing > self.config.DAYS_UNTIL_DROPPED:
            return 5

        # Paused
        # Needs to be updated depending on how long it's been since the last update and if the current status is Current
        if watch_episodes_unchanged and plex_anime.episodes_watched > 0 and days_since_last_viewing is not None and days_since_last_viewing > self.config.DAYS_UNTIL_PAUSED:
            return 4

        # Current
        if plex_anime.episodes_watched > 0 and (total_episodes is None or plex_anime.episodes_watched < total_episodes):
            return 2

        # Planning
        if self.config.MARK_UNWATCHED_EPISODES_AS_PLANNING and plex_anime.episodes_watched == 0:
            return 3

    def process_wildcard_mapping(self, series: List[PlexAnime], anime_list: AnimeList):
        # If there's only one season select the first season otherwise pick the second
        # This is so we don't select the specials season
        if series[0].season_number == "0" and len(series) > 1:
            imutable_plex_anime = series[1]
        else:
            imutable_plex_anime = series[0]

        wildcard_mappings: List[TvdbToAnilistMapping] = self.mapping_service.get_mapping_by_tvdb_id(
            imutable_plex_anime.tvdb_id, "*")

        # Mapping has a wildcard mapping
        if len(wildcard_mappings) == 0:
            return False

        # We need to combine all episodes into one list
        all_episodes = []
        for s in series:
            all_episodes.extend(s.episodes if s.season_number != "0" else [])

        for wildcard_mapping in wildcard_mappings:
            # The mapping is ignored so we can skip the show
            if wildcard_mapping.ignored:
                return False

            logger.info(f'Updating: "{imutable_plex_anime.display_name}"')

            # We can skip if no episodes have been watched and we aren't setting shows as planning
            if not self.config.MARK_UNWATCHED_EPISODES_AS_PLANNING and imutable_plex_anime.episodes_watched == 0:
                return

            list_anime = anime_list.get_anime(wildcard_mapping.anilist_id)

            plex_anime = copy.deepcopy(imutable_plex_anime)

            plex_anime.set_cached_episodes(all_episodes)

            # If the mapping only includes an episode range we need to remove the non-tracked episodes
            if wildcard_mapping.season_length is not None:
                start_index = wildcard_mapping.episode_start - 1
                end_index = min(wildcard_mapping.episode_start +
                                wildcard_mapping.season_length, len(plex_anime.episodes))
                plex_anime.set_cached_episodes(plex_anime.episodes[start_index:end_index])

            # plex_anime._episodes_watched = sum([x.episodes_watched for x in series if x.season_number != "0"])
            # plex_anime._last_viewed_at = max([x.last_viewed_at for x in series if x.season_number != "0" and x.last_viewed_at is not None])
            watch_status = self.get_watch_status(plex_anime, wildcard_mapping, list_anime)
            # If we don't have a watch status we can skip this one
            if watch_status is None:
                return

            # The anime isn't on the list so we need to add it
            if list_anime is None:
                self.anilist_service.update_anime(wildcard_mapping.anilist_id,
                                                  plex_anime.episodes_watched, watch_status, plex_anime.title)
            elif list_anime.watch_status != 1 and (list_anime.watch_status != watch_status or list_anime.watched_episodes != plex_anime.episodes_watched):
                self.anilist_service.update_anime(wildcard_mapping.anilist_id,
                                                  plex_anime.episodes_watched, watch_status, plex_anime.title)

    def process_series(self, series: List[PlexAnime], anime_list: AnimeList):
        self.process_wildcard_mapping(series, anime_list)

        for immutable_plex_anime in series:
            # We keep track of wildcard mappings to suppress "no mappings" logs
            wildcard_mappings: List[TvdbToAnilistMapping] = self.mapping_service.get_mapping_by_tvdb_id(
                series[0].tvdb_id, "*")

            mappings: List[TvdbToAnilistMapping] = self.mapping_service.get_mapping_by_tvdb_id(
                series[0].tvdb_id, immutable_plex_anime.season_number)

            # We don't want to print mapping errors for "specials" seasons because they are unlikely to be mapped
            if len(mappings) == 0 and str(immutable_plex_anime.season_number) != "0":
                # We only want to log if there aren't any wildcard mappings
                if (len(wildcard_mappings) == 0):
                    logger.info(f'No mappings for "{immutable_plex_anime.display_name}"')
                continue

            for mapping in mappings:
                # The mapping is ignored so we can skip the show
                if mapping.ignored:
                    continue

                logger.info(f'Updating: "{immutable_plex_anime.display_name}"')

                plex_anime = copy.deepcopy(immutable_plex_anime)
                # If the mapping only includes an episode range we need to remove the non-tracked episodes
                if mapping.season_length is not None:
                    start_index = mapping.episode_start - 1
                    end_index = min(mapping.episode_start + mapping.season_length, len(plex_anime.episodes))
                    plex_anime.set_cached_episodes(plex_anime.episodes[start_index:end_index])

                # We can skip if no episodes have been watched and we aren't setting shows as planning
                if not self.config.MARK_UNWATCHED_EPISODES_AS_PLANNING and plex_anime.episodes_watched == 0:
                    continue

                list_anime = anime_list.get_anime(mapping.anilist_id)
                watch_status = self.get_watch_status(plex_anime, mapping, list_anime)
                # If we don't have a watch status we can skip this one
                if watch_status is None:
                    continue

                # The anime isn't on the list so we need to add it
                if list_anime is None:
                    self.anilist_service.update_anime(mapping.anilist_id, plex_anime.episodes_watched,
                                                      watch_status, plex_anime.title)
                elif list_anime.watch_status != 1 and (list_anime.watch_status != watch_status or list_anime.watched_episodes != plex_anime.episodes_watched):
                    self.anilist_service.update_anime(mapping.anilist_id, plex_anime.episodes_watched,
                                                      watch_status, plex_anime.title)

    def run(self):
        self.plex_service.authenticate()

        anime_list = self.anilist_service.get_anime_list()
        all_anime = self.plex_service.get_all_anime()
        for series in all_anime:
            self.create_mapping_for_anime_series(series)
            self.process_series(series, anime_list)


if __name__ == "__main__":
    sync_runner = SyncRunner()
    try:
        sync_runner.run()
    except Exception as e:
        print(e)
