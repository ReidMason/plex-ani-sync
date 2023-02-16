struct Config;

pub trait ConfigInterface {
    fn get_plex_base_url(&self) -> String;
    fn get_plex_token(&self) -> String;
}

impl ConfigInterface for Config {
    fn get_plex_base_url(&self) -> String {
        String::from("http://10.128.0.100:32400")
    }

    fn get_plex_token(&self) -> String {
        String::from("HbmM3nLd3xodZnkdF9iT")
    }
}
