from typing import List

from config import PLEX_SERVER_URL, ANILIST_TOKEN
from models.plex.plexAnime import PlexAnime
from services.animeListServices.anilistService import AnilistService
from services.mappingServices.mappingService import MappingService
from services.plexService import PlexService
from fileManager import load_json, save_json

anilist_service = AnilistService(ANILIST_TOKEN)

mapping_service = MappingService()

plex_service = PlexService(PLEX_SERVER_URL)
plex_service.authenticate()
all_anime = plex_service.get_all_anime()

# Save and load failed mappings
failed_tvdb_id_mappings = load_json("failed_tvdb_id_mappings.json", [])


def process_anime_series(series: List[PlexAnime]):
    for anime in series:
        # Skip specials seasons
        if anime.season_number == "0" or any([x.get("tvdb_id") == anime.tvdb_id and x.get("season_number") == anime.season_number for x in failed_tvdb_id_mappings]):
            continue

        mapping = mapping_service.find_mapping_by_tvdb_id(anime.tvdb_id, anime.season_number)
        if mapping is None:
            mapping_service.find_new_anilist_mapping(anime)
            mapping = mapping_service.find_mapping_by_tvdb_id(anime.tvdb_id, anime.season_number)

        # Mapping could not be found
        if mapping is None:
            failed_tvdb_id_mappings.append({
                "title": anime.title,
                "season_number": anime.season_number,
                "tvdb_id": anime.tvdb_id
            })
            save_json("failed_tvdb_id_mappings.json", failed_tvdb_id_mappings)
            print(f"Falied to find mapping for {anime.display_name}")
            return


for series in all_anime:
    process_anime_series(series)
