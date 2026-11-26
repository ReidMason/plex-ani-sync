import { Link } from "@tanstack/react-router";

import { type AnimeShow, useAnime } from "@/api";
import { PageHeader } from "@/components/PageHeader";
import { Card } from "@/components/ui/card";

const statusLabels: Record<AnimeShow["status"], string> = {
  mapped: "Mapped",
  needs_review: "Needs review",
  unmapped: "Unmapped",
};

function range(first: number, last: number | null) {
  if (last === null) return `${first}+`;
  return first === last ? `${first}` : `${first}-${last}`;
}

export function Anime() {
  const anime = useAnime();

  return (
    <>
      <PageHeader title="Anime" description="Shows in your anime libraries and how they map to AniList." />
      {anime.isPending && <p className="text-muted-foreground">Loading from Plex</p>}
      {anime.isError && <p className="text-destructive">{anime.error.message}</p>}
      {anime.isSuccess && anime.data.length === 0 && (
        <p className="text-muted-foreground">
          No anime yet. Pick libraries on the{" "}
          <Link to="/media-sources" className="text-primary hover:underline">
            Media sources
          </Link>{" "}
          page.
        </p>
      )}
      {anime.isSuccess && anime.data.length > 0 && (
        <Card className="gap-0 py-0">
          <ul className="divide-y divide-border">
            {anime.data.map((show, i) => (
              <li key={`${show.mediaSourceId}-${i}`} className="px-4 py-3">
                <details>
                  <summary className="cursor-pointer font-medium">
                    {show.title}{" "}
                    <span className={show.status === "mapped" ? "text-sm text-muted-foreground" : "text-sm text-destructive"}>
                      {statusLabels[show.status]}
                    </span>
                  </summary>
                  <div className="mt-2 flex flex-col gap-1 pl-4 text-sm">
                    <p className="text-muted-foreground">
                      TVDB {show.externalIds.tvdb ?? "none"} · TMDB {show.externalIds.tmdb ?? "none"}
                    </p>
                    {show.reason && <p className="text-destructive">{show.reason}</p>}
                    {show.mappings.length > 0 && (
                      <ul>
                        {show.mappings.map((m) => (
                          <li key={`${m.season}-${m.firstEpisode}-${m.anilistId}`}>
                            Season {m.season} episodes {range(m.firstEpisode, m.lastEpisode)} to AniList {m.anilistId} episodes{" "}
                            {range(m.entryFirstEpisode, m.entryLastEpisode)}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                  <ul className="mt-2 flex flex-col gap-2 pl-4">
                    {show.seasons.map((season) => (
                      <li key={season.number}>
                        <details>
                          <summary className="cursor-pointer text-sm">
                            Season {season.number}{" "}
                            <span className="text-muted-foreground">
                              {season.episodes.filter((e) => e.watched).length}/{season.episodes.length} watched
                            </span>
                          </summary>
                          <ul className="mt-1 pl-4 text-sm">
                            {season.episodes.map((e) => (
                              <li key={e.number} className={e.watched ? "" : "text-muted-foreground"}>
                                {e.watched ? "✓" : "·"} {e.number}. {e.title}
                              </li>
                            ))}
                          </ul>
                        </details>
                      </li>
                    ))}
                  </ul>
                </details>
              </li>
            ))}
          </ul>
        </Card>
      )}
    </>
  );
}
