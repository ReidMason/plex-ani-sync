use async_trait::async_trait;
use log::error;
use std::vec;

use crate::services::anime_list_service::anime_list_service::{AnimeListService, AnimeResult};
use crate::services::dbstore::dbstore::DbStore;
use crate::services::dbstore::sqlite::Mapping;
use crate::services::plex::plex_api::{PlexSeason, PlexSeries, ResponsePlexSeason};

#[async_trait]
pub trait MappingHandlerInterface {
    async fn create_mapping(&self, series: &PlexSeries) -> Result<Vec<Mapping>, anyhow::Error>;
    async fn find_match_for_season(
        &self,
        season: &PlexSeason,
    ) -> Result<Option<AnimeResult>, anyhow::Error>;
    async fn get_all_relevant_mappings(
        &self,
        all_series: &Vec<PlexSeries>,
    ) -> Vec<MappingWithListData>;
    async fn get_all_mappings(&self) -> Vec<Mapping>;
}

pub struct MappingHandler<T, J>
where
    T: AnimeListService,
    J: DbStore,
{
    anime_list_service: T,
    db_store: J,
}

impl<T, J> MappingHandler<T, J>
where
    T: AnimeListService,
    J: DbStore,
{
    pub fn new(anime_list_service: T, db_store: J) -> Self {
        Self {
            anime_list_service,
            db_store,
        }
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
        let potential_match_episodes: usize = potential_match.result.episodes.unwrap_or(0).into();
        if potential_match_episodes == target.episodes.len() {
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
                if compare_strings(english_title, potential_title) {
                    potential_match.score += 50;
                }
            }

            if compare_strings(&potential_match.result.title.romaji, potential_title) {
                potential_match.score += 50;
            }

            for synonym in potential_match.result.synonyms.clone() {
                if compare_strings(&synonym, potential_title) {
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
    potential_matches.retain(|x| x.score > min_score);

    match potential_matches.pop() {
        Some(x) => Some(x.result),
        None => None,
    }
}

fn get_mapped_episode_count(mappings: &[Mapping], rating_key: &str) -> u32 {
    mappings
        .iter()
        .filter_map(|x| {
            if x.plex_id == rating_key {
                return Some(x.season_length);
            }
            None
        })
        .sum::<u32>()
}

fn get_prev_mapping(mappings: &[Mapping], season: &PlexSeason) -> Option<Mapping> {
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

#[derive(Debug)]
pub struct MappingWithListData {
    pub mapping: Mapping,
    pub anilist_series: AnimeResult,
}

#[async_trait]
impl<T, J> MappingHandlerInterface for MappingHandler<T, J>
where
    T: AnimeListService + Sync + Send,
    J: DbStore,
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

    async fn get_all_relevant_mappings(
        &self,
        all_series: &Vec<PlexSeries>,
    ) -> Vec<MappingWithListData> {
        let mut mappings: Vec<MappingWithListData> = vec![];
        for series in all_series {
            let series_mappings = self
                .db_store
                .get_mapping_for_series(&series.rating_key)
                .await
                .unwrap();

            for mapping in series_mappings {
                let anilist_series = self
                    .anime_list_service
                    .get_anime(&mapping.anime_list_id)
                    .await;
                let anilist_series = match anilist_series {
                    Ok(x) => x,
                    Err(_) => {
                        error!("Failed to request anilist entry");
                        continue;
                    }
                };

                let anilist_series = match anilist_series {
                    Some(x) => x,
                    None => {
                        error!("Failed to find anilist entry for mapping");
                        continue;
                    }
                };

                let mapping_with_data = MappingWithListData {
                    mapping,
                    anilist_series,
                };
                mappings.push(mapping_with_data);
            }
        }

        mappings
    }

    async fn get_all_mappings(&self) -> Vec<Mapping> {
        let result = self.db_store.get_all_mappings().await;
        match result {
            Ok(x) => x,
            Err(_) => vec![],
        }
    }

    async fn create_mapping(&self, series: &PlexSeries) -> Result<Vec<Mapping>, anyhow::Error> {
        // TODO: Reduce the chance of mapping errors by building up a vec of mappings for one
        // season then only push them if all the episodes are covered

        // Load any existing mappings
        let mut mappings = self
            .db_store
            .get_mapping_for_series(&series.rating_key)
            .await?;

        // Just skip big series for now
        if series.seasons.len() > 6 {
            return Ok(mappings);
        }

        for (i, season) in series.seasons.iter().enumerate() {
            // let mappings = match result {
            //     Ok(x) => x,
            //     Err(e) => {
            //         error!(
            //             "Failed to get existing mappings for {} season {}",
            //             season.parent_title, season.index
            //         );
            //         return Err(e).into();
            //     }
            // };

            let is_specials_season = season.index == 0;
            if is_specials_season {
                continue;
            }

            // We start by just mapping the first season
            let is_first_season = season.index == 1;
            if is_first_season
                && get_mapped_episode_count(&mappings, &season.rating_key)
                    < season.episodes.len().try_into().unwrap()
            {
                let found_match = self.find_match_for_season(season).await?;
                let found_match = match found_match {
                    Some(x) => x,
                    None => return Ok(mappings),
                };

                let mapping = Mapping {
                    id: 0,
                    list_provider_id: 1,
                    plex_id: season.rating_key.clone(),
                    plex_series_id: series.rating_key.clone(),
                    plex_episode_start: 1,
                    season_length: found_match
                        .episodes
                        .unwrap_or(season.episodes.len().try_into().unwrap())
                        .into(),
                    anime_list_id: found_match.id.to_string(),
                    episode_start: 1,
                    enabled: true,
                    ignored: false,
                };
                mappings.push(mapping);
            }

            if !mappings.is_empty() {
                let limit = 5;
                let mut counter = 0;

                while counter < limit
                    && get_mapped_episode_count(&mappings, &season.rating_key)
                        < season.episodes.len().try_into().unwrap()
                {
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
                    // TODO: Don't just use 0 if the episode number isn't known This likely means it's still releasing but we need to check
                    if current_mapped_episodes + u32::from(sequel.episodes.unwrap_or(0))
                        != season.get_episode_count()
                    {
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
                        id: 0,
                        list_provider_id: 1,
                        plex_id: season.rating_key.clone(),
                        plex_series_id: series.rating_key.clone(),
                        plex_episode_start,
                        season_length: sequel
                            .episodes
                            .unwrap_or(season.episodes.len().try_into().unwrap())
                            .into(),
                        anime_list_id: sequel.id.to_string(),
                        episode_start: 1,
                        enabled: true,
                        ignored: false,
                    };
                    // println!("Added mapping: {:?}", mapping);
                    mappings.push(mapping);
                }
            }
        }

        let new_mappings = mappings.iter().filter(|x| x.id == 0);
        for mapping in new_mappings {
            let _ = self.db_store.save_mapping(mapping).await;
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
            plex::plex_api::{PlexEpisode, ResponsePlexSeason, ResponsePlexSeries},
        },
        utils::{get_db_file_location, init_logger},
    };

    use super::*;

    async fn init() -> MappingHandler<AnilistService<ConfigService, Sqlite>, Sqlite> {
        init_logger();

        let mut db_store = Sqlite::new(&get_db_file_location()).await;
        db_store.migrate().await;

        let config = db_store.get_config().await;
        let config_service = ConfigService::new(config);

        let list_service = AnilistService::new(config_service, db_store, None);

        let db_store = Sqlite::new(&get_db_file_location()).await;
        MappingHandler::new(list_service, db_store)
    }

    fn generate_episodes(num_episodes: u16) -> Vec<PlexEpisode> {
        let mut episodes: Vec<PlexEpisode> = vec![];

        for i in 1..=num_episodes {
            episodes.push(PlexEpisode {
                rating_key: i.to_string(),
                view_count: 0,
                last_viewed_at: None,
            })
        }

        return episodes;
    }

    #[tokio::test]
    async fn test_one_to_one_mapping() {
        let mapper = init().await;

        let series = PlexSeries {
            rating_key: "12345".to_string(),
            seasons: vec![PlexSeason {
                rating_key: "12345".to_string(),
                parent_title: "Mysterious Girlfriend X".to_string(),
                index: 1,
                episodes: generate_episodes(13),
            }],
        };

        let result = mapper
            .create_mapping(&series)
            .await
            .expect("Faied to get result for one to one mapping");

        println!("{}", series.seasons[0].episodes.len());
        for res in result.iter() {
            println!("{}", res.anime_list_id);
        }

        assert_eq!(1, result.len());
        assert_eq!("12467".to_string(), result[0].anime_list_id)
    }

    #[tokio::test]
    async fn two_season_mapping() {
        let mapper = init().await;

        let series = PlexSeries {
            rating_key: "12794".to_string(),
            seasons: vec![
                PlexSeason {
                    rating_key: "12795".to_string(),
                    parent_title: "Vinland Saga".to_string(),
                    index: 1,
                    episodes: generate_episodes(24),
                },
                PlexSeason {
                    rating_key: "45711".to_string(),
                    parent_title: "Vinland Saga".to_string(),
                    index: 2,
                    episodes: generate_episodes(24),
                },
            ],
        };

        let result = mapper
            .create_mapping(&series)
            .await
            .expect("Faied to get result for two season mapping");

        assert_eq!(2, result.len());
        assert_eq!("101348".to_string(), result[0].anime_list_id);
        assert_eq!("136430".to_string(), result[1].anime_list_id);
    }

    #[tokio::test]
    async fn overlord_series_with_difficult_name() {
        let mapper = init().await;

        let series = PlexSeries {
            rating_key: "10618".to_string(),
            seasons: vec![
                PlexSeason {
                    rating_key: "29790".to_string(),
                    index: 0,
                    parent_title: "Overlord".to_string(),
                    episodes: generate_episodes(37),
                },
                PlexSeason {
                    rating_key: "10619".to_string(),
                    index: 1,
                    parent_title: "Overlord".to_string(),
                    episodes: generate_episodes(13),
                },
                PlexSeason {
                    rating_key: "10647".to_string(),
                    index: 2,
                    parent_title: "Overlord".to_string(),
                    episodes: generate_episodes(13),
                },
                PlexSeason {
                    rating_key: "10663".to_string(),
                    index: 3,
                    parent_title: "Overlord".to_string(),
                    episodes: generate_episodes(13),
                },
                PlexSeason {
                    rating_key: "43158".to_string(),
                    index: 4,
                    parent_title: "Overlord".to_string(),
                    episodes: generate_episodes(13),
                },
            ],
        };

        let result = mapper
            .create_mapping(&series)
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

        let series = PlexSeries {
            rating_key: "17456".to_string(),
            seasons: vec![
                PlexSeason {
                    rating_key: "30037".to_string(),
                    index: 0,
                    parent_title: "Attack on Titan".to_string(),
                    episodes: generate_episodes(8),
                },
                PlexSeason {
                    rating_key: "17457".to_string(),
                    index: 1,
                    parent_title: "Attack on Titan".to_string(),
                    episodes: generate_episodes(25),
                },
                PlexSeason {
                    rating_key: "17483".to_string(),
                    index: 2,
                    parent_title: "Attack on Titan".to_string(),
                    episodes: generate_episodes(12),
                },
                PlexSeason {
                    rating_key: "17496".to_string(),
                    index: 3,
                    parent_title: "Attack on Titan".to_string(),
                    episodes: generate_episodes(22),
                },
                PlexSeason {
                    rating_key: "22191".to_string(),
                    index: 4,
                    parent_title: "Attack on Titan".to_string(),
                    episodes: generate_episodes(29),
                },
            ],
        };

        let result = mapper
            .create_mapping(&series)
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

    // TODO: Write test for way of the house husband because of an indexing error
    // TODO: Add test for maken ki
}
