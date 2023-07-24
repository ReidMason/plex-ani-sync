use dotenv::dotenv;

pub fn init_logger() {
    tracing_subscriber::fmt::try_init();
}

pub fn get_db_file_location() -> String {
    "./data/data.db?mode=rwc".to_string()
}
