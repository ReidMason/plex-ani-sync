# Ingest community mapping datasets and keep Custom Mappings alongside them

Show-to-Entry Mappings come from community datasets (Fribb/anime-lists, Anime-Lists) ingested into our own table and refreshed daily, rather than fetched live, so syncs don't depend on a third party and upstream changes can't silently alter results. Mappings are keyed by TVDB show ID so they work across Media Sources. Mappings we maintain ourselves live in the same table, distinguished by an origin; a refresh replaces only dataset rows and drops any incoming row whose TVDB or Entry episode range overlaps a Custom Mapping. We chose this over per-User overrides because bad mappings should be fixed once for everyone, in code review.

## Consequences

- Dataset rows that partly overlap a Custom Mapping are dropped whole, never trimmed; uncovered episodes surface as Needs Review.
- Where dataset rows overlap each other, the first in a fixed order wins, so results stay consistent.
- Status (mapped, Needs Review, Unmapped) is computed on read, never stored.
