export interface ProcessLogs {
    updates: Array<ProcessLog>;
    syncIsRunning: boolean;
}

export interface ProcessLog {
    seriesTitle: string;
}

export interface AnilistAnime {
    anime_id: number;
    entry_id: number;
    romaji_title: string;
    title: string;
    total_episodes: number;
    watch_status: number;
    watched_episodes: number;
}