use async_trait::async_trait;
use serde::{Deserialize, Serialize};

pub type PlexLibraryResponse = BaseResponse<DirectoryResponse<PlexLibrary>>;
pub type PlexSeriesResponse = BaseResponse<MetadataResponse<Vec<PlexSeries>>>;
pub type PlexSeasonResponse = BaseResponse<MetadataResponse<Vec<PlexSeason>>>;

#[async_trait]
pub trait PlexInterface {
    async fn get_libraries(self) -> Result<Vec<PlexLibrary>, reqwest::Error>;
    async fn get_series(&self, library_id: u8) -> Result<Vec<PlexSeries>, reqwest::Error>;
    async fn get_seasons(&self, rating_key: &str) -> Result<Vec<PlexSeason>, reqwest::Error>;
    async fn get_all_seasons(
        &self,
        library_id: u8,
    ) -> Result<Vec<SeriesWithSeason>, reqwest::Error>;
    async fn popualte_seasons(
        &self,
        series: PlexSeries,
    ) -> Result<SeriesWithSeason, reqwest::Error>;
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
