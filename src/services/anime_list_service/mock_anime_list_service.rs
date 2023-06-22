use std::fs;

use async_trait::async_trait;
use serde::Deserialize;

use super::{
    anilist_service::{AnilistResponse, AnimeSearchRequestResult},
    anime_list_service::{AnimeListService, AnimeResult},
};

pub struct MockAnimeListService {}

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
    cwd.push_str("/test_data/anilist_responses.json");
    let data =
        fs::read_to_string(cwd.as_str()).expect("Unable to read anilist responses test file");
    let responses: Vec<MockResponse> =
        serde_json::from_str(data.as_str()).expect("Failed to parse anilist responses test file");

    for saved_repsonse in responses {
        if saved_repsonse.name == response {
            return saved_repsonse.response;
        }
    }

    panic!("Failed to find response '{}'", response)
}

#[async_trait]
impl AnimeListService for MockAnimeListService {
    async fn search_anime(&self, _search_term: &str) -> Result<Vec<AnimeResult>, anyhow::Error> {
        let response = get_response("anime_search");
        let result: AnilistResponse<AnimeSearchRequestResult> =
            serde_json::from_str(response.as_str()).expect("Failed to deserialize test data");

        return Ok(result.data.page.media);
    }

    async fn get_anime(&self, anime_id: &str) -> Result<Option<AnimeResult>, anyhow::Error> {
        todo!()
    }

    async fn find_sequel(
        &self,
        anime_result: AnimeResult,
    ) -> Result<Option<AnimeResult>, anyhow::Error> {
        todo!()
    }
}
