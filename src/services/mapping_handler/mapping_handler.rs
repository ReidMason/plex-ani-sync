use async_trait::async_trait;
use log::warn;

use crate::services::anime_list_service::anime_list_service::{AnimeListService, AnimeResult};
use crate::services::plex_api::plex_api::{PlexSeason, SeriesWithSeason};

#[async_trait]
pub trait MappingHandlerInterface {
    async fn find_mapping(&self, series: SeriesWithSeason) -> Result<Vec<Mapping>, anyhow::Error>;
}

pub struct Mapping {
    id: u32,

    plex_id: String,
    plex_episode_start: u16,
    pub season_length: u16,

    anime_list_id: String,
    episode_start: u16,

    enabled: bool,
    ignored: bool,
}

pub struct MappingHandler<T>
where
    T: AnimeListService,
{
    anime_list_service: T,
}

impl<T> MappingHandler<T>
where
    T: AnimeListService,
{
    pub fn new(anime_list_service: T) -> Self {
        return Self { anime_list_service };
    }
}

fn find_match(results: Vec<AnimeResult>, target: &PlexSeason) -> Option<AnimeResult> {
    for result in results {
        if result.episodes == Some(target.episodes) {
            return Some(result);
        }
    }
    None
}

#[async_trait]
impl<T> MappingHandlerInterface for MappingHandler<T>
where
    T: AnimeListService + Sync + Send,
{
    async fn find_mapping(&self, series: SeriesWithSeason) -> Result<Vec<Mapping>, anyhow::Error> {
        let mut mappings: Vec<Mapping> = vec![];
        for season in series.seasons {
            let results = self
                .anime_list_service
                .search_anime(&season.parent_title)
                .await?;

            if results.is_empty() {
                warn!(
                    "Found no results for {} season: {}",
                    season.parent_title, season.index
                );
                continue;
            }

            let result = match find_match(results, &season) {
                Some(result) => result,
                None => {
                    warn!(
                        "Didn't find a good match for {} season: {}",
                        season.parent_title, season.index
                    );
                    continue;
                }
            };

            let mapping = Mapping {
                id: 1,
                plex_id: season.rating_key,
                plex_episode_start: 0,
                season_length: season.episodes,
                anime_list_id: result.id.to_string(),
                episode_start: 0,
                enabled: true,
                ignored: false,
            };
            mappings.push(mapping);
        }
        return Ok(mappings);
    }
}

#[cfg(test)]
mod tests {
    use crate::{
        services::{
            anime_list_service::anilist_service::AnilistService,
            config::config::ConfigService,
            dbstore::{dbstore::DbStore, sqlite::Sqlite},
            plex_api::plex_api::{PlexSeason, PlexSeries},
        },
        utils::{get_db_file_location, init_logger},
    };

    use super::*;

    #[tokio::test]
    async fn test_one_to_one_mapping() {
        init_logger();

        let mut db_store = Sqlite::new(&get_db_file_location()).await;
        db_store.migrate().await;

        let config = db_store.get_config().await;
        let config_service = ConfigService::new(config);

        let list_service = AnilistService::new(config_service, db_store, None);

        let mapper = MappingHandler::new(list_service);

        let series = SeriesWithSeason {
            series: PlexSeries {
                title: "Mysterious Girlfriend X".to_string(),
                rating_key: "12345".to_string(),
                last_viewed_at: Some(0),
            },
            seasons: vec![PlexSeason {
                rating_key: "12345".to_string(),
                parent_title: "Mysterious Girlfriend X".to_string(),
                index: 1,
                episodes: 13,
                parent_year: Some(2012),
                watched_episodes: 0,
                last_viewed_at: Some(0),
            }],
        };

        let result = mapper
            .find_mapping(series)
            .await
            .expect("Faied to get result for one to one mapping");

        assert_eq!(1, result.len());
        assert_eq!("12467".to_string(), result[0].anime_list_id)
    }
}
