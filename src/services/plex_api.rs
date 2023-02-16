use crate::services::http_client::HttpClientInterface;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use tracing::{error, info};

pub struct PlexApi<T: HttpClientInterface + Send + Sync> {
    http_client: T,
}

impl<T: HttpClientInterface + Send + Sync> PlexApi<T> {
    pub fn new(http_client: T) -> PlexApi<T> {
        Self { http_client }
    }
}

#[async_trait]
pub trait PlexInterface {
    async fn get_libraries(&self) -> Vec<PlexLibrary>;
    fn get_series(&self);
    fn get_seasons(&self);
}

#[async_trait]
impl<T: HttpClientInterface + Send + Sync> PlexInterface for PlexApi<T> {
    // Get the libraries response string
    async fn get_libraries(&self) -> Vec<PlexLibrary> {
        info!("Getting Plex libraries");
        match self.http_client.get("/library/sections/".to_string()).await {
            Ok(response_string) => {
                let library_response = parse_libraries_response(response_string);
                info!("Found {} libraries", {
                    library_response.media_container.directory.len()
                });
                library_response.media_container.directory
            }
            Err(err) => {
                error!("Request to get Plex libraries failed: {}", err);
                vec![]
            }
        }
    }

    fn get_series(&self) {
        todo!()
    }

    fn get_seasons(&self) {
        todo!()
    }
}

fn parse_libraries_response(response: String) -> BaseResponse<DirectoryResponse<PlexLibrary>> {
    // Parse the library response
    let data =
        serde_json::from_str::<BaseResponse<DirectoryResponse<PlexLibrary>>>(response.as_str());
    match data {
        // Return parsed data
        Ok(library_response) => library_response,
        // Unable to parse json so again empty response
        Err(err) => {
            error!("Failed to parse Plex libraries response: {}", err);
            BaseResponse {
                media_container: DirectoryResponse { directory: vec![] },
            }
        }
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct PlexLibrary {
    title: String,
}

#[derive(Debug, Deserialize, Serialize)]
struct BaseResponse<T> {
    #[serde(rename = "MediaContainer")]
    media_container: T,
}

#[derive(Debug, Deserialize, Serialize)]
struct DirectoryResponse<T> {
    #[serde(rename = "Directory")]
    directory: Vec<T>,
}

#[derive(Serialize, Deserialize)]
pub struct MockResponse {
    name: String,
    response: String,
}

#[cfg(test)]
mod tests {
    use std::fs;

    use crate::services::http_client::MockHttpClient;

    use super::*;

    fn init_logger() {
        let _ = tracing_subscriber::fmt::try_init();
    }

    fn get_response(response: &str) -> String {
        let response = String::from(response);
        let mut cwd = std::env::current_dir()
            .expect("Unable to get cwd")
            .display()
            .to_string();
        cwd.push_str("/src/test_data/plex_responses.json");
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
        let http_client = MockHttpClient::new(response, false);
        let plex_api = PlexApi::new(http_client);

        let libraries = plex_api.get_libraries().await;

        assert_eq!(libraries.len(), 4);
    }

    #[tokio::test]
    async fn test_get_libraries_invalid_response() {
        init_logger();

        let http_client = MockHttpClient::new(String::new(), false);
        let plex_api = PlexApi::new(http_client);

        let libraries = plex_api.get_libraries().await;

        assert_eq!(libraries.len(), 0);
    }

    #[tokio::test]
    async fn test_get_libraries_error_response() {
        init_logger();

        let response = get_response("library");
        let http_client = MockHttpClient::new(response, true);
        let plex_api = PlexApi::new(http_client);

        let libraries = plex_api.get_libraries().await;

        assert_eq!(libraries.len(), 0);
    }
}
