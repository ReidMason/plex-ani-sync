use crate::services::dbstore::sqlite::Mapping;

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

#[cfg(test)]
mod tests {
    use super::*;

    fn create_mapping(plex_id: &str, season_length: u32) -> Mapping {
        return Mapping {
            plex_id: plex_id.to_string(),
            season_length,
            id: 0,
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

    #[tokio::test]
    async fn test_mapped_episode_count() {
        let mappings: Vec<Mapping> = vec![
            create_mapping("12345", 5),
            create_mapping("12345", 5),
            create_mapping("12345678", 5),
        ];
        let rating_key = "12345";

        let result = get_mapped_episode_count(&mappings, rating_key);

        assert_eq!(10, result)
    }

    #[tokio::test]
    async fn test_cleanup_string() {
        let input = ":Some:String line ";
        let result = cleanup_string(input);

        assert_eq!("somestringline", result)
    }
}
