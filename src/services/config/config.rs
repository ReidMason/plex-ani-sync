use crate::services::dbstore::sqlite;

#[derive(Debug, Clone)]
pub struct ConfigService {
    config: sqlite::Config,
}

impl ConfigService {
    pub fn new(config: sqlite::Config) -> Self {
        Self { config }
    }
}

pub trait ConfigInterface: Sync + Send {
    fn get_plex_base_url(&self) -> &str;
    fn get_plex_token(&self) -> &str;
    fn get_anilist_token(&self) -> &str;
}

impl ConfigInterface for ConfigService {
    fn get_plex_base_url(&self) -> &str {
        &self.config.plex_url
    }

    fn get_plex_token(&self) -> &str {
        &self.config.plex_token
    }

    fn get_anilist_token(&self) -> &str {
        &self.config.anilist_token
    }
}

#[derive(Debug)]
pub struct MockConfig {
    pub plex_base_url: String,
    pub plex_token: String,
    pub anilist_token: String,
}

impl ConfigInterface for MockConfig {
    fn get_plex_base_url(&self) -> &str {
        &self.plex_base_url
    }

    fn get_plex_token(&self) -> &str {
        &self.plex_token
    }

    fn get_anilist_token(&self) -> &str {
        &self.anilist_token
    }
}
