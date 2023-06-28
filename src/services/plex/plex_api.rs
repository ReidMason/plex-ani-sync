use std::vec;

use async_trait::async_trait;
use serde::{Deserialize, Serialize};

pub type PlexLibraryResponse = BaseResponse<DirectoryResponse<ResponsePlexLibrary>>;
pub type PlexSeriesResponse = BaseResponse<MetadataResponse<Vec<ResponsePlexSeries>>>;
pub type PlexSeasonResponse = BaseResponse<MetadataResponse<Vec<ResponsePlexSeason>>>;
pub type PlexEpisodesResponse = BaseResponse<MetadataResponse<Vec<ResponsePlexEpisode>>>;

#[async_trait]
pub trait PlexInterface {
    async fn get_libraries(self) -> Result<Vec<ResponsePlexLibrary>, reqwest::Error>;
    async fn get_series(&self, library_id: u8) -> Result<Vec<ResponsePlexSeries>, reqwest::Error>;
    async fn get_episodes(&self, season: &mut PlexSeason) -> Result<(), reqwest::Error>;
    async fn get_seasons2(&self, series: &mut PlexSeries) -> Result<(), reqwest::Error>;
    async fn get_full_series_data(&self, library_id: u8)
        -> Result<Vec<PlexSeries>, reqwest::Error>;
}

pub struct PlexSeries {
    pub rating_key: String,
    pub seasons: Vec<PlexSeason>,
}

impl From<ResponsePlexSeries> for PlexSeries {
    fn from(series: ResponsePlexSeries) -> Self {
        Self {
            rating_key: series.rating_key,
            seasons: vec![],
        }
    }
}

pub struct PlexSeason {
    pub rating_key: String,
    pub episodes: Vec<PlexEpisode>,
}

impl From<ResponsePlexSeason> for PlexSeason {
    fn from(season: ResponsePlexSeason) -> Self {
        Self {
            rating_key: season.rating_key,
            episodes: vec![],
        }
    }
}

#[derive(Clone)]
pub struct PlexEpisode {
    pub rating_key: String,
    pub last_viewed_at: Option<i64>,
}

impl From<ResponsePlexEpisode> for PlexEpisode {
    fn from(episode: ResponsePlexEpisode) -> Self {
        Self {
            rating_key: episode.rating_key,
            last_viewed_at: episode.last_viewed_at,
        }
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ResponsePlexEpisode {
    #[serde(rename = "ratingKey")]
    pub rating_key: String,

    #[serde(rename = "lastViewedAt")]
    pub last_viewed_at: Option<i64>,
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
pub struct ResponsePlexLibrary {
    pub title: String,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct ResponsePlexSeason {
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
pub struct ResponsePlexSeries {
    #[serde(rename = "ratingKey")]
    pub rating_key: String,

    pub title: String,

    #[serde(rename = "lastViewedAt")]
    pub last_viewed_at: Option<u32>,
}

#[derive(Debug, Clone)]
pub struct SeriesWithSeason {
    pub series: ResponsePlexSeries,
    pub seasons: Vec<ResponsePlexSeason>,
}

impl SeriesWithSeason {
    pub fn new(series: ResponsePlexSeries, seasons: Vec<ResponsePlexSeason>) -> Self {
        Self { series, seasons }
    }
}

impl Default for BaseResponse<DirectoryResponse<ResponsePlexLibrary>> {
    fn default() -> Self {
        Self {
            media_container: Default::default(),
        }
    }
}

impl Default for DirectoryResponse<ResponsePlexLibrary> {
    fn default() -> Self {
        Self {
            directory: Default::default(),
        }
    }
}
