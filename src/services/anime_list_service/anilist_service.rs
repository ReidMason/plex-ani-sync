use std::{thread, time::Duration};

use async_trait::async_trait;
use log::{error, info};
use reqwest::header::{self, HeaderMap, HeaderValue, ACCEPT, AUTHORIZATION, CONTENT_TYPE};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use serde_json::json;

use crate::services::{config::config::ConfigInterface, dbstore::dbstore::DbStore};

use super::anime_list_service::{
    AnilistWatchStatus, AnimeListEntry, AnimeListService, AnimeResult, RelationType,
};

// We need to use to visit this page then we'll redirect them back to the main page to get the auth
// code token thing https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/implicit-grant
// https://anilist.co/api/v2/oauth/authorize?client_id=4688&response_type=token
pub struct AnilistService<J: ConfigInterface, K: DbStore> {
    config: J,
    dbstore: K,
    http_client: reqwest::Client,
    base_url: String,
}

#[derive(Serialize)]
struct GraphQlBody {
    query: String,
    variables: serde_json::Value,
}

impl<T, J> AnilistService<T, J>
where
    T: ConfigInterface,
    J: DbStore,
{
    pub fn new(config: T, dbstore: J, base_url: Option<String>) -> Self {
        AnilistService {
            config,
            dbstore,
            http_client: reqwest::Client::new(),
            base_url: base_url.unwrap_or(String::from("https://graphql.anilist.co/")),
        }
    }

    fn get_headers(&self) -> HeaderMap {
        let auth_value = format!("Bearer {}", self.config.get_anilist_token());

        let mut headers = header::HeaderMap::new();
        headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
        headers.insert(ACCEPT, HeaderValue::from_static("application/json"));
        headers.insert(
            AUTHORIZATION,
            HeaderValue::from_str(&auth_value).expect("Failed to parse anilist token"),
        );
        headers
    }

    async fn make_request<R: DeserializeOwned, D: Serialize>(
        &self,
        data: D,
    ) -> Result<AnilistResponse<R>, anyhow::Error> {
        let response = self
            .http_client
            .post(&self.base_url)
            .json(&data)
            .headers(self.get_headers())
            .send()
            .await?;

        let response_body = match response.text().await {
            Ok(x) => x,
            Err(e) => {
                error!("Failed getting text from response");
                return Err(e.into());
            }
        };

        let response: AnilistResponse<R> = match serde_json::from_str(&response_body) {
            Ok(x) => x,
            Err(e) => {
                error!(
                    "Failed to parse Anilist response. Error: {} \n Response: {}",
                    e, response_body
                );
                return Err(e.into());
            }
        };

        // Sleep to avoid rate limit
        let ten_millis = Duration::from_millis(1000);
        thread::sleep(ten_millis);

        Ok(response)
    }

    async fn get_user(&self) -> Result<AnilistUser, anyhow::Error> {
        let query = r#"query {
                        Viewer {
                            id
                            name
                        }
                    }"#;

        let data = GraphQlBody {
            query: String::from(query),
            variables: json!({}),
        };

        let result: AnilistResponse<AnilistUserResponse> = self.make_request(data).await?;

        return Ok(result.data.viewer);
    }
}

#[derive(Deserialize)]
struct AnilistUserResponse {
    #[serde(rename = "Viewer")]
    viewer: AnilistUser,
}

#[derive(Deserialize)]
struct AnilistUser {
    id: u32,
    name: String,
}

#[derive(Serialize)]
struct MediaListCollectionVars {
    user_id: u32,
}

#[derive(Serialize)]
struct SearchAnimeVars {
    anime_name: String,
}

#[derive(Serialize)]
struct GetAnimeVars {
    anime_id: String,
}

#[derive(Serialize)]
struct UpdateAnimeListEntryVars {
    media_id: u32,
    status: AnilistWatchStatus,
    progress: u16,
}

#[async_trait]
impl<T: ConfigInterface, J: DbStore> AnimeListService for AnilistService<T, J> {
    async fn search_anime(&self, search_term: &str) -> Result<Vec<AnimeResult>, anyhow::Error> {
        let result = self
            .dbstore
            .get_cached_anime_search_result(search_term)
            .await;

        if let Some(result) = result {
            info!(
                "Found cached anilist search response for search term: {}",
                search_term
            );
            return Ok(result);
        }

        info!("Quering anilist API for search term: {}", search_term);

        let query = r#"query ($anime_name: String) {
                Page(perPage: 10) {
                    media(search: $anime_name, type: ANIME, sort: SEARCH_MATCH) {
                        id
                        format
                        episodes
                        synonyms
                        status
                        endDate {
                            year
                            month
                            day
                        }
                        startDate {
                            year
                            month
                            day
                        }
                        title {
                            english
                            romaji
                        }
                        relations {
                            edges {
                                relationType
                            }
                            nodes {
                                id
                                format
                                episodes
                                endDate {
                                    year
                                    month
                                    day
                                }
                                startDate {
                                    year
                                    month
                                    day
                                }
                            }
                        }
                    }
                }
            }"#;

        let vars = SearchAnimeVars {
            anime_name: search_term.to_string(),
        };
        let data = GraphQlBody {
            query: String::from(query),
            variables: json!(vars),
        };

        let result: AnilistResponse<AnimeSearchRequestResult> = self.make_request(data).await?;

        self.dbstore
            .save_cached_anime_search_result(search_term, result.data.page.media.clone())
            .await;

        return Ok(result.data.page.media);
    }

    async fn get_anime(&self, anime_id: &str) -> Result<Option<AnimeResult>, anyhow::Error> {
        let result = self.dbstore.get_cached_anime_result(&anime_id).await;

        if result.is_some() {
            info!(
                "Found cached anilist anime response for anime_i: {}",
                anime_id
            );
            return Ok(result);
        }

        info!("Quering anilist API for anime_id: {}", anime_id);

        let query = r#"query ($anime_id: Int) {
    Media(id: $anime_id, type: ANIME) {
      id
      format
      episodes
      synonyms
      status
      endDate {
        year
        month
        day
      }
      startDate {
        year
        month
        day
      }
      title {
        english
        romaji
      }
      relations {
        edges {
          relationType
        }
        nodes {
          id
          format
          endDate {
            year
            month
            day
          }
          startDate {
  }"#;

        let vars = GetAnimeVars {
            anime_id: anime_id.to_string(),
        };
        let data = GraphQlBody {
            query: String::from(query),
            variables: json!(vars),
        };

        let result: AnilistResponse<GetAnimeRequestResult> = self.make_request(data).await?;

        self.dbstore
            .save_cached_anime_result(&anime_id, result.data.media.clone())
            .await;

        return Ok(Some(result.data.media));
    }

    async fn find_sequel(
        &self,
        anime_result: AnimeResult,
    ) -> Result<Option<AnimeResult>, anyhow::Error> {
        let nodes = anime_result.relations.nodes;
        let edges = anime_result.relations.edges;
        for (edge, node) in edges.iter().zip(nodes) {
            if edge.relation_type == RelationType::Sequel {
                return self.get_anime(&node.id.to_string()).await;
            }
        }

        return Ok(None);
    }

    async fn get_list(&self, user_id: u32) -> Result<Vec<AnimeListEntry>, anyhow::Error> {
        let query = r#"query($user_id: Int) {
    MediaListCollection(userId: $user_id, type: ANIME) {
        lists {
            name
            status
            isCustomList
            entries {
                mediaId
                progress
            }
        }
    }
}"#;

        let vars = MediaListCollectionVars { user_id };
        let data = GraphQlBody {
            query: String::from(query),
            variables: json!(vars),
        };

        let result: AnilistResponse<AnilistListsMediaListCollectionResponse> =
            self.make_request(data).await?;
        let mut anime_list: Vec<AnimeListEntry> = vec![];

        for list in result.data.media_list_collection.lists {
            let status = match list.status {
                Some(x) => x,
                None => continue,
            };

            for entry in list.entries {
                anime_list.push(AnimeListEntry {
                    status: status.clone(),
                    progress: entry.progress,
                    media_id: entry.media_id,
                });
            }
        }

        Ok(anime_list)
    }

    async fn update_list_entry(
        &self,
        media_id: u32,
        status: AnilistWatchStatus,
        progress: u16,
    ) -> Result<SaveMediaListEntry, anyhow::Error> {
        let query = r#"mutation ($media_id: Int, $status: MediaListStatus, $progress: Int) {
                SaveMediaListEntry (mediaId: $media_id, status: $status, progress: $progress) {
                    id
                    status,
                    progress
                }
            }"#;

        let vars = UpdateAnimeListEntryVars {
            progress,
            status,
            media_id,
        };

        let data = GraphQlBody {
            query: String::from(query),
            variables: json!(vars),
        };

        let result: AnilistResponse<SaveMediaListEntryResponse> = self.make_request(data).await?;

        Ok(result.data.save_media_list_entry)
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SaveMediaListEntry {
    pub id: u32,
    pub status: AnilistWatchStatus,
    pub progress: u16,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SaveMediaListEntryResponse {
    #[serde(rename = "SaveMediaListEntry")]
    pub save_media_list_entry: SaveMediaListEntry,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnilistListsMediaListCollectionResponse {
    #[serde(rename = "MediaListCollection")]
    pub media_list_collection: AnilistListsResponse,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnilistListsResponse {
    pub lists: Vec<AnilistList>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnilistList {
    pub name: String,
    #[serde(rename = "isCustomList")]
    pub is_custom_list: bool,
    pub status: Option<AnilistWatchStatus>,
    pub entries: Vec<AnilistListEntryResponse>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnilistListEntryResponse {
    #[serde(rename = "mediaId")]
    pub media_id: u32,
    pub progress: u16,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnilistResponse<T> {
    pub data: T,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GetAnimeRequestResult {
    #[serde(rename = "Media")]
    pub media: AnimeResult,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnimeSearchRequestResult {
    #[serde(rename = "Page")]
    pub page: Page,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Page {
    pub media: Vec<AnimeResult>,
}

#[cfg(test)]
mod tests {
    use crate::{
        services::{config::config::ConfigService, dbstore::sqlite::Sqlite},
        utils::init_logger,
    };
    use serde::Deserialize;
    use std::fs;
    use wiremock::{
        matchers::{bearer_token, body_partial_json, headers, method, path},
        Mock, MockServer, ResponseTemplate,
    };

    use super::*;

    #[derive(Serialize, Deserialize)]
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
        let responses: Vec<MockResponse> = serde_json::from_str(data.as_str())
            .expect("Failed to parse anilist responses test file");

        for saved_repsonse in responses {
            if saved_repsonse.name == response {
                return saved_repsonse.response;
            }
        }

        panic!("Failed to find response '{}'", response)
    }

    #[tokio::test]
    async fn test_update_entry() {
        init_logger();

        let response = r#"{
        "data": {
        "SaveMediaListEntry": {
            "id": 89949907,
            "status": "PLANNING",
            "progress": 0
        }
    }
}"#;
        let mut db_store = Sqlite::new("sqlite::memory:").await;
        db_store.migrate().await;

        let mut config = db_store.get_config().await;
        config.anilist_token = "testToken123".to_string();

        let config_service = ConfigService::new(config);

        let mock_server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(bearer_token(config_service.get_anilist_token()))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let list_service = AnilistService::new(config_service, db_store, Some(mock_server.uri()));

        let response = list_service
            .update_list_entry(12345, AnilistWatchStatus::Planning, 5)
            .await
            .expect("Failed to update anilist entry");

        assert_eq!(AnilistWatchStatus::Planning, response.status);
    }

    #[tokio::test]
    async fn test_get_list() {
        init_logger();

        let response = get_response("get_list");
        let mut db_store = Sqlite::new("sqlite::memory:").await;
        db_store.migrate().await;

        let mut config = db_store.get_config().await;
        config.anilist_token = "testToken123".to_string();

        let config_service = ConfigService::new(config);

        let mock_server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(bearer_token(config_service.get_anilist_token()))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let list_service = AnilistService::new(config_service, db_store, Some(mock_server.uri()));

        let response = list_service
            .get_list(12345)
            .await
            .expect("Failed to get anilist list");

        assert_eq!(9, response.len());
        assert_eq!(136149, response[0].media_id);
        assert_eq!(0, response[0].progress);
        assert_eq!(8, response[8].progress);
    }

    #[tokio::test]
    async fn test_get_user() {
        init_logger();

        let response = get_response("get_user");
        let mut db_store = Sqlite::new("sqlite::memory:").await;
        db_store.migrate().await;

        let mut config = db_store.get_config().await;
        config.anilist_token = "testToken123".to_string();

        // let db_config = db_store.get_config().await;
        // let config_service = ConfigService::new(db_config.clone());
        let config_service = ConfigService::new(config);

        let mock_server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/"))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(bearer_token(config_service.get_anilist_token()))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let list_service = AnilistService::new(config_service, db_store, Some(mock_server.uri()));

        let response = list_service
            .get_user()
            .await
            .expect("Failed to get anilist user");

        assert_eq!("UserName", response.name);
        assert_eq!(12345, response.id);
    }

    #[tokio::test]
    async fn test_anilist_search() {
        init_logger();

        let response = get_response("anime_search");
        let mut db_store = Sqlite::new("sqlite::memory:").await;
        db_store.migrate().await;

        let mut config = db_store.get_config().await;
        config.anilist_token = "testToken123".to_string();
        let config_service = ConfigService::new(config);

        let search_term = "Sword Art Online";
        let expected_body = json!({
            "variables": {
                "anime_name": search_term
            }
        });
        let mock_server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/"))
            .and(body_partial_json(expected_body))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(bearer_token(config_service.get_anilist_token()))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .expect(1)
            .mount(&mock_server)
            .await;

        let list_service = AnilistService::new(config_service, db_store, Some(mock_server.uri()));

        let response = list_service
            .search_anime(search_term)
            .await
            .expect("Failed to get anilist results");

        assert_eq!(response.len(), 5);
    }

    #[tokio::test]
    async fn test_anilist_get_anime() {
        init_logger();

        let response = get_response("get_anime");
        let mut db_store = Sqlite::new("sqlite::memory:").await;
        db_store.migrate().await;

        let mut config = db_store.get_config().await;
        config.anilist_token = "testToken123".to_string();
        let config_service = ConfigService::new(config);

        let anime_id = "11757";
        let expected_body = json!({
            "variables": {
                "anime_id": anime_id
            }
        });
        let mock_server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/"))
            .and(body_partial_json(expected_body))
            .and(headers(CONTENT_TYPE, vec!["application/json"]))
            .and(headers(ACCEPT, vec!["application/json"]))
            .and(bearer_token(config_service.get_anilist_token()))
            .respond_with(ResponseTemplate::new(200).set_body_string(response))
            .mount(&mock_server)
            .await;

        let list_service = AnilistService::new(config_service, db_store, Some(mock_server.uri()));

        let response = list_service
            .get_anime(anime_id)
            .await
            .expect("Failed to get anilist anime result")
            .expect("No anime found");

        assert_eq!(response.id, 11757);
    }
}
