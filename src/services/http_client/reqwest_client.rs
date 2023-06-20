use async_trait::async_trait;
use reqwest::header::{self, HeaderMap, HeaderName, HeaderValue};
use reqwest::Error;
use serde::de::DeserializeOwned;
use serde::Serialize;

use super::http_client::HttpClient;

pub struct ReqwestHttpClient {
    client: reqwest::Client,
    headers: HeaderMap,
}

impl ReqwestHttpClient {
    pub fn new() -> Self {
        let client = reqwest::Client::builder();
        let client = client.build().unwrap();
        Self {
            client,
            headers: header::HeaderMap::new(),
        }
    }
}

#[async_trait]
impl HttpClient for ReqwestHttpClient {
    fn add_header(&mut self, header_name: HeaderName, value: HeaderValue) {
        self.headers.insert(header_name, value);
    }

    async fn get<R: DeserializeOwned>(&self, url: String) -> Result<R, Error> {
        Ok(self.client.get(url).send().await?.json::<R>().await?)
    }

    async fn post<R: DeserializeOwned, T: Serialize + Send + Sync>(
        &self,
        url: String,
        data: T,
        headers: Option<HeaderMap>,
    ) -> Result<R, reqwest::Error> {
        let mut request = self.client.post(url).json(&data);
        if let Some(headers) = headers {
            request = request.headers(headers);
        }
        Ok(request.send().await?.json::<R>().await?)
    }

    fn get_headers(&self) -> &HeaderMap {
        &self.headers
    }
}
