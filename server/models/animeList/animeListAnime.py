class AnimeListAnime:
    def __init__(self, entry_id: int, anime_id: int, title: str, romaji_title: str, total_episodes: int,
                 watched_episodes: int,
                 watch_status: int):
        self.entry_id: int = int(entry_id)
        self.anime_id: int = int(anime_id)
        self.title: str = str(title)
        self.romaji_title: str = str(romaji_title)
        self.total_episodes: int = int(total_episodes if total_episodes is not None else 0)
        self.watched_episodes: int = int(watched_episodes)
        self.watch_status: int = int(watch_status)
