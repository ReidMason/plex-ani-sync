use async_trait::async_trait;
use serde::{Deserialize, Serialize};

use super::anilist_service::SaveMediaListEntry;

#[async_trait]
pub trait AnimeListService: Sync + Send {
    async fn search_anime(&self, search_term: &str) -> Result<Vec<AnimeResult>, anyhow::Error>;
    async fn get_anime(&self, anime_id: u32) -> Result<Option<AnimeResult>, anyhow::Error>;
    async fn find_sequel(
        &self,
        anime_result: AnimeResult,
    ) -> Result<Option<AnimeResult>, anyhow::Error>;
    async fn get_list(&self, user_id: u32) -> Result<Vec<AnimeListEntry>, anyhow::Error>;
    async fn update_list_entry(
        &self,
        media_id: u32,
        status: AnilistWatchStatus,
        progress: u16,
    ) -> Result<SaveMediaListEntry, anyhow::Error>;
}

#[derive(Debug, PartialEq)]
pub struct AnimeListEntry {
    pub media_id: u32,
    pub status: AnilistWatchStatus,
    pub progress: u16,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum AnilistWatchStatus {
    Planning,
    Current,
    Paused,
    Dropped,
    Completed,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum MediaFormat {
    Tv,
    TvShort,
    Movie,
    Special,
    Ova,
    Ona,
    Music,
    Manga,
    Novel,
    OneShot,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum MediaStatus {
    Finished,
    Releasing,
    NotYetReleased,
    Cancelled,
    Hiatus,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum RelationType {
    Adaptation,
    Prequel,
    Sequel,
    Parent,
    SideStory,
    Character,
    Summary,
    Alternative,
    SpinOff,
    Other,
    Source,
    Compilation,
    Contains,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnimeResult {
    pub id: u32,
    pub format: Option<MediaFormat>,
    pub episodes: Option<u16>,
    pub synonyms: Vec<String>,
    pub status: MediaStatus,
    pub end_date: Date,
    pub start_date: Date,
    pub title: Title,
    pub relations: Relations,
}

impl AnimeResult {
    pub fn get_title(&self) -> &str {
        match &self.title.english {
            Some(x) => &x,
            None => &self.title.romaji,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Date {
    pub year: Option<i64>,
    pub month: Option<i64>,
    pub day: Option<i64>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Title {
    pub english: Option<String>,
    pub romaji: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Relations {
    pub edges: Vec<Edge>,
    pub nodes: Vec<Node>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Edge {
    pub relation_type: RelationType,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Node {
    pub id: u32,
    pub format: Option<MediaFormat>,
    pub episodes: Option<u16>,
    pub end_date: Date,
    pub start_date: Date,
}
