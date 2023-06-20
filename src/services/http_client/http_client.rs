use async_trait::async_trait;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use serde::{de::DeserializeOwned, Serialize};

#[async_trait]
pub trait HttpClient: Send + Sync {
    async fn get<R: DeserializeOwned>(&self, url: String) -> Result<R, reqwest::Error>;
    async fn post<R: DeserializeOwned, T: Serialize + Send + Sync>(
        &self,
        url: String,
        data: T,
        headers: Option<HeaderMap>,
    ) -> Result<R, reqwest::Error>;
    fn add_header(&mut self, header_name: HeaderName, value: HeaderValue);
    fn get_headers(&self) -> &HeaderMap;
}
