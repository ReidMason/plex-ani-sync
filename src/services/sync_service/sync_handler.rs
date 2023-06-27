use std::cmp::min;

use chrono::{Duration, Utc};

use crate::services::{
    anime_list_service::anime_list_service::AnimeResult, dbstore::sqlite::Mapping,
};

struct AnimeEntryPlexRepresentation {
    anime_result: AnimeResult,
    plex_episodes: Vec<PlexEpisode>,
}

struct PlexSeries {
    seasons: Vec<PlexSeason>,
}

struct PlexSeason {
    rating_key: String,
    episodes: Vec<PlexEpisode>,
}

#[derive(Clone)]
struct PlexEpisode {
    id: u32,
    last_viewed_at: Option<i64>,
}

fn get_plex_episodes_for_anime_list_id(
    all_plex_series: &Vec<PlexSeries>,
    all_mappings: &Vec<Mapping>,
    anime_result: AnimeResult,
) -> AnimeEntryPlexRepresentation {
    let mut plex_episodes: Vec<PlexEpisode> = vec![];
    let relevant_mappings: Vec<&Mapping> = all_mappings
        .into_iter()
        .filter(|x| x.anime_list_id == anime_result.id.to_string())
        .collect();

    for mapping in relevant_mappings {
        for series in all_plex_series {
            for season in series.seasons.iter() {
                if season.rating_key != mapping.plex_id {
                    continue;
                }

                let start = usize::try_from(mapping.episode_start - 1).unwrap();
                let end = min(
                    usize::try_from(mapping.episode_start + mapping.season_length - 1).unwrap(),
                    season.episodes.len(),
                );

                for i in start..end {
                    let index = usize::try_from(i).expect("Failed to convert index");
                    plex_episodes.push(season.episodes[index].clone());
                }
            }
        }
    }

    return AnimeEntryPlexRepresentation {
        anime_result,
        plex_episodes,
    };
}

#[derive(Debug, PartialEq)]
enum WatchStatus {
    Complete,
    Watching,
    Paused,
    Dropped,
    Planning,
}

struct AnimeListWatchStatus {
    anime_list_id: String,
    watch_status: WatchStatus,
    episodes_watched: u16,
}

fn get_watch_status(
    anime_entry_representation: AnimeEntryPlexRepresentation,
) -> AnimeListWatchStatus {
    let episodes_watched = anime_entry_representation
        .plex_episodes
        .iter()
        .filter(|x| x.last_viewed_at.is_some())
        .count();
    let episodes_watched: u16 = u16::try_from(episodes_watched).unwrap();

    if episodes_watched == 0 {
        return AnimeListWatchStatus {
            anime_list_id: anime_entry_representation.anime_result.id.to_string(),
            watch_status: WatchStatus::Planning,
            episodes_watched,
        };
    }

    let total_episodes = anime_entry_representation.anime_result.episodes.unwrap();
    if total_episodes == episodes_watched {
        return AnimeListWatchStatus {
            anime_list_id: anime_entry_representation.anime_result.id.to_string(),
            watch_status: WatchStatus::Complete,
            episodes_watched,
        };
    }

    let last_viewed_at = anime_entry_representation
        .plex_episodes
        .into_iter()
        .map(|x| x.last_viewed_at)
        .max()
        .unwrap()
        .unwrap();

    let dropped_threshold = Utc::now() - Duration::days(30);
    if episodes_watched > 0 && last_viewed_at <= dropped_threshold.timestamp() {
        return AnimeListWatchStatus {
            anime_list_id: anime_entry_representation.anime_result.id.to_string(),
            watch_status: WatchStatus::Dropped,
            episodes_watched,
        };
    }

    let paused_threshold = Utc::now() - Duration::days(14);
    if episodes_watched > 0 && last_viewed_at <= paused_threshold.timestamp() {
        return AnimeListWatchStatus {
            anime_list_id: anime_entry_representation.anime_result.id.to_string(),
            watch_status: WatchStatus::Paused,
            episodes_watched,
        };
    }

    if episodes_watched > 0 && episodes_watched < total_episodes {
        return AnimeListWatchStatus {
            anime_list_id: anime_entry_representation.anime_result.id.to_string(),
            watch_status: WatchStatus::Watching,
            episodes_watched,
        };
    }

    return AnimeListWatchStatus {
        anime_list_id: anime_entry_representation.anime_result.id.to_string(),
        watch_status: WatchStatus::Planning,
        episodes_watched,
    };
}

#[cfg(test)]
mod tests {
    use chrono::{Duration, Utc};

    use crate::services::anime_list_service::anime_list_service::{
        Date, MediaStatus, Relations, Title,
    };

    use super::*;

    const ANIME_RESULT: AnimeResult = AnimeResult {
        id: 16498,
        format: None,
        episodes: Some(3),
        synonyms: vec![],
        status: MediaStatus::Finished,
        end_date: Date {
            year: None,
            month: None,
            day: None,
        },
        start_date: Date {
            year: None,
            month: None,
            day: None,
        },
        title: Title {
            english: None,
            romaji: String::new(),
        },
        relations: Relations {
            edges: vec![],
            nodes: vec![],
        },
    };

    #[test]
    fn test_get_watch_status_complete() {
        let anime_entry_representation = AnimeEntryPlexRepresentation {
            anime_result: ANIME_RESULT,
            plex_episodes: vec![
                PlexEpisode {
                    id: 1,
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    id: 2,
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    id: 3,
                    last_viewed_at: Some(12345),
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(WatchStatus::Complete, result.watch_status);
    }

    #[test]
    fn test_get_watch_status_planning() {
        let anime_entry_representation = AnimeEntryPlexRepresentation {
            anime_result: ANIME_RESULT,
            plex_episodes: vec![
                PlexEpisode {
                    id: 1,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    id: 2,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    id: 3,
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(WatchStatus::Planning, result.watch_status);
    }

    #[test]
    fn test_get_watch_status_dropped() {
        let a_month_ago = Utc::now() - Duration::days(30);

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            anime_result: ANIME_RESULT,
            plex_episodes: vec![
                PlexEpisode {
                    id: 1,
                    last_viewed_at: Some(a_month_ago.timestamp()),
                },
                PlexEpisode {
                    id: 2,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    id: 3,
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(WatchStatus::Dropped, result.watch_status);
    }

    #[test]
    fn test_get_watch_status_paused() {
        let now = Utc::now();
        let two_weeks_ago = now - Duration::days(14);
        let a_month_ago = now - Duration::days(30);

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            anime_result: ANIME_RESULT,
            plex_episodes: vec![
                PlexEpisode {
                    id: 1,
                    last_viewed_at: Some(two_weeks_ago.timestamp()),
                },
                PlexEpisode {
                    id: 2,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    id: 3,
                    last_viewed_at: Some(a_month_ago.timestamp()),
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(WatchStatus::Paused, result.watch_status);
    }

    #[test]
    fn test_get_watch_status_watching() {
        let now = Utc::now();

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            anime_result: ANIME_RESULT,
            plex_episodes: vec![
                PlexEpisode {
                    id: 1,
                    last_viewed_at: Some(now.timestamp()),
                },
                PlexEpisode {
                    id: 2,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    id: 3,
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(WatchStatus::Watching, result.watch_status);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_multiple_mappings_across_multiple_plex_seasons() {
        let all_plex_series = vec![PlexSeries {
            seasons: vec![
                PlexSeason {
                    rating_key: "17457".to_string(),
                    episodes: vec![
                        PlexEpisode {
                            id: 1,
                            last_viewed_at: Some(12345),
                        },
                        PlexEpisode {
                            id: 2,
                            last_viewed_at: Some(12345),
                        },
                    ],
                },
                PlexSeason {
                    rating_key: "12345".to_string(),
                    episodes: vec![
                        PlexEpisode {
                            id: 3,
                            last_viewed_at: Some(12345),
                        },
                        PlexEpisode {
                            id: 4,
                            last_viewed_at: Some(12345),
                        },
                    ],
                },
            ],
        }];
        let all_mappings = vec![
            Mapping {
                id: 1,
                list_provider_id: 1,
                plex_id: "17457".to_string(),
                plex_series_id: "".to_string(),
                plex_episode_start: 1,
                season_length: 1,
                anime_list_id: "16498".to_string(),
                episode_start: 1,
                enabled: true,
                ignored: false,
            },
            Mapping {
                id: 2,
                list_provider_id: 1,
                plex_id: "12345".to_string(),
                plex_series_id: "".to_string(),
                plex_episode_start: 1,
                season_length: 2,
                anime_list_id: "16498".to_string(),
                episode_start: 1,
                enabled: true,
                ignored: false,
            },
        ];

        let result =
            get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, ANIME_RESULT);

        assert_eq!(3, result.plex_episodes.len());
        assert_eq!(1, result.plex_episodes[0].id);
        assert_eq!(3, result.plex_episodes[1].id);
        assert_eq!(4, result.plex_episodes[2].id);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id() {
        let all_plex_series = vec![PlexSeries {
            seasons: vec![PlexSeason {
                rating_key: "17457".to_string(),
                episodes: vec![
                    PlexEpisode {
                        id: 1,
                        last_viewed_at: Some(12345),
                    },
                    PlexEpisode {
                        id: 2,
                        last_viewed_at: Some(12345),
                    },
                ],
            }],
        }];
        let all_mappings = vec![Mapping {
            id: 1,
            list_provider_id: 1,
            plex_id: "17457".to_string(),
            plex_series_id: "17456".to_string(),
            plex_episode_start: 1,
            season_length: 2,
            anime_list_id: "16498".to_string(),
            episode_start: 1,
            enabled: true,
            ignored: false,
        }];
        let result =
            get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, ANIME_RESULT);

        assert_eq!(2, result.plex_episodes.len());
        assert_eq!(1, result.plex_episodes[0].id);
        assert_eq!(2, result.plex_episodes[1].id);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_when_plex_season_has_more_episodes_than_mapping() {
        let all_plex_series = vec![PlexSeries {
            seasons: vec![PlexSeason {
                rating_key: "17457".to_string(),
                episodes: vec![
                    PlexEpisode {
                        id: 1,
                        last_viewed_at: Some(12345),
                    },
                    PlexEpisode {
                        id: 2,
                        last_viewed_at: Some(12345),
                    },
                ],
            }],
        }];
        let all_mappings = vec![Mapping {
            id: 1,
            list_provider_id: 1,
            plex_id: "17457".to_string(),
            plex_series_id: "17456".to_string(),
            plex_episode_start: 1,
            season_length: 1,
            anime_list_id: "16498".to_string(),
            episode_start: 1,
            enabled: true,
            ignored: false,
        }];

        let result =
            get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, ANIME_RESULT);

        assert_eq!(1, result.plex_episodes.len());
        assert_eq!(1, result.plex_episodes[0].id);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_when_plex_season_has_less_episodes_than_mapping() {
        let all_plex_series = vec![PlexSeries {
            seasons: vec![PlexSeason {
                rating_key: "17457".to_string(),
                episodes: vec![PlexEpisode {
                    id: 1,
                    last_viewed_at: Some(12345),
                }],
            }],
        }];
        let all_mappings = vec![Mapping {
            id: 1,
            list_provider_id: 1,
            plex_id: "17457".to_string(),
            plex_series_id: "17456".to_string(),
            plex_episode_start: 1,
            season_length: 6,
            anime_list_id: "16498".to_string(),
            episode_start: 1,
            enabled: true,
            ignored: false,
        }];

        let result =
            get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, ANIME_RESULT);

        assert_eq!(1, result.plex_episodes.len());
    }
}
