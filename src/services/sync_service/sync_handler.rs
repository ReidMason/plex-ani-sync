use std::cmp::min;

use chrono::{Duration, Utc};

use crate::services::{
    anime_list_service::anime_list_service::{AnilistWatchStatus, AnimeListEntry},
    dbstore::sqlite::Mapping,
    plex::plex_api::{PlexEpisode, PlexSeries},
};

pub fn plex_series_to_animelist_entry(
    plex_anime_entry: AnimeEntryPlexRepresentation,
) -> AnimeListEntry {
    let watched_episodes = plex_anime_entry
        .plex_episodes
        .iter()
        .filter(|x| x.last_viewed_at.is_some())
        .count() as u16;

    AnimeListEntry {
        media_id: plex_anime_entry.anime_list_id,
        status: get_watch_status(plex_anime_entry),
        progress: watched_episodes,
    }
}

pub struct AnimeEntryPlexRepresentation {
    anime_list_id: u32,
    episodes: Option<u16>,
    plex_episodes: Vec<PlexEpisode>,
}

pub fn get_plex_episodes_for_anime_list_id(
    all_plex_series: &Vec<PlexSeries>,
    all_mappings: &Vec<Mapping>,
    anime_list_id: u32,
) -> AnimeEntryPlexRepresentation {
    let mut plex_episodes: Vec<PlexEpisode> = vec![];
    let relevant_mappings: Vec<&Mapping> = all_mappings
        .into_iter()
        .filter(|x| x.anime_list_id == anime_list_id)
        .collect();

    for mapping in relevant_mappings.iter() {
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

                let mut selected_episodes: Vec<PlexEpisode> = season
                    .episodes
                    .iter()
                    .enumerate()
                    .filter(|(i, _)| i >= &start && i < &end)
                    .map(|(_, x)| x.clone())
                    .collect();
                plex_episodes.append(&mut selected_episodes);
            }
        }
    }

    let episodes = match relevant_mappings.get(0) {
        Some(x) => x.episodes,
        None => None,
    };

    return AnimeEntryPlexRepresentation {
        plex_episodes,
        episodes,
        anime_list_id,
    };
}

fn get_watch_status(
    anime_entry_representation: AnimeEntryPlexRepresentation,
) -> AnilistWatchStatus {
    let episodes_watched = anime_entry_representation
        .plex_episodes
        .iter()
        .filter(|x| x.view_count > 0)
        .count();
    let episodes_watched: u16 = u16::try_from(episodes_watched).unwrap();

    let total_episodes = anime_entry_representation.episodes;

    if total_episodes == Some(episodes_watched) {
        return AnilistWatchStatus::Completed;
    }

    let last_viewed_at = anime_entry_representation
        .plex_episodes
        .into_iter()
        .map(|x| x.last_viewed_at)
        .max()
        .unwrap_or(None);

    let dropped_threshold = Utc::now() - Duration::days(30);
    if episodes_watched > 0
        && last_viewed_at.is_some()
        && last_viewed_at.unwrap() <= dropped_threshold.timestamp()
    {
        return AnilistWatchStatus::Dropped;
    }

    let paused_threshold = Utc::now() - Duration::days(14);
    if episodes_watched > 0
        && last_viewed_at.is_some()
        && last_viewed_at.unwrap() <= paused_threshold.timestamp()
    {
        return AnilistWatchStatus::Paused;
    }

    if episodes_watched > 0 && (total_episodes.is_none() || Some(episodes_watched) < total_episodes)
    {
        return AnilistWatchStatus::Current;
    }

    return AnilistWatchStatus::Planning;
}

#[cfg(test)]
mod tests {
    use crate::services::plex::plex_api::PlexSeason;
    use chrono::{Duration, Utc};

    use super::*;

    #[test]
    fn test_anime_list_entry_equality_when_not_equal_media_id() {
        let current = AnimeListEntry {
            media_id: 1234567,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        let new = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        assert!(current != new);
    }

    #[test]
    fn test_anime_list_entry_equality_when_not_equal_status() {
        let current = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Planning,
        };

        let new = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        assert!(current != new);
    }

    #[test]
    fn test_anime_list_entry_equality_when_not_equal_progress() {
        let current = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        let new = AnimeListEntry {
            media_id: 16498,
            progress: 4,
            status: AnilistWatchStatus::Completed,
        };

        assert!(current != new);
    }

    #[test]
    fn test_anime_list_entry_equality_when_equal() {
        let current = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        let new = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };

        assert!(current == new);
    }

    #[test]
    fn test_plex_series_to_animelist_entry() {
        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 16498,
            plex_episodes: vec![
                PlexEpisode {
                    view_count: 1,
                    rating_key: "1".to_string(),
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    view_count: 1,
                    rating_key: "2".to_string(),
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    view_count: 1,
                    rating_key: "3".to_string(),
                    last_viewed_at: Some(12345),
                },
            ],
        };

        let expected = AnimeListEntry {
            media_id: 16498,
            progress: 3,
            status: AnilistWatchStatus::Completed,
        };
        let result = plex_series_to_animelist_entry(anime_entry_representation);

        assert_eq!(expected, result);
    }

    #[test]
    fn test_get_watch_status_complete() {
        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 6789,
            plex_episodes: vec![
                PlexEpisode {
                    view_count: 1,
                    rating_key: "1".to_string(),
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    view_count: 1,
                    rating_key: "2".to_string(),
                    last_viewed_at: Some(12345),
                },
                PlexEpisode {
                    view_count: 1,
                    rating_key: "3".to_string(),
                    last_viewed_at: Some(12345),
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(AnilistWatchStatus::Completed, result);
    }

    #[test]
    fn test_get_watch_status_planning() {
        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 6789,
            plex_episodes: vec![
                PlexEpisode {
                    view_count: 0,
                    rating_key: "1".to_string(),
                    last_viewed_at: None,
                },
                PlexEpisode {
                    view_count: 0,
                    rating_key: "2".to_string(),
                    last_viewed_at: None,
                },
                PlexEpisode {
                    view_count: 0,
                    rating_key: "3".to_string(),
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(AnilistWatchStatus::Planning, result);
    }

    #[test]
    fn test_get_watch_status_dropped() {
        let a_month_ago = Utc::now() - Duration::days(30);

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 6789,
            plex_episodes: vec![
                PlexEpisode {
                    view_count: 1,
                    rating_key: "1".to_string(),
                    last_viewed_at: Some(a_month_ago.timestamp()),
                },
                PlexEpisode {
                    view_count: 0,
                    rating_key: "2".to_string(),
                    last_viewed_at: None,
                },
                PlexEpisode {
                    view_count: 0,
                    rating_key: "3".to_string(),
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(AnilistWatchStatus::Dropped, result);
    }

    #[test]
    fn test_get_watch_status_paused() {
        let now = Utc::now();
        let two_weeks_ago = now - Duration::days(14);
        let a_month_ago = now - Duration::days(30);

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 6789,
            plex_episodes: vec![
                PlexEpisode {
                    rating_key: "1".to_string(),
                    view_count: 1,
                    last_viewed_at: Some(two_weeks_ago.timestamp()),
                },
                PlexEpisode {
                    rating_key: "2".to_string(),
                    view_count: 0,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    rating_key: "3".to_string(),
                    view_count: 1,
                    last_viewed_at: Some(a_month_ago.timestamp()),
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(AnilistWatchStatus::Paused, result);
    }

    #[test]
    fn test_get_watch_status_watching() {
        let now = Utc::now();

        let anime_entry_representation = AnimeEntryPlexRepresentation {
            episodes: Some(3),
            anime_list_id: 6789,
            plex_episodes: vec![
                PlexEpisode {
                    rating_key: "1".to_string(),
                    view_count: 1,
                    last_viewed_at: Some(now.timestamp()),
                },
                PlexEpisode {
                    rating_key: "2".to_string(),
                    view_count: 0,
                    last_viewed_at: None,
                },
                PlexEpisode {
                    rating_key: "3".to_string(),
                    view_count: 0,
                    last_viewed_at: None,
                },
            ],
        };
        let result = get_watch_status(anime_entry_representation);

        assert_eq!(AnilistWatchStatus::Current, result);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_multiple_mappings_across_multiple_plex_seasons() {
        let all_plex_series = vec![PlexSeries {
            title: "".to_string(),
            rating_key: "1234".to_string(),
            seasons: vec![
                PlexSeason {
                    rating_key: "17457".to_string(),
                    index: 1,
                    parent_title: "".to_string(),
                    episodes: vec![
                        PlexEpisode {
                            rating_key: "1".to_string(),
                            view_count: 1,
                            last_viewed_at: Some(12345),
                        },
                        PlexEpisode {
                            rating_key: "2".to_string(),
                            view_count: 1,
                            last_viewed_at: Some(12345),
                        },
                    ],
                },
                PlexSeason {
                    rating_key: "12345".to_string(),
                    index: 2,
                    parent_title: "".to_string(),
                    episodes: vec![
                        PlexEpisode {
                            rating_key: "3".to_string(),
                            view_count: 1,
                            last_viewed_at: Some(12345),
                        },
                        PlexEpisode {
                            rating_key: "4".to_string(),
                            view_count: 1,
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
                anime_list_id: 16498,
                episode_start: 1,
                enabled: true,
                ignored: false,
                episodes: Some(2),
            },
            Mapping {
                id: 2,
                list_provider_id: 1,
                plex_id: "12345".to_string(),
                plex_series_id: "".to_string(),
                plex_episode_start: 1,
                season_length: 2,
                anime_list_id: 16498,
                episode_start: 1,
                enabled: true,
                ignored: false,
                episodes: Some(2),
            },
        ];

        let result = get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, 16498);

        assert_eq!(3, result.plex_episodes.len());
        assert_eq!("1", &result.plex_episodes[0].rating_key);
        assert_eq!("3", &result.plex_episodes[1].rating_key);
        assert_eq!("4", &result.plex_episodes[2].rating_key);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id() {
        let all_plex_series = vec![PlexSeries {
            title: "".to_string(),
            rating_key: "1234".to_string(),
            seasons: vec![PlexSeason {
                rating_key: "17457".to_string(),
                index: 1,
                parent_title: "".to_string(),
                episodes: vec![
                    PlexEpisode {
                        rating_key: "1".to_string(),
                        view_count: 1,
                        last_viewed_at: Some(12345),
                    },
                    PlexEpisode {
                        rating_key: "2".to_string(),
                        view_count: 1,
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
            anime_list_id: 16498,
            episode_start: 1,
            enabled: true,
            ignored: false,
            episodes: Some(2),
        }];
        let result = get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, 16498);

        assert_eq!(2, result.plex_episodes.len());
        assert_eq!("1", &result.plex_episodes[0].rating_key);
        assert_eq!("2", &result.plex_episodes[1].rating_key);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_when_plex_season_has_more_episodes_than_mapping() {
        let all_plex_series = vec![PlexSeries {
            title: "".to_string(),
            rating_key: "1234".to_string(),
            seasons: vec![PlexSeason {
                rating_key: "17457".to_string(),
                index: 1,
                parent_title: "".to_string(),
                episodes: vec![
                    PlexEpisode {
                        rating_key: "1".to_string(),
                        view_count: 1,
                        last_viewed_at: Some(12345),
                    },
                    PlexEpisode {
                        rating_key: "2".to_string(),
                        view_count: 1,
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
            anime_list_id: 16498,
            episode_start: 1,
            enabled: true,
            ignored: false,
            episodes: Some(2),
        }];

        let result = get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, 16498);

        assert_eq!(1, result.plex_episodes.len());
        assert_eq!("1", &result.plex_episodes[0].rating_key);
    }

    #[test]
    fn test_get_plex_episodes_for_anime_list_id_when_plex_season_has_less_episodes_than_mapping() {
        let all_plex_series = vec![PlexSeries {
            title: "".to_string(),
            rating_key: "1234".to_string(),
            seasons: vec![PlexSeason {
                index: 1,
                parent_title: "".to_string(),
                rating_key: "17457".to_string(),
                episodes: vec![PlexEpisode {
                    rating_key: "1".to_string(),
                    view_count: 1,
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
            anime_list_id: 16498,
            episode_start: 1,
            enabled: true,
            ignored: false,
            episodes: Some(2),
        }];

        let result = get_plex_episodes_for_anime_list_id(&all_plex_series, &all_mappings, 16498);

        assert_eq!(1, result.plex_episodes.len());
    }
}
