use std::vec;

use async_trait::async_trait;
use serde::{Deserialize, Serialize};

pub type PlexLibraryResponse = BaseResponse<DirectoryResponse<PlexLibrary>>;
pub type PlexSeriesResponse = BaseResponse<MetadataResponse<Vec<PlexSeries>>>;
pub type PlexSeasonResponse = BaseResponse<MetadataResponse<Vec<PlexSeason>>>;
pub type PlexEpisodesResponse = BaseResponse<MetadataResponse<Vec<PlexEpisodeResponse>>>;

#[async_trait]
pub trait PlexInterface {
    async fn get_libraries(self) -> Result<Vec<PlexLibrary>, reqwest::Error>;
    async fn get_series(&self, library_id: u8) -> Result<Vec<PlexSeries>, reqwest::Error>;
    async fn get_seasons(&self, rating_key: &str) -> Result<Vec<PlexSeason>, reqwest::Error>;
    async fn get_all_seasons(
        &self,
        library_id: u8,
    ) -> Result<Vec<SeriesWithSeason>, reqwest::Error>;
    async fn populate_seasons(
        &self,
        series: PlexSeries,
    ) -> Result<SeriesWithSeason, reqwest::Error>;
    async fn get_episodes(&self, season: &mut PlexSeason2) -> Result<(), reqwest::Error>;

    async fn get_seasons2(&self, series: &mut PlexSeries2) -> Result<(), reqwest::Error>;
    async fn get_full_series_data(
        &self,
        library_id: u8,
    ) -> Result<Vec<PlexSeries2>, reqwest::Error>;
}

pub struct PlexSeries2 {
    pub rating_key: String,
    pub seasons: Vec<PlexSeason2>,
}

impl From<PlexSeries> for PlexSeries2 {
    fn from(series: PlexSeries) -> Self {
        Self {
            rating_key: series.rating_key,
            seasons: vec![],
        }
    }
}

pub struct PlexSeason2 {
    pub rating_key: String,
    pub episodes: Vec<PlexEpisode>,
}

impl From<PlexSeason> for PlexSeason2 {
    fn from(season: PlexSeason) -> Self {
        Self {
            rating_key: season.rating_key,
            episodes: vec![],
        }
    }
}

pub struct PlexEpisode {
    pub rating_key: String,
}

impl From<PlexEpisodeResponse> for PlexEpisode {
    fn from(episode: PlexEpisodeResponse) -> Self {
        Self {
            rating_key: episode.rating_key,
        }
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct PlexEpisodeResponse {
    #[serde(rename = "ratingKey")]
    pub rating_key: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct BaseResponse<T> {
    #[serde(rename = "MediaContainer")]
    pub media_container: T,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MetadataResponse<T> {
    #[serde(rename = "Metadata")]
    pub metadata: T,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct DirectoryResponse<T> {
    #[serde(rename = "Directory")]
    pub directory: Vec<T>,
}

#[derive(Clone, Debug, Deserialize, Serialize, Default)]
pub struct PlexLibrary {
    pub title: String,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct PlexSeason {
    #[serde(rename = "ratingKey")]
    pub rating_key: String,

    pub index: u8,

    #[serde(rename = "parentTitle")]
    pub parent_title: String,

    #[serde(rename = "parentYear")]
    pub parent_year: Option<u16>,

    #[serde(rename = "viewedLeafCount")]
    pub watched_episodes: u8,

    #[serde(rename = "leafCount")]
    pub episodes: u16,

    #[serde(rename = "lastViewedAt")]
    pub last_viewed_at: Option<u32>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct PlexSeries {
    #[serde(rename = "ratingKey")]
    pub rating_key: String,

    pub title: String,

    #[serde(rename = "lastViewedAt")]
    pub last_viewed_at: Option<u32>,
}

#[derive(Debug, Clone)]
pub struct SeriesWithSeason {
    pub series: PlexSeries,
    pub seasons: Vec<PlexSeason>,
}

impl SeriesWithSeason {
    pub fn new(series: PlexSeries, seasons: Vec<PlexSeason>) -> Self {
        Self { series, seasons }
    }
}

impl Default for BaseResponse<DirectoryResponse<PlexLibrary>> {
    fn default() -> Self {
        Self {
            media_container: Default::default(),
        }
    }
}

impl Default for DirectoryResponse<PlexLibrary> {
    fn default() -> Self {
        Self {
            directory: Default::default(),
        }
    }
}
