use std::{thread, time::Duration};

use async_trait::async_trait;
use log::{error, info};
use reqwest::header::{self, HeaderMap, HeaderValue, ACCEPT, AUTHORIZATION, CONTENT_TYPE};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use serde_json::json;

use crate::services::{config::config::ConfigInterface, dbstore::dbstore::DbStore};

use super::anime_list_service::{AnimeListService, AnimeResult};

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
            HeaderValue::from_str(auth_value.as_str()).expect("Failed to parse anilist token"),
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
}

#[derive(Serialize)]
struct SearchAnimeVars {
    anime_name: String,
}

#[derive(Serialize)]
struct GetAnimeVars {
    anime_id: String,
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
                Page(perPage: 5) {
                    media(search: $anime_name, type: ANIME) {
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

    async fn get_anime(&self, anime_id: String) -> Result<Option<AnimeResult>, anyhow::Error> {
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
						year
						month
						day
					}
				}
			}
		}
	}"#;

        let vars = GetAnimeVars {
            anime_id: anime_id.clone(),
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
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnilistResponse<T> {
    pub data: T,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GetAnimeRequestResult {
    #[serde(rename = "Media")]
    pub media: AnimeResult,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnimeSearchRequestResult {
    #[serde(rename = "Page")]
    pub page: Page,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
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
            .get_anime(String::from(anime_id))
            .await
            .expect("Failed to get anilist anime result")
            .expect("No anime found");

        assert_eq!(response.id, 11757);
    }
}
