use async_trait::async_trait;
use futures::{future::join_all, stream::FuturesUnordered};
use log::{error, info};
use reqwest::header::{self, HeaderMap, HeaderValue, ACCEPT, AUTHORIZATION, CONTENT_TYPE};
use serde::de::DeserializeOwned;
use url::Url;

use crate::services::{
    config::config::ConfigInterface,
    plex_api::plex_api::{PlexLibraryResponse, PlexSeasonResponse, PlexSeriesResponse},
};

use super::plex_api::{PlexInterface, PlexLibrary, PlexSeason, PlexSeries, SeriesWithSeason};

pub struct PlexApi<T>
where
    T: ConfigInterface,
{
    config: T,
    http_client: reqwest::Client,
    headers: header::HeaderMap,
}

impl<T> PlexApi<T>
where
    T: ConfigInterface,
{
    pub fn new(config: T) -> Self {
        let mut header_map = header::HeaderMap::new();
        header_map.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
        header_map.insert(ACCEPT, HeaderValue::from_static("application/json"));

        Self {
            config,
            http_client: reqwest::Client::new(),
            headers: header_map,
        }
    }

    fn get_headers(&self) -> HeaderMap {
        let auth_value = format!("Bearer {}", self.config.get_plex_token());
        let mut headers = self.headers.clone();
        headers.insert(
            AUTHORIZATION,
            HeaderValue::from_str(auth_value.as_str()).expect("Failed to parse anilist token"),
        );
        headers
    }

    async fn make_request<R>(&self, path: &str) -> Result<R, reqwest::Error>
    where
        R: DeserializeOwned,
    {
        let url = self.build_request_url(path);
        self.http_client
            .get(&url)
            .headers(self.get_headers())
            .send()
            .await?
            .json::<R>()
            .await
    }

    fn build_request_url(&self, path: &str) -> String {
        let base_url = self.config.get_plex_base_url();
        let token = self.config.get_plex_token();

        let url_builder = Url::parse(base_url).expect("Failed to parse Plex base url");
        let mut url_builder = url_builder
            .join(path)
            .expect("Failed to join Plex sections url path to base url");
        let query = format!("X-Plex-Token={}", token);
        url_builder.set_query(Some(&query));

        url_builder.to_string()
    }
}

#[async_trait]
impl<T> PlexInterface for PlexApi<T>
where
    T: ConfigInterface,
{
    async fn get_libraries(self) -> Result<Vec<PlexLibrary>, reqwest::Error> {
        let path = "/library/sections/";

        info!("Getting Plex libraries");
        let response: PlexLibraryResponse = self.make_request(path).await?;

        let library_count = response.media_container.directory.len();
        info!("Found {} libraries", { library_count });
        return Ok(response.media_container.directory);
    }

    async fn get_series(&self, library_id: u8) -> Result<Vec<PlexSeries>, reqwest::Error> {
        let path = format!("/library/sections/{}/all", library_id);

        info!("Getting Plex series for library id: {}", library_id);
        let response: PlexSeriesResponse = match __self.make_request(&path).await {
            Ok(x) => x,
            Err(e) => {
                error!("Error getting series for library_id: {}", library_id);
                return Err(e);
            }
        };

        let series_count = response.media_container.metadata.len();
        info!(
            "Found {} series for library_id {}",
            series_count, library_id
        );
        return Ok(response.media_container.metadata);
    }

    async fn get_seasons(&self, rating_key: &str) -> Result<Vec<PlexSeason>, reqwest::Error> {
        let path = format!("/library/metadata/{}/children", rating_key);

        info!("Getting Plex seasons for series id: {}", rating_key);
        let response: PlexSeasonResponse = match __self.make_request(&path).await {
            Ok(x) => x,
            Err(e) => {
                error!(
                    "Error getting seasons for series {}. Error: {}",
                    rating_key, e
                );
                return Err(e);
            }
        };

        let season_count = response.media_container.metadata.len();
        info!("Found {} seasons for series {}", season_count, rating_key);
        return Ok(response.media_container.metadata);
    }

    async fn popualte_seasons(
        &self,
        series: PlexSeries,
    ) -> Result<SeriesWithSeason, reqwest::Error> {
        let seasons = self.get_seasons(&series.rating_key).await?;

        Ok(SeriesWithSeason::new(series, seasons))
    }

    async fn get_all_seasons(
        &self,
        library_id: u8,
    ) -> Result<Vec<SeriesWithSeason>, reqwest::Error> {
        let series = self.get_series(library_id).await?;

        let futures = FuturesUnordered::new();
        for s in series {
            futures.push(self.popualte_seasons(s));
        }

        let futures = join_all(futures).await;

        let result: Vec<SeriesWithSeason> = futures
            .into_iter()
            .filter_map(|x| match x {
                Ok(x) => Some(x),
                Err(_) => None,
            })
            .collect();

        return Ok(result);
    }
}

#[cfg(test)]
mod tests {
    use std::fs;

    use serde::Deserialize;
    use wiremock::{
        matchers::{headers, method, path, query_param},
        Mock, MockServer, ResponseTemplate,
    };

    use crate::{services::config::config::MockConfig, utils::init_logger};

    use super::*;

    #[derive(Deserialize)]
    pub struct MockResponse {
        pub name: String,
        pub response: String,
    }

    fn get_response(response: &str) -> String {
        let response = String::from(response);
        let mut cwd = std::env::current_dir()
            .expect("Unable to get cwd")
            .display()
            .to_string();
        cwd.push_str("/test_data/plex_responses.json");
        let data =
            fs::read_to_string(cwd.as_str()).expect("Unable to read plex responses test file");
        let responses: Vec<MockResponse> =
            serde_json::from_str(data.as_str()).expect("Failed to parse plex responses test file");

        for saved_repsonse in responses {
            if saved_repsonse.name == response {
                return saved_repsonse.response;
            }
        }

        panic!("Failed to find response '{}'", response)
    }

    #[tokio::test]
    async fn test_get_libraries() {
        init_logger();

        let response = get_response("library");
        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/library/sections/"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param(
                "X-Plex-Token".to_string(),
                config_service.get_plex_token(),
            ))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        let libraries = plex_api.get_libraries().await.unwrap();

        assert_eq!(libraries.len(), 4);
        assert_eq!(libraries[0].clone().title, "Movies".to_string());
        assert_eq!(libraries[1].clone().title, "Anime".to_string());
    }

    #[tokio::test]
    async fn test_get_libraries_404_error_response() {
        init_logger();

        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(404))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        assert!(plex_api.get_libraries().await.is_err());
    }

    #[tokio::test]
    async fn test_get_series() {
        init_logger();

        let response = get_response("series");
        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/library/sections/1/all"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param(
                "X-Plex-Token".to_string(),
                config_service.get_plex_token(),
            ))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        let series = plex_api.get_series(1).await.unwrap();

        assert_eq!(series.len(), 2);
        assert_eq!(series[0].clone().rating_key, "24581".to_string());
        assert_eq!(series[1].clone().rating_key, "26268".to_string());
    }

    #[tokio::test]
    async fn test_get_series_404_error_response() {
        init_logger();

        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(404))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        assert!(plex_api.get_series(1).await.is_err());
    }

    #[tokio::test]
    async fn test_get_seasons() {
        init_logger();

        let response = get_response("seasons");
        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/library/metadata/24581/children"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param(
                "X-Plex-Token".to_string(),
                config_service.get_plex_token(),
            ))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        let series = plex_api.get_seasons("24581").await.unwrap();

        assert_eq!(series.len(), 1);
        assert_eq!(series[0].clone().rating_key, "24582".to_string());
    }

    #[tokio::test]
    async fn test_get_seasons_404_error_response() {
        init_logger();

        let mut config_service = MockConfig {
            plex_token: "123abc".to_string(),
            plex_base_url: "".to_string(),
            anilist_token: "".to_string(),
        };

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(404))
            .expect(1)
            .mount(&mock_server)
            .await;

        config_service.plex_base_url = mock_server.uri();

        let plex_api = PlexApi::new(config_service);

        assert!(plex_api.get_seasons("12345").await.is_err());
    }
}
