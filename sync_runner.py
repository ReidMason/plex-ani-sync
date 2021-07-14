import copy
import time
from typing import List, Optional

import utils
from Models.seasonmapping import SeasonMapping
from Models.seriesmapping import SeriesMapping
from Models.user import User
from anilist import Anilist, AnilistAnime
from config import USER
from mapper import Mapper
from plex import Plex, PlexAnime

logger = utils.create_logger(__name__)


class Syncher:
    def __init__(self):
        self.only_allow_completed = False
        self.user: User
        self.plex: Plex
        self.mapper: Mapper
        self.anilist: Anilist
        self.mapping_errors: List[str] = []

    def create_new_plex_mappings(self):
        # Don't include specials in this as they can't be trusted to map properly
        for anime in [x for x in self.plex.all_anime if x.season_number != "0"]:
            has_mapping = self.mapper.has_mapping(anime)
            if not has_mapping:
                logger.info(f"Creating mapping for {anime.display_name}")
                # Get a suggested anilistid for the series
                anilistid_suggestion = self.mapper.get_anilistid_suggestion(anime.tvdbid, anime.season_number)

                # Create a new season mapping
                season_mapping = SeasonMapping()
                season_mapping.plex_season_number = anime.season_number
                season_mapping.anilistid = anilistid_suggestion

                # Add the new mapping
                self.mapper.add_mapping(anime.tvdbid, anime.title, season_mapping)

    def calculate_watch_status(self, plex_anime: PlexAnime, anilist_anime: Optional[AnilistAnime],
                               days_since_last_update: int):
        status_not_planning = anilist_anime is not None and anilist_anime.status != "PLANNING"
        same_plex_watched_episodes = anilist_anime is not None and plex_anime.episodes_watched <= anilist_anime.episodes_watched

        if anilist_anime is not None and plex_anime.episodes_watched >= anilist_anime.total_episodes > 0:
            return "COMPLETED"
        elif status_not_planning and same_plex_watched_episodes and days_since_last_update >= self.user.dropped_days_threshold:
            return "DROPPED"
        elif status_not_planning and same_plex_watched_episodes and days_since_last_update >= self.user.paused_days_threshold:
            return "PAUSED"
        elif plex_anime.episodes_watched == 0:
            return "PLANNING"

        return "CURRENT"

    def combine_plex_anime(self, plex_animes: List[PlexAnime]) -> Optional[PlexAnime]:
        if len(plex_animes) == 0:
            return None

        plex_anime = copy.deepcopy(plex_animes.pop(0))

        # If there was only one anime so there's nothing to combine
        if len(plex_animes) == 0:
            return plex_anime

        for anime in plex_animes:
            plex_anime.episodes.extend(anime.episodes)

        return plex_anime

    def process_season_mapping(self, mapping: SeriesMapping, season_mapping: SeasonMapping):
        # Ignore shows marked to be ignored or with seasons that have wildcards
        if mapping.ignore or season_mapping.ignore or mapping.contains_wildcard_season:
            return

        # The anilist mapping is missing so one needs to be added manually
        if season_mapping.anilistid is None:
            logger.warning(
                f"No mapping for {mapping.name} Season {season_mapping.plex_season_number} tvdbid: ({mapping.tvdbid}) ")
            self.mapping_errors.append(f"No mapping for {mapping.name} Season {season_mapping.plex_season_number}\n"
                                       f"    tvdbid: ({mapping.tvdbid})\n")

            return

        plex_animes = self.plex.find_anime_by_tvdbid(mapping.tvdbid)
        wildcard_present = '*' in season_mapping.all_seasons
        plex_animes = [x for x in plex_animes if x.season_number in season_mapping.all_seasons or wildcard_present]

        plex_anime = self.combine_plex_anime(plex_animes)

        if plex_anime is None:
            return

        anilist_anime = self.anilist.get_anime_from_anilistid(season_mapping.anilistid)

        if season_mapping.has_custom_range:
            # Range starts from the index of the episode number provided
            range_start = plex_anime.find_episode_index(int(season_mapping.episode_start))
            # The slice end is exclusive so we need to add one to the ending index
            range_end = plex_anime.find_episode_index(season_mapping.episode_end) + 1
            if None in (range_start, range_end):
                return

            plex_anime.episodes = plex_anime.episodes[range_start: range_end]

        # Anilist anime wasn't found we just need to add it using the info from plex that we have
        if anilist_anime is None:
            watch_status = self.calculate_watch_status(plex_anime, anilist_anime, season_mapping.days_since_last_update)
            # The extra mapping info will show if there's an episode mapping going on
            extra_mapping_info = ""
            if season_mapping.has_custom_range:
                extra_mapping_info = f"Ep: {season_mapping.episode_start} - {season_mapping.episode_end}"
            print(
                f"Updating {plex_anime.title} Season {plex_anime.season_number} - {extra_mapping_info}\n"
                f"  Watch status: {watch_status}\n"
                f"  Episodes watched: {plex_anime.episodes_watched}\n")

            self.anilist.update_anime(season_mapping.anilistid, plex_anime.episodes_watched, watch_status)
            return

        self.update_anilist(anilist_anime, plex_anime, season_mapping)

    def update_anilist(self, anilist_anime: AnilistAnime, plex_anime: PlexAnime, season_mapping: SeasonMapping):
        # If it's marked as completed leave it alone we don't want to update this
        if anilist_anime.status == "COMPLETED":
            # logger.warning(f"{anilist_anime.title} is completed, skipping show")
            return

        # If the season mapping is none then we need to set it according to the anilist status
        if not season_mapping.last_updated_is_none:
            if anilist_anime.status == "DROPPED":
                season_mapping.unix_last_updated = time.time() - (86400 * self.user.dropped_days_threshold)
            elif anilist_anime.status == "PAUSED":
                season_mapping.unix_last_updated = time.time() - (86400 * self.user.paused_days_threshold)
            else:
                season_mapping.unix_last_updated = anilist_anime.get_last_updated()
            self.mapper.save_mapping()

        if anilist_anime.status != "COMPLETED" and self.only_allow_completed:
            return

        watch_status = self.calculate_watch_status(plex_anime, anilist_anime, season_mapping.days_since_last_update)

        # We want the episodes watched to be less than the total episodes but only if the total episodes are more than zero
        episodes_watched = plex_anime.episodes_watched
        if anilist_anime.total_episodes > 0:
            episodes_watched = min(plex_anime.episodes_watched, anilist_anime.total_episodes)

        episodes_watched_mismatch = episodes_watched != anilist_anime.episodes_watched
        anilist_not_ahead = episodes_watched >= anilist_anime.episodes_watched
        more_episodes_watched = episodes_watched > anilist_anime.episodes_watched
        watch_status_change = watch_status != anilist_anime.status

        # Don't replace dropped shows with paused
        if anilist_anime.status == 'DROPPED' and watch_status == 'PAUSED':
            return

        # Check if an update on anilist is needed
        if (more_episodes_watched or watch_status_change or episodes_watched_mismatch) and anilist_not_ahead:
            print(
                f"Updating {anilist_anime.title} - {season_mapping.plex_season_number} \n"
                f"  Watch status: {anilist_anime.status} -> {watch_status}\n"
                f"  Episodes watched: {anilist_anime.episodes_watched} -> {episodes_watched}\n")

            # Don't change the update time if the watch status is being set to dropped or paused
            if watch_status not in ["DROPPED", "PAUSED"]:
                season_mapping.unix_last_updated = time.time()
                self.mapper.save_mapping()

            self.anilist.update_anime(season_mapping.anilistid, episodes_watched, watch_status)

    def save_mapping_errors(self):
        with open('data/mappingErrors.txt', 'w') as f:
            f.write('\n'.join(self.mapping_errors))

    def do_sync(self):
        for mapping in self.mapper.mapping:
            for season_mapping in mapping.seasons:
                self.process_season_mapping(mapping, season_mapping)

    def start_sync(self):
        self.only_allow_completed = False
        self.user = User(USER)

        self.plex = Plex(self.user)
        self.mapper = Mapper()
        self.anilist = Anilist(self.user.anilist_token)

        self.create_new_plex_mappings()
        self.do_sync()
        self.save_mapping_errors()
        # Reset the list to rescan for completed shows
        logger.info("Re-running the check to catch any shows newly added to the list")
        self.anilist.reset_anime_list()
        self.only_allow_completed = True
        self.do_sync()
