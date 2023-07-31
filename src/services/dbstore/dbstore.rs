use async_trait::async_trait;

use crate::services::anime_list_service::anime_list_service::AnimeResult;

use super::sqlite::{Config, Mapping};

#[async_trait]
pub trait DbStore: Sync + Send {
    async fn get_cached_anime_search_result(&self, search_term: &str) -> Option<Vec<AnimeResult>>;
    async fn save_cached_anime_search_result(&self, search_term: &str, data: Vec<AnimeResult>);
    async fn get_cached_anime_result(&self, anime_id: u32) -> Option<AnimeResult>;
    async fn save_cached_anime_result(&self, anime_id: u32, data: AnimeResult);
    async fn clear_anime_search_cache(&self);
    async fn get_config(&self) -> Config;
    async fn get_mappings(&self) -> Result<Vec<Mapping>, sqlx::Error>;
    async fn save_mapping(&self, mapping: &Mapping) -> Result<(), sqlx::Error>;
    async fn get_mapping_for_series(
        &self,
        plex_series_id: &str,
    ) -> Result<Vec<Mapping>, sqlx::Error>;
    async fn get_all_mappings(&self) -> Result<Vec<Mapping>, sqlx::Error>;
}
