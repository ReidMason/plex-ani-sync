use anyhow::Ok;
use async_trait::async_trait;

use crate::services::anime_list_service::anime_list_service::{AnimeListService, AnimeResult};
use crate::services::plex_api::plex_api::{PlexSeason, SeriesWithSeason};

#[async_trait]
pub trait MappingHandlerInterface {
    async fn find_mapping(&self, series: SeriesWithSeason) -> Result<Vec<Mapping>, anyhow::Error>;
    async fn find_match_for_season(
        &self,
        season: &PlexSeason,
    ) -> Result<Option<AnimeResult>, anyhow::Error>;
}

#[derive(Clone, Debug)]
pub struct Mapping {
    id: u32,

    plex_id: String,
    plex_episode_start: u16,
    pub season_length: u16,

    pub anime_list_id: String,
    episode_start: u16,

    enabled: bool,
    ignored: bool,
}

pub struct MappingHandler<T>
where
    T: AnimeListService,
{
    anime_list_service: T,
}

impl<T> MappingHandler<T>
where
    T: AnimeListService,
{
    pub fn new(anime_list_service: T) -> Self {
        Self { anime_list_service }
    }
}

fn cleanup_string(string: &str) -> String {
    string.replace([':', ' '], "").trim().to_lowercase()
}

fn compare_strings(string1: &str, string2: &str) -> bool {
    cleanup_string(string1) == cleanup_string(string2)
}

#[derive(Clone)]
struct ResultScore {
    result: AnimeResult,
    score: u16,
}

fn find_match(results: Vec<AnimeResult>, target: &PlexSeason) -> Option<AnimeResult> {
    let mut potential_matches: Vec<ResultScore> = results
        .into_iter()
        .map(|x| ResultScore {
            result: x,
            score: 0,
        })
        .collect();

    potential_matches.iter_mut().for_each(|potential_match| {
        // Match episode count
        if potential_match.result.episodes == Some(target.episodes) {
            potential_match.score += 100;
        }

        // Match title
        let mut potential_titles = [
            &format!("{} season {}", target.parent_title, target.index),
            &format!("{} {}", target.parent_title, target.index),
        ];

        let is_first_season = target.index == 1;
        if is_first_season {
            potential_titles[0] = &target.parent_title;
        }

        for potential_title in potential_titles {
            if let Some(english_title) = &potential_match.result.title.english {
                if compare_strings(english_title, &potential_title) {
                    potential_match.score += 50;
                }
            }

            if compare_strings(&potential_match.result.title.romaji, &potential_title) {
                potential_match.score += 50;
            }

            for synonym in potential_match.result.synonyms.clone() {
                if compare_strings(&synonym, &potential_title) {
                    potential_match.score += 10;
                }
            }
        }
    });

    potential_matches.sort_by_key(|x| x.score);
    // println!("Looking for: '{} {}'", target.parent_title, target.index,);
    // potential_matches.clone().into_iter().for_each(|x| {
    //     println!(
    //         "  - {} - Score: {}",
    //         x.result.title.english.unwrap_or(x.result.title.romaji),
    //         x.score
    //     );
    // });

    let min_score = 0;
    potential_matches = potential_matches
        .into_iter()
        .filter(|x| x.score > min_score)
        .collect();

    match potential_matches.pop() {
        Some(x) => Some(x.result),
        None => None,
    }
}

fn get_mapped_episode_count(mappings: &Vec<Mapping>, rating_key: &str) -> u16 {
    mappings
        .iter()
        .filter_map(|x| {
            if x.plex_id == rating_key {
                return Some(x.season_length);
            }
            None
        })
        .sum::<u16>()
}

fn get_prev_mapping(mappings: &Vec<Mapping>, season: &PlexSeason) -> Option<Mapping> {
    let mut prev_mappings: Vec<&Mapping> = mappings
        .iter()
        .filter(|x| x.plex_id == season.rating_key)
        .collect();

    if !prev_mappings.is_empty() {
        prev_mappings.sort_by_key(|x| x.plex_episode_start);
        prev_mappings.reverse();
        return Some(prev_mappings[0].to_owned());
    }

    None
}

#[async_trait]
impl<T> MappingHandlerInterface for MappingHandler<T>
where
    T: AnimeListService + Sync + Send,
{
    async fn find_match_for_season(
        &self,
        season: &PlexSeason,
    ) -> Result<Option<AnimeResult>, anyhow::Error> {
        let results = self
            .anime_list_service
            .search_anime(&season.parent_title)
            .await?;

        if results.is_empty() {
            return Ok(None);
        }

        return Ok(find_match(results, season));
    }

    async fn find_mapping(&self, series: SeriesWithSeason) -> Result<Vec<Mapping>, anyhow::Error> {
        // TODO: Reduce the chance of mapping errors by building up a vec of mappings for one
        // season then only push them if all the episodes are covered
        let mut mappings: Vec<Mapping> = vec![];
        // Just skip big series for now
        if series.seasons.len() > 6 {
            return Ok(mappings);
        }

        for (i, season) in series.seasons.iter().enumerate() {
            let is_specials_season = season.index == 0;
            if is_specials_season {
                continue;
            }

            // We start by just mapping the first season
            let is_first_season = season.index == 1;
            if is_first_season {
                let found_match = self.find_match_for_season(season).await?;
                let found_match = match found_match {
                    Some(x) => x,
                    None => return Ok(mappings),
                };

                let mapping = Mapping {
                    id: 1,
                    plex_id: season.rating_key.clone(),
                    plex_episode_start: 0,
                    season_length: found_match.episodes.unwrap_or(season.episodes),
                    anime_list_id: found_match.id.to_string(),
                    episode_start: 0,
                    enabled: true,
                    ignored: false,
                };
                mappings.push(mapping);
            }

            if !mappings.is_empty() {
                let limit = 5;
                let mut counter = 0;
                while counter < limit
                    && get_mapped_episode_count(&mappings, &season.rating_key) < season.episodes
                {
                    // println!("");
                    counter += 1;
                    let mut prev_mapping = get_prev_mapping(&mappings, season);
                    // If we got a prev mapping matching the current season this is a multi entry
                    // season

                    let mutli_entry_season = prev_mapping.is_some();
                    if prev_mapping.is_none() && i > 0 {
                        let prev_plex_season = &series.seasons[i - 1];
                        prev_mapping = get_prev_mapping(&mappings, prev_plex_season);
                    }

                    let prev_mapping = match prev_mapping {
                        Some(x) => x,
                        None => return Ok(mappings),
                    };

                    // println!(
                    //     "Plex season {} previous mapping '{:?}'",
                    //     season.index, prev_mapping
                    // );

                    let prev_mapping_entry = self
                        .anime_list_service
                        .get_anime(&prev_mapping.anime_list_id)
                        .await?;

                    let prev_mapping_entry = match prev_mapping_entry {
                        Some(x) => x,
                        None => return Ok(mappings),
                    };

                    let sequel = self
                        .anime_list_service
                        .find_sequel(prev_mapping_entry)
                        .await?;

                    let mut sequel = match sequel {
                        Some(x) => x,
                        None => return Ok(mappings),
                    };

                    // println!("Found sequel '{}'", sequel.title.english.as_ref().unwrap());

                    let current_mapped_episodes =
                        get_mapped_episode_count(&mappings, &season.rating_key);

                    // TODO: Don't just use 0 if the episode number isn't known
                    // This likely means it's still releasing but we need to check
                    if current_mapped_episodes + sequel.episodes.unwrap_or(0) > season.episodes {
                        let found_match = find_match(vec![sequel], season);
                        let found_match = match found_match {
                            Some(x) => x,
                            None => {
                                // println!("Didn't find a good match for the season");
                                return Ok(mappings);
                            }
                        };
                        sequel = found_match;
                    }

                    let mut plex_episode_start = 0;
                    if mutli_entry_season {
                        // println!("It's a multi season!");
                        plex_episode_start =
                            prev_mapping.plex_episode_start + prev_mapping.season_length;
                    }

                    let mapping = Mapping {
                        id: 1,
                        plex_id: season.rating_key.clone(),
                        plex_episode_start,
                        season_length: sequel.episodes.unwrap_or(season.episodes),
                        anime_list_id: sequel.id.to_string(),
                        episode_start: 0,
                        enabled: true,
                        ignored: false,
                    };
                    // println!("Added mapping: {:?}", mapping);
                    mappings.push(mapping);
                }
            }
        }
        return Ok(mappings);
    }
}

#[cfg(test)]
mod tests {
    use crate::{
        services::{
            anime_list_service::anilist_service::AnilistService,
            config::config::ConfigService,
            dbstore::{dbstore::DbStore, sqlite::Sqlite},
            plex_api::plex_api::{PlexSeason, PlexSeries},
        },
        utils::{get_db_file_location, init_logger},
    };

    use super::*;

    async fn init() -> MappingHandler<AnilistService<ConfigService, Sqlite>> {
        init_logger();

        let mut db_store = Sqlite::new(&get_db_file_location()).await;
        db_store.migrate().await;

        let config = db_store.get_config().await;
        let config_service = ConfigService::new(config);

        let list_service = AnilistService::new(config_service, db_store, None);

        MappingHandler::new(list_service)
    }

    #[tokio::test]
    async fn test_one_to_one_mapping() {
        let mapper = init().await;

        let series = SeriesWithSeason {
            series: PlexSeries {
                title: "Mysterious Girlfriend X".to_string(),
                rating_key: "12345".to_string(),
                last_viewed_at: Some(0),
            },
            seasons: vec![PlexSeason {
                rating_key: "12345".to_string(),
                parent_title: "Mysterious Girlfriend X".to_string(),
                index: 1,
                episodes: 13,
                parent_year: Some(2012),
                watched_episodes: 0,
                last_viewed_at: Some(0),
            }],
        };

        let result = mapper
            .find_mapping(series)
            .await
            .expect("Faied to get result for one to one mapping");

        assert_eq!(1, result.len());
        assert_eq!("12467".to_string(), result[0].anime_list_id)
    }

    #[tokio::test]
    async fn two_season_mapping() {
        let mapper = init().await;

        let series = SeriesWithSeason {
            series: PlexSeries {
                rating_key: "12794".to_string(),
                title: "Vinland Saga".to_string(),
                last_viewed_at: Some(1687277617),
            },
            seasons: vec![
                PlexSeason {
                    rating_key: "12795".to_string(),
                    index: 1,
                    parent_title: "Vinland Saga".to_string(),
                    parent_year: None,
                    watched_episodes: 24,
                    episodes: 24,
                    last_viewed_at: Some(1682194762),
                },
                PlexSeason {
                    rating_key: "45711".to_string(),
                    index: 2,
                    parent_title: "Vinland Saga".to_string(),
                    parent_year: Some(2019),
                    watched_episodes: 24,
                    episodes: 24,
                    last_viewed_at: Some(1687277617),
                },
            ],
        };

        let result = mapper
            .find_mapping(series)
            .await
            .expect("Faied to get result for two season mapping");

        assert_eq!(2, result.len());
        assert_eq!("101348".to_string(), result[0].anime_list_id);
        assert_eq!("136430".to_string(), result[1].anime_list_id);
    }

    #[tokio::test]
    async fn overlord_series_with_difficult_name() {
        let mapper = init().await;

        let series = SeriesWithSeason {
            series: PlexSeries {
                rating_key: "10618".to_string(),
                title: "Overlord".to_string(),
                last_viewed_at: Some(1659087323),
            },
            seasons: vec![
                PlexSeason {
                    rating_key: "29790".to_string(),
                    index: 0,
                    parent_title: "Overlord".to_string(),
                    parent_year: None,
                    watched_episodes: 13,
                    episodes: 37,
                    last_viewed_at: None,
                },
                PlexSeason {
                    rating_key: "10619".to_string(),
                    index: 1,
                    parent_title: "Overlord".to_string(),
                    parent_year: None,
                    watched_episodes: 13,
                    episodes: 13,
                    last_viewed_at: Some(1656434507),
                },
                PlexSeason {
                    rating_key: "10647".to_string(),
                    index: 2,
                    parent_title: "Overlord".to_string(),
                    parent_year: None,
                    watched_episodes: 13,
                    episodes: 13,
                    last_viewed_at: Some(1659087323),
                },
                PlexSeason {
                    rating_key: "10663".to_string(),
                    index: 3,
                    parent_title: "Overlord".to_string(),
                    parent_year: None,
                    watched_episodes: 13,
                    episodes: 13,
                    last_viewed_at: Some(1609995008),
                },
                PlexSeason {
                    rating_key: "43158".to_string(),
                    index: 4,
                    parent_title: "Overlord".to_string(),
                    parent_year: Some(2015),
                    watched_episodes: 0,
                    episodes: 13,
                    last_viewed_at: None,
                },
            ],
        };

        let result = mapper
            .find_mapping(series)
            .await
            .expect("Faied to get result for complex name mapping");

        assert_eq!(4, result.len());
        assert_eq!("20832".to_string(), result[0].anime_list_id);
        assert_eq!("98437".to_string(), result[1].anime_list_id);
        assert_eq!("101474".to_string(), result[2].anime_list_id);
        assert_eq!("133844".to_string(), result[3].anime_list_id);
    }

    #[tokio::test]
    async fn series_with_two_anilist_entries_for_one_plex_season() {
        let mapper = init().await;

        let series = SeriesWithSeason {
            series: PlexSeries {
                rating_key: "17456".to_string(),
                title: "Attack on Titan".to_string(),
                last_viewed_at: Some(1682194387),
            },
            seasons: vec![
                PlexSeason {
                    rating_key: "30037".to_string(),
                    index: 0,
                    parent_title: "Attack on Titan".to_string(),
                    parent_year: None,
                    watched_episodes: 8,
                    episodes: 8,
                    last_viewed_at: None,
                },
                PlexSeason {
                    rating_key: "17457".to_string(),
                    index: 1,
                    parent_title: "Attack on Titan".to_string(),
                    parent_year: None,
                    watched_episodes: 25,
                    episodes: 25,
                    last_viewed_at: Some(1682194387),
                },
                PlexSeason {
                    rating_key: "17483".to_string(),
                    index: 2,
                    parent_title: "Attack on Titan".to_string(),
                    parent_year: None,
                    watched_episodes: 0,
                    episodes: 12,
                    last_viewed_at: Some(1639331438),
                },
                PlexSeason {
                    rating_key: "17496".to_string(),
                    index: 3,
                    parent_title: "Attack on Titan".to_string(),
                    parent_year: None,
                    watched_episodes: 0,
                    episodes: 22,
                    last_viewed_at: Some(1602543627),
                },
                PlexSeason {
                    rating_key: "22191".to_string(),
                    index: 4,
                    parent_title: "Attack on Titan".to_string(),
                    parent_year: None,
                    watched_episodes: 29,
                    episodes: 29,
                    last_viewed_at: Some(1678324708),
                },
            ],
        };

        let result = mapper
            .find_mapping(series)
            .await
            .expect("Faied to get result for complex name mapping");

        // println!("{:?}", result);
        assert_eq!(7, result.len());
        assert_eq!("16498".to_string(), result[0].anime_list_id);
        assert_eq!("20958".to_string(), result[1].anime_list_id);
        assert_eq!("99147".to_string(), result[2].anime_list_id);
        assert_eq!("104578".to_string(), result[3].anime_list_id);
        assert_eq!("110277".to_string(), result[4].anime_list_id);
        assert_eq!("131681".to_string(), result[5].anime_list_id);
        assert_eq!("146984".to_string(), result[6].anime_list_id);
    }

    // Write test for way of the house husband because of an indexing error
}
