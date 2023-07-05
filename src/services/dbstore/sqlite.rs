use std::str::FromStr;

use super::dbstore::DbStore;
use crate::services::anime_list_service::anime_list_service::AnimeResult;

use async_trait::async_trait;
use log::{error, info};
use serde::{Deserialize, Serialize};
use sqlx::{
    sqlite::{SqliteConnectOptions, SqlitePoolOptions},
    ConnectOptions, FromRow, Pool,
};

#[derive(Clone)]
pub struct Sqlite {
    pool: Pool<sqlx::Sqlite>,
}

impl Sqlite {
    pub async fn new(db_location: &str) -> Self {
        // let conn = SqliteConnection::connect(db_location.as_str())
        //     .await
        //     .unwrap();

        let mut connect_options = SqliteConnectOptions::from_str(db_location)
            .expect("Unable to create database connection options");
        connect_options.log_statements(log::LevelFilter::Debug);

        let pool = SqlitePoolOptions::new()
            .max_connections(5)
            .connect_with(connect_options)
            .await
            .expect("Unable to connect to database");

        Self { pool }
    }

    pub async fn migrate(&mut self) {
        sqlx::migrate!("./migrations")
            .run(&self.pool)
            .await
            .expect("Failed to perform migration");
    }
}

#[async_trait]
impl DbStore for Sqlite {
    async fn get_config(&self) -> Config {
        let result = sqlx::query_as::<_, Config>("SELECT * FROM config LIMIT 1")
            .fetch_one(&self.pool)
            .await
            .expect("Failed to load config");

        return result;
    }

    async fn get_cached_anime_search_result(&self, search_term: &str) -> Option<Vec<AnimeResult>> {
        let search_result = sqlx::query_as::<_, CachedAnimeResult>(
            "SELECT * FROM anime_search_cache WHERE search_term = ?",
        )
        .bind(search_term)
        .fetch_optional(&self.pool)
        .await
        .unwrap_or(None);

        let data = match search_result {
            Some(x) => x.data,
            None => {
                return None;
            }
        };

        let anime_data = serde_json::from_str(data.as_str());
        match anime_data {
            Ok(x) => return Some(x),
            Err(e) => {
                error!(
                    "Failed to parse cached anime search result for: {}. {}",
                    search_term, e
                );
                return None;
            }
        };
    }

    async fn save_cached_anime_search_result(&self, search_term: &str, data: Vec<AnimeResult>) {
        let string_data = serde_json::to_string(&data).unwrap();

        let result = sqlx::query(
            "INSERT OR REPLACE INTO anime_search_cache (search_term, data) VALUES (?, ?)",
        )
        .bind(search_term)
        .bind(string_data)
        .execute(&self.pool)
        .await;

        match result {
            Ok(_) => {}
            Err(e) => {
                error!("Failed to save anime search result. {}", e);
            }
        };
    }

    async fn get_cached_anime_result(&self, anime_id: &str) -> Option<AnimeResult> {
        let search_result =
            sqlx::query_as::<_, CachedAnime>("SELECT * FROM anime_cache WHERE anime_id = ?")
                .bind(anime_id)
                .fetch_optional(&self.pool)
                .await
                .unwrap_or(None);

        if !search_result.is_some() {
            return None;
        }

        let data = search_result
            .expect("Errored getting cached anime data")
            .data;

        let anime_data: AnimeResult =
            serde_json::from_str(data.as_str()).expect("Failed to deserialize cached data");

        return Some(anime_data);
    }

    async fn save_cached_anime_result(&self, anime_id: &str, data: AnimeResult) {
        let string_data = serde_json::to_string(&data).unwrap();

        sqlx::query_as::<_, CachedAnime>(
            "INSERT INTO anime_cache (anime_id, data) VALUES (?, ?) RETURNING *",
        )
        .bind(anime_id)
        .bind(string_data)
        .fetch_one(&self.pool)
        .await
        .expect("Failed to insert cache data into database");
    }

    async fn clear_anime_search_cache(&self) {
        info!("Clearing anime search cache");
        let _ = sqlx::query("DELETE FROM anime_search_cache;")
            .execute(&self.pool)
            .await
            .expect("Failed to clear anime cache");
    }

    async fn get_mappings(&self) -> Result<Vec<Mapping>, sqlx::Error> {
        sqlx::query_as::<_, Mapping>("SELECT * FROM mapping")
            .fetch_all(&self.pool)
            .await
    }

    async fn get_mapping_for_series(
        &self,
        plex_series_id: &str,
    ) -> Result<Vec<Mapping>, sqlx::Error> {
        sqlx::query_as::<_, Mapping>("SELECT * FROM mapping WHERE plex_series_id = ?")
            .bind(plex_series_id)
            .fetch_all(&self.pool)
            .await
    }

    async fn get_all_mappings(&self) -> Result<Vec<Mapping>, sqlx::Error> {
        sqlx::query_as::<_, Mapping>("SELECT * FROM mapping")
            .fetch_all(&__self.pool)
            .await
    }

    async fn save_mapping(&self, mapping: &Mapping) -> Result<(), sqlx::Error> {
        sqlx::query("INSERT INTO mapping (list_provider_id, plex_id, plex_series_id, plex_episode_start, season_length, anime_list_id, episode_start, enabled, ignored) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
            .bind(mapping.list_provider_id)
            .bind(&mapping.plex_id)
            .bind(&mapping.plex_series_id)
            .bind(mapping.plex_episode_start)
            .bind(mapping.season_length)
            .bind(&mapping.anime_list_id)
            .bind(mapping.episode_start)
            .bind(mapping.enabled)
            .bind(mapping.ignored)
            .fetch_one(&self.pool)
            .await?;
        Ok(())
    }
}

#[derive(FromRow, Clone, Serialize, Deserialize)]
pub struct CachedAnimeResult {
    id: u32,
    search_term: String,
    data: String,
}

#[derive(FromRow, Clone, Serialize, Deserialize)]
pub struct CachedAnime {
    id: u32,
    anime_id: String,
    data: String,
}

#[derive(FromRow, Clone, Serialize, Deserialize)]
pub struct Config {
    id: u32,
    pub plex_url: String,
    pub plex_token: String,
    pub anilist_token: String,
}

#[derive(FromRow, Serialize, Deserialize)]
pub struct ListProvider {
    pub id: u32,
    pub name: String,
}

#[derive(FromRow, Clone, Serialize, Deserialize, Debug)]
pub struct Mapping {
    pub id: u32,
    pub list_provider_id: u32,
    pub plex_id: String,
    pub plex_series_id: String,
    pub plex_episode_start: u32,
    pub season_length: u32,
    pub anime_list_id: String,
    pub episode_start: u32,
    pub enabled: bool,
    pub ignored: bool,
}

#[cfg(test)]
mod tests {
    use crate::{
        services::anime_list_service::{
            anime_list_service::{AnimeListService, Date, MediaStatus, Relations, Title},
            mock_anime_list_service::MockAnimeListService,
        },
        utils::init_logger,
    };

    use super::*;

    #[tokio::test]
    async fn test_cache_anime_list_response() {
        init_logger();

        let mut dbstore = Sqlite::new("sqlite::memory:").await;
        dbstore.migrate().await;

        let anime_list_service = MockAnimeListService {};
        let search_term = String::from("Sword art online");
        let result = anime_list_service
            .search_anime(&search_term)
            .await
            .expect("Failed to perform mock anime search");

        dbstore
            .save_cached_anime_search_result(&search_term, result)
            .await;

        let cached_data = dbstore
            .get_cached_anime_search_result(&search_term)
            .await
            .expect("Failed to get cached data");

        assert_eq!(cached_data.len(), 5);
    }

    #[tokio::test]
    async fn test_override_anime_cache() {
        init_logger();

        let mut dbstore = Sqlite::new("sqlite::memory:").await;
        dbstore.migrate().await;

        let search_term = "Sword art online";
        let result = vec![AnimeResult {
            id: 1,
            format: None,
            episodes: None,
            synonyms: vec![],
            status: MediaStatus::Finished,
            end_date: Date {
                year: None,
                month: None,
                day: None,
            },
            start_date: Date {
                year: None,
                month: None,
                day: None,
            },
            title: Title {
                english: None,
                romaji: "Result 1".to_string(),
            },
            relations: Relations {
                edges: vec![],
                nodes: vec![],
            },
        }];

        let mut result2 = result.clone();
        result2[0].id = 2;

        dbstore
            .save_cached_anime_search_result(&search_term, result)
            .await;

        let cached_data = dbstore
            .get_cached_anime_search_result(&search_term)
            .await
            .expect("Failed to get cached data");

        assert_eq!(cached_data[0].id, 1);

        dbstore
            .save_cached_anime_search_result(&search_term, result2)
            .await;

        let cached_data = dbstore
            .get_cached_anime_search_result(&search_term)
            .await
            .expect("Failed to get cached data");

        assert_eq!(cached_data[0].id, 2);
    }
}
