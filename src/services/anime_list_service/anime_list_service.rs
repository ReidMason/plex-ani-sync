use async_trait::async_trait;
use serde::{Deserialize, Serialize};

#[async_trait]
pub trait AnimeListService {
    async fn search_anime(&self, search_term: &str) -> Result<Vec<AnimeResult>, anyhow::Error>;
    async fn get_anime(&self, anime_id: String) -> Result<Option<AnimeResult>, anyhow::Error>;
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnimeResult {
    pub id: i64,
    pub format: Option<String>,
    pub episodes: Option<u16>,
    pub synonyms: Vec<String>,
    pub status: String,
    pub end_date: Date,
    pub start_date: Date,
    pub title: Title,
    pub relations: Relations,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Date {
    pub year: Option<i64>,
    pub month: Option<i64>,
    pub day: Option<i64>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Title {
    pub english: Option<String>,
    pub romaji: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Relations {
    pub edges: Vec<Edge>,
    pub nodes: Vec<Node>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Edge {
    pub relation_type: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Node {
    pub id: i64,
    pub format: Option<String>,
    pub end_date: Date,
    pub start_date: Date,
}
