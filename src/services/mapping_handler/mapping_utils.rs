struct ResultScore {
    result: AnimeResult,
    score: u16,
}

use crate::services::{
    anime_list_service::anime_list_service::{AnimeResult, RelationType},
    dbstore::sqlite::Mapping,
    plex::plex_api::PlexSeason,
};

pub fn get_mapped_episode_count(mappings: &[Mapping], rating_key: &str) -> u32 {
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

fn cleanup_string(string: &str) -> String {
    string.replace([':', ' '], "").trim().to_lowercase()
}

pub fn compare_strings(string1: &str, string2: &str) -> bool {
    cleanup_string(string1) == cleanup_string(string2)
}

pub fn find_match(
    results: Vec<AnimeResult>,
    target: &PlexSeason,
    offset: u16,
) -> Option<AnimeResult> {
    let mut potential_matches: Vec<ResultScore> = results
        .into_iter()
        .map(|x| ResultScore {
            result: x,
            score: offset,
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

        let has_no_prequel = potential_match
            .result
            .relations
            .edges
            .iter()
            .filter(|x| x.relation_type == RelationType::Prequel)
            .count()
            == 0;
        if has_no_prequel && target.index == 1 {
            potential_match.score += 50;
        }
    });

    potential_matches.sort_by_key(|x| x.score);

    let min_score = 0;
    potential_matches.retain(|x| x.score > min_score);

    match potential_matches.pop() {
        Some(x) => Some(x.result),
        None => None,
    }
}

pub fn get_prev_mapping(mappings: &[Mapping], rating_key: &str) -> Option<Mapping> {
    let mut prev_mappings: Vec<&Mapping> = mappings
        .iter()
        .filter(|x| x.plex_id == rating_key)
        .collect();

    if !prev_mappings.is_empty() {
        prev_mappings.sort_by_key(|x| x.plex_episode_start);
        prev_mappings.reverse();
        return Some(prev_mappings[0].to_owned());
    }

    None
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_mapping(id: u32, plex_id: &str, season_length: u32) -> Mapping {
        return Mapping {
            id,
            plex_id: plex_id.to_string(),
            season_length,
            list_provider_id: 0,
            plex_series_id: "".to_string(),
            plex_episode_start: 0,
            anime_list_id: 0,
            episode_start: 0,
            enabled: true,
            ignored: true,
            episodes: Some(0),
        };
    }

    #[test]
    fn test_mapped_episode_count() {
        let mappings: Vec<Mapping> = vec![
            create_mapping(1, "12345", 5),
            create_mapping(2, "12345", 5),
            create_mapping(3, "12345678", 5),
        ];
        let rating_key = "12345";

        let result = get_mapped_episode_count(&mappings, rating_key);

        assert_eq!(10, result)
    }

    #[test]
    fn test_cleanup_string() {
        let input = ":Some:String line ";
        let result = cleanup_string(input);

        assert_eq!("somestringline", result)
    }

    #[test]
    fn test_compare_strings() {
        let input1 = ":Some:String line ";
        let input2 = " some String: line";
        let result = compare_strings(input1, input2);

        assert!(result)
    }

    #[test]
    fn test_compare_strings_returns_false_when_not_the_same() {
        let input1 = ":Some:String line ";
        let input2 = " some String: line1";
        let result = compare_strings(input1, input2);

        assert!(!result)
    }

    #[test]
    fn test_get_prev_mapping() {
        let mappings: Vec<Mapping> = vec![
            create_mapping(1, "12345", 0),
            create_mapping(2, "12345", 0),
            create_mapping(3, "123456", 0),
        ];
        let rating_key = "12345";

        let result = get_prev_mapping(&mappings, rating_key);
        let result = match result {
            Some(x) => x,
            None => panic!(),
        };

        assert_eq!(2, result.id)
    }

    #[test]
    fn test_get_prev_mapping_when_one_doesnt_exist() {
        let mappings: Vec<Mapping> = vec![
            create_mapping(1, "12345", 0),
            create_mapping(2, "12345", 0),
            create_mapping(3, "123456", 0),
        ];
        let rating_key = "1234566789";

        let result = get_prev_mapping(&mappings, rating_key);
        assert!(result.is_none())
    }
}
