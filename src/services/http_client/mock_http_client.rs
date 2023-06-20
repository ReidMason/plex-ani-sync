use super::http_client::HttpClient;
use async_trait::async_trait;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use serde::{de::DeserializeOwned, Serialize};

pub struct MockHttpClient {
    response: String,
    error: bool,
    headers: HeaderMap,
}

impl MockHttpClient {
    pub fn new(response: String, error: bool) -> MockHttpClient {
        Self {
            response,
            error,
            headers: HeaderMap::new(),
        }
    }
}

#[async_trait]
impl HttpClient for MockHttpClient {
    async fn get<R: DeserializeOwned>(&self, _url: String) -> Result<R, reqwest::Error> {
        if self.error {
            // We need to force a fake error because reqwest doesn't allow us to create one
            let client = reqwest::Client::new();
            client.get("").send().await?;
        }

        let response: R =
            serde_json::from_str(self.response.as_str()).expect("Error parsing test response data");

        return Ok(response);
    }

    async fn post<R: DeserializeOwned, T: Serialize + Send + Sync>(
        &self,
        _url: String,
        _data: T,
        _headers: Option<HeaderMap>,
    ) -> Result<R, reqwest::Error> {
        let response: R =
            serde_json::from_str(self.response.as_str()).expect("Error parsing test response data");

        return Ok(response);
    }

    fn add_header(&mut self, header_name: HeaderName, value: HeaderValue) {
        self.headers.insert(header_name, value);
    }

    fn get_headers(&self) -> &HeaderMap {
        &self.headers
    }
}
