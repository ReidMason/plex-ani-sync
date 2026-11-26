import { Link } from "@tanstack/react-router";

import { useAnime } from "@/api";
import { PageHeader } from "@/components/PageHeader";
import { Card } from "@/components/ui/card";

export function Anime() {
  const anime = useAnime();

  return (
    <>
      <PageHeader title="Anime" description="Shows in the libraries you marked as anime." />
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
            {anime.data.map((show) => (
              <li key={`${show.mediaSourceId}-${show.title}`} className="px-4 py-3">
                <details>
                  <summary className="cursor-pointer font-medium">{show.title}</summary>
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
