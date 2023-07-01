use async_trait::async_trait;
use serde::{Deserialize, Serialize};

#[async_trait]
pub trait AnimeListService {
    async fn search_anime(&self, search_term: &str) -> Result<Vec<AnimeResult>, anyhow::Error>;
    async fn get_anime(&self, anime_id: &str) -> Result<Option<AnimeResult>, anyhow::Error>;
    async fn find_sequel(
        &self,
        anime_result: AnimeResult,
    ) -> Result<Option<AnimeResult>, anyhow::Error>;
    async fn get_list(&self) -> Result<AnimeList, anyhow::Error>;
}

pub struct AnimeList {}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum MediaFormat {
    TV,
    #[serde(rename = "TV_SHORT")]
    TvShort,
    #[serde(rename = "MOVIE")]
    Movie,
    #[serde(rename = "SPECIAL")]
    Special,
    OVA,
    ONA,
    #[serde(rename = "MUSIC")]
    Music,
    #[serde(rename = "MANGA")]
    Manga,
    #[serde(rename = "NOVEL")]
    Novel,
    #[serde(rename = "ONE_SHOT")]
    OneShot,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum MediaStatus {
    #[serde(rename = "FINISHED")]
    Finished,
    #[serde(rename = "RELEASING")]
    Releasing,
    #[serde(rename = "NOT_YET_RELEASED")]
    NotYetReleased,
    #[serde(rename = "CANCELLED")]
    Cancelled,
    #[serde(rename = "HIATUS")]
    Hiatus,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum RelationType {
    #[serde(rename = "ADAPTATION")]
    Adaptation,
    #[serde(rename = "PREQUEL")]
    Prequel,
    #[serde(rename = "SEQUEL")]
    Sequel,
    #[serde(rename = "PARENT")]
    Parent,
    #[serde(rename = "SIDE_STORY")]
    SideStory,
    #[serde(rename = "CHARACTER")]
    Character,
    #[serde(rename = "SUMMARY")]
    Summary,
    #[serde(rename = "ALTERNATIVE")]
    Alternative,
    #[serde(rename = "SPIN_OFF")]
    SpinOff,
    #[serde(rename = "OTHER")]
    Other,
    #[serde(rename = "SOURCE")]
    Source,
    #[serde(rename = "COMPILATION")]
    Compilation,
    #[serde(rename = "CONTAINS")]
    Contains,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnimeResult {
    pub id: i64,
    pub format: Option<MediaFormat>,
    pub episodes: Option<u16>,
    pub synonyms: Vec<String>,
    pub status: MediaStatus,
    pub end_date: Date,
    pub start_date: Date,
    pub title: Title,
    pub relations: Relations,
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
    pub id: i64,
    pub format: Option<MediaFormat>,
    pub episodes: Option<u16>,
    pub end_date: Date,
    pub start_date: Date,
}
