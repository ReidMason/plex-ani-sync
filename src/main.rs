use std::env;

use services::{
    anime_list_service::{
        anilist_service::AnilistService,
        anime_list_service::{AnilistWatchStatus, AnimeListService},
    },
    config::config::ConfigService,
    dbstore::sqlite::Sqlite,
    mapping_handler::mapping_handler::{MappingHandler, MappingHandlerInterface},
    plex::{plex_api::PlexInterface, plex_api_service::PlexApi},
    sync_service::sync_handler::{
        get_plex_episodes_for_anime_list_id, plex_series_to_animelist_entry,
    },
};

use crate::{services::dbstore::dbstore::DbStore, utils::get_db_file_location};

mod services;
mod utils;

#[tokio::main]
async fn main() {
    env::set_var("RUST_BACKTRACE", "1");
    utils::init_logger();

    let mut db_store = Sqlite::new(&get_db_file_location()).await;
    db_store.migrate().await;

    let db_config = db_store.get_config().await;

    let config = ConfigService::new(db_config.clone());
    let plex_service = PlexApi::new(config);
    // db_store.clear_anime_search_cache().await;

    let db_store = Sqlite::new(&get_db_file_location()).await;
    let config = ConfigService::new(db_config.clone());
    let anilist_service = AnilistService::new(config, db_store, None);

    let anilist_user = anilist_service
        .get_user()
        .await
        .expect("Failed to get anilist user");
    let anime_list = anilist_service
        .get_list(anilist_user.id)
        .await
        .expect("Failed to get anilist list");

    let db_store = Sqlite::new(&get_db_file_location()).await;
    let mapping_handler = MappingHandler::new(anilist_service, db_store);

    let series = plex_service.get_full_series_data(1).await.unwrap();

    for (i, s) in series.iter().enumerate() {
        println!("Creating new mappings: {}/{}", i, series.len());
        let _ = mapping_handler.create_mapping(s).await;
    }
    println!("Done creating mappings");

    println!("something");

    let mappings = mapping_handler.get_all_relevant_mappings(&series).await;
    let ma = mapping_handler.get_all_mappings().await;

    // We need the anilist id and the number of episodes

    return;
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
            println!(
                "{} needs adding to list\n{:?}\n",
                anime_name, new_anilist_entry
            );
        } else if list_entry.is_some() {
            let list_entry = list_entry.unwrap();
            if &new_anilist_entry != list_entry {
                println!(
                    "{} needs updating in list\n{:?}\n",
                    anime_name, new_anilist_entry
                );
            } else {
                continue;
            }
        }

        // let updated_entry = anilist_service
        //     .update_list_entry(
        //         new_anilist_entry.media_id,
        //         new_anilist_entry.status,
        //         new_anilist_entry.progress,
        //     )
        //     .await;
        //
        // match updated_entry {
        //     Ok(_) => println!("Update successful"),
        //     Err(e) => println!("Failed to update. Error: {}", e),
        // }
    }

    return;

    // ulimit changed with "ulimit -n 256" to go back to default
    // use command "ulimit -n"
}
