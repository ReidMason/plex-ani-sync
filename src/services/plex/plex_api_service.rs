use async_trait::async_trait;
use futures::{future::join_all, stream::FuturesUnordered};
use log::{error, info};
use reqwest::header::{self, HeaderMap, HeaderValue, ACCEPT, AUTHORIZATION, CONTENT_TYPE};
use serde::de::DeserializeOwned;
use tracing::instrument;
use url::Url;

use crate::services::plex::plex_api::{
    PlexLibraryResponse, PlexSeasonResponse, PlexSeriesResponse,
};

use super::plex_api::{
    PlexEpisode, PlexEpisodesResponse, PlexInterface, PlexSeason, PlexSeries, ResponsePlexLibrary,
    ResponsePlexSeries,
};

#[derive(Debug)]
pub struct PlexApi {
    plex_url: String,
    plex_token: String,
    http_client: reqwest::Client,
    headers: header::HeaderMap,
}

impl PlexApi {
    pub fn new(plex_url: String, plex_token: String) -> Self {
        let mut header_map = header::HeaderMap::new();
        header_map.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
        header_map.insert(ACCEPT, HeaderValue::from_static("application/json"));

        Self {
            plex_url,
            plex_token,
            http_client: reqwest::Client::new(),
            headers: header_map,
        }
    }

    fn get_headers(&self) -> HeaderMap {
        let auth_value = format!("Bearer {}", self.plex_token);
        let mut headers = self.headers.clone();
        headers.insert(
            AUTHORIZATION,
            HeaderValue::from_str(auth_value.as_str()).expect("Failed to parse anilist token"),
        );
        headers
    }

    async fn make_request<T>(&self, path: &str) -> Result<T, reqwest::Error>
    where
        T: DeserializeOwned,
    {
        let url = self.build_request_url(path);
        self.http_client
            .get(&url)
            .headers(self.get_headers())
            .send()
            .await?
            .json::<T>()
            .await
    }

    fn build_request_url(&self, path: &str) -> String {
        let base_url = &self.plex_url;
        let token = &self.plex_token;

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
impl PlexInterface for PlexApi {
    async fn get_libraries(self) -> Result<Vec<ResponsePlexLibrary>, reqwest::Error> {
        let path = "/library/sections/";

        info!("Getting Plex libraries");
        let response: PlexLibraryResponse = self.make_request(path).await?;

        let library_count = response.media_container.directory.len();
        info!("Found {} libraries", { library_count });
        return Ok(response.media_container.directory);
    }

    #[instrument(skip(self))]
    async fn get_series(&self, library_id: u8) -> Result<Vec<ResponsePlexSeries>, reqwest::Error> {
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

    async fn populate_episodes(&self, season: &mut PlexSeason) -> Result<(), reqwest::Error> {
        let path = format!("/library/metadata/{}/children", season.rating_key);

        let response: PlexEpisodesResponse = self.make_request(&path).await?;
        season.episodes = response
            .media_container
            .metadata
            .into_iter()
            .map(|x| PlexEpisode::from(x))
            .collect();

        Ok(())
    }

    async fn populate_seasons(&self, series: &mut PlexSeries) -> Result<(), reqwest::Error> {
        let path = format!("/library/metadata/{}/children", series.rating_key);

        let response: PlexSeasonResponse = self.make_request(&path).await?;
        let mut seasons: Vec<PlexSeason> = response
            .media_container
            .metadata
            .into_iter()
            .map(|x| PlexSeason::from(x))
            .collect();

        let futures = FuturesUnordered::new();
        for season in seasons.iter_mut() {
            futures.push(self.populate_episodes(season));
        }

        join_all(futures).await;

        series.seasons = seasons;
        Ok(())
    }

    async fn get_full_series_data(
        &self,
        library_id: u8,
    ) -> Result<Vec<PlexSeries>, reqwest::Error> {
        let all_series = self.get_series(library_id).await?;
        let mut all_series: Vec<PlexSeries> = all_series
            .into_iter()
            .map(|x| PlexSeries::from(x))
            .collect();

        for chunk in all_series.chunks_mut(50) {
            info!("Processing chunk");
            let futures = FuturesUnordered::new();
            for series in chunk.iter_mut() {
                futures.push(self.populate_seasons(series));
            }
            join_all(futures).await;
        }

        return Ok(all_series);
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
    async fn test_get_full_series_data() {
        let plex_token = "123abc".to_string();
        let mock_server = MockServer::start().await;
        let series_response = get_response("series");
        Mock::given(method("GET"))
            .and(path("/library/sections/1/all"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param("X-Plex-Token".to_string(), &plex_token))
            .respond_with(ResponseTemplate::new(200).set_body_string(series_response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let seasons_response = get_response("seasons");
        Mock::given(method("GET"))
            .and(path("/library/metadata/17456/children"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param("X-Plex-Token".to_string(), &plex_token))
            .respond_with(ResponseTemplate::new(200).set_body_string(seasons_response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let episodes_response = get_response("episodes");
        Mock::given(method("GET"))
            .and(path("/library/metadata/30037/children"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param("X-Plex-Token".to_string(), &plex_token))
            .respond_with(ResponseTemplate::new(200).set_body_string(episodes_response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let plex_service = PlexApi::new(mock_server.uri(), plex_token);

        let data = plex_service.get_full_series_data(1).await.unwrap();
        assert_eq!(1, data.len());
        let series = &data[0];
        assert_eq!(5, series.seasons.len());
        let seasons = &series.seasons;
        assert_eq!(8, seasons[0].episodes.len());
    }

    #[tokio::test]
    async fn test_get_libraries() {
        init_logger();

        let response = get_response("library");
        let plex_token = "123abc".to_string();

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/library/sections/"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param("X-Plex-Token".to_string(), &plex_token))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let plex_api = PlexApi::new(mock_server.uri(), plex_token);

        let libraries = plex_api.get_libraries().await.unwrap();

        assert_eq!(libraries.len(), 4);
        assert_eq!(libraries[0].clone().title, "Movies".to_string());
        assert_eq!(libraries[1].clone().title, "Anime".to_string());
    }

    #[tokio::test]
    async fn test_get_libraries_404_error_response() {
        init_logger();

        let plex_token = "123abc".to_string();

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(404))
            .expect(1)
            .mount(&mock_server)
            .await;

        let plex_api = PlexApi::new(mock_server.uri(), plex_token);

        assert!(plex_api.get_libraries().await.is_err());
    }

    #[tokio::test]
    async fn test_get_series() {
        init_logger();

        let response = get_response("series");
        let plex_token = "123abc".to_string();

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/library/sections/1/all"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(query_param("X-Plex-Token".to_string(), &plex_token))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let plex_api = PlexApi::new(mock_server.uri(), plex_token);

        let series = plex_api.get_series(1).await.unwrap();

        assert_eq!(1, series.len());
        assert_eq!("17456".to_string(), series[0].clone().rating_key);
    }

    #[tokio::test]
    async fn test_get_series_404_error_response() {
        init_logger();

        let plex_token = "123abc".to_string();

        let mock_server = MockServer::start().await;
        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(404))
            .expect(1)
            .mount(&mock_server)
            .await;

        let plex_api = PlexApi::new(mock_server.uri(), plex_token);

        assert!(plex_api.get_series(1).await.is_err());
    }
}
