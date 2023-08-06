use log::info;
use services::{
    anime_list_service::{
        anilist_service::AnilistService,
        anime_list_service::{AnilistWatchStatus, AnimeListService},
    },
    dbstore::sqlite::Sqlite,
    mapping_handler::mapping_handler::{MappingHandler, MappingHandlerInterface},
    plex::plex_api_service::PlexApi,
    sync_service::sync_handler::{
        get_plex_episodes_for_anime_list_id, plex_series_to_animelist_entry,
    },
};

use crate::{
    services::{dbstore::dbstore::DbStore, plex::plex_api_service::get_full_series_data},
    utils::get_db_file_location,
};

mod services;
mod utils;

#[tokio::main]
async fn main() {
    utils::init_logger();
    info!("----- Plex Ani Sync started -----");

    info!("Performing database migrations");
    let mut db_store = Sqlite::new(&get_db_file_location()).await;
    db_store.migrate().await;

    info!("Creating Plex service");
    let config = db_store.get_config().await;
    let plex_service = PlexApi::new(config.plex_url, config.plex_token);
    // db_store.clear_anime_search_cache().await;

    info!("Creating Anilist service");
    let anilist_service = AnilistService::new(config.anilist_token.clone(), db_store, None);

    info!("Getting Anilist user");
    let anilist_user = anilist_service
        .get_user()
        .await
        .expect("Failed to get anilist user");

    info!("Getting Anilist list");
    let anime_list = anilist_service
        .get_list(anilist_user.id)
        .await
        .expect("Failed to get anilist list");

    info!("Checking mappings for all series");
    let db_store = Sqlite::new(&get_db_file_location()).await;
    let mapping_handler = MappingHandler::new(anilist_service, db_store.clone());

    let list_id = 1;
    let series = get_full_series_data(&plex_service, list_id).await.unwrap();

    for (i, s) in series.iter().enumerate() {
        info!(
            "Checking mappings for '{}': {}/{}",
            s.title,
            i,
            series.len()
        );
        let _ = mapping_handler.create_mapping(s).await;
    }
    info!("Done checking mappings");

    let mappings = mapping_handler.get_all_relevant_mappings(&series).await;
    let ma = mapping_handler.get_all_mappings().await;

    let anilist_service = AnilistService::new(config.anilist_token.clone(), db_store, None);

    // We need the anilist id and the number of episodes
    for mapping in mappings {
        let list_entry = anime_list
            .iter()
            .find(|x| x.media_id == mapping.anime_list_id);

        let thing = get_plex_episodes_for_anime_list_id(&series, &ma, mapping.anime_list_id);
        let new_anilist_entry = plex_series_to_animelist_entry(thing);

        let update_planning = false;
        if !update_planning && new_anilist_entry.status == AnilistWatchStatus::Planning {
            continue;
        }

        let anime_name = mapping.anime_list_id;
        if list_entry.is_none() {
            info!(
                "{} needs adding to list\n{:?}\n",
                anime_name, new_anilist_entry
            );
        } else if list_entry.is_some() {
            let list_entry = list_entry.unwrap();
            if &new_anilist_entry != list_entry {
                info!(
                    "{} needs updating in list\n{:?}\n",
                    anime_name, new_anilist_entry
                );
            } else {
                continue;
            }
        }

        let updated_entry = anilist_service
            .update_list_entry(
                new_anilist_entry.media_id,
                new_anilist_entry.status,
                new_anilist_entry.progress,
            )
            .await;

        match updated_entry {
            Ok(_) => info!("Update successful"),
            Err(e) => info!("Failed to update. Error: {}", e),
        }
    }

    return;

    // ulimit changed with "ulimit -n 256" to go back to default
    // use command "ulimit -n"
}
