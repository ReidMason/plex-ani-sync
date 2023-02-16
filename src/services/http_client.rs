use crate::services::config::ConfigInterface;
use async_trait::async_trait;
use reqwest::{
    header::{ACCEPT, CONTENT_TYPE},
    Error,
};

use super::plex_api::MockResponse;

struct HttpClient<T: ConfigInterface + Send + Sync> {
    config: T,
}

#[async_trait]
pub trait HttpClientInterface {
    async fn get(&self, url: String) -> Result<String, Error>;
}

pub struct MockHttpClient {
    response: String,
    error: bool,
}

impl MockHttpClient {
    pub fn new(response: String, error: bool) -> MockHttpClient {
        Self { response, error }
    }
}

#[async_trait]
impl HttpClientInterface for MockHttpClient {
    async fn get(&self, url: String) -> Result<String, Error> {
        if self.error {
            // We need to force a fake error because reqwest doens't allow us to create one
            let client = reqwest::Client::new();
            client.get("").send().await?;
        }

        return Ok(self.response.clone());
    }
}

#[async_trait]
impl<T: ConfigInterface + Send + Sync> HttpClientInterface for HttpClient<T> {
    async fn get(&self, url: String) -> Result<String, Error> {
        let base_url = self.config.get_plex_base_url();
        let token = self.config.get_plex_token();
        let request_url = base_url + &url + "?X-Plex-Token=" + token.as_str();

        let client = reqwest::Client::new();
        let response = client
            .get(request_url)
            .header(CONTENT_TYPE, "application/json")
            .header(ACCEPT, "application/json")
            .send()
            .await?;

        let content = response.text().await?;
        Ok(content)
    }
}
