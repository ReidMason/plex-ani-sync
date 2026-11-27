import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { linkAniList, unlinkAniList, useAniListStatus } from "@/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const outcomeMessages: Record<string, string> = {
  denied: "AniList authorisation was denied. Your Tracker is unchanged",
  invalid: "AniList authorisation could not be verified. Your Tracker is unchanged. Try again",
  failed: "Could not complete authorisation with AniList. Your Tracker is unchanged. Try again",
};

function outcomeFromUrl(): { message: string; error: boolean } | null {
  const url = new URL(window.location.href);
  const result = url.searchParams.get("anilist");
  if (result === null) return null;
  url.searchParams.delete("anilist");
  window.history.replaceState(window.history.state, "", url);
  if (result === "authorised") return { message: "Authorised on AniList", error: false };
  const message = outcomeMessages[result];
  return message ? { message, error: true } : null;
}

export function AniListLink({ returnTo }: { returnTo: "setup" | "settings" }) {
  const queryClient = useQueryClient();
  const current = useAniListStatus();
  const [username, setUsername] = useState("");
  const [outcome] = useState(outcomeFromUrl);
  const [previous, setPrevious] = useState<string | null>(null);
  const reportedPrevious = current.data?.previousUsername;
  useEffect(() => {
    if (reportedPrevious) setPrevious(reportedPrevious);
  }, [reportedPrevious]);
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["anilist"] });
  const link = useMutation({
    mutationFn: () => linkAniList(username.trim()),
    onSuccess: () => {
      setUsername("");
      return refresh();
    },
  });
  const unlink = useMutation({ mutationFn: unlinkAniList, onSuccess: refresh });
  const revokeHint = (
    <p className="text-xs text-muted-foreground">
      To also revoke access on AniList, visit{" "}
      <a className="underline" href="https://anilist.co/settings/apps" target="_blank" rel="noreferrer">
        anilist.co/settings/apps
      </a>
      .
    </p>
  );
  const linked = current.data?.username;
  const authorised = current.data?.authorised ?? false;
  const canAuthorise = current.data?.authorisationAvailable ?? false;

  return (
    <div className="flex flex-col gap-3">
      {outcome && <p className={`text-sm ${outcome.error ? "text-destructive" : "text-muted-foreground"}`}>{outcome.message}</p>}
      {previous && outcome && !outcome.error && (
        <p className="text-sm text-muted-foreground">Your username changed from {previous} to the authorised account</p>
      )}
      {linked && (
        <div className="flex flex-col gap-3 rounded-lg border border-border px-4 py-3">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm">
              {authorised ? "Authorised as" : "Linked as"} <span className="font-medium">{linked}</span>
            </p>
            <Button variant="outline" size="sm" disabled={unlink.isPending} onClick={() => unlink.mutate()}>
              Remove
            </Button>
          </div>
          {!authorised && <p className="text-sm text-muted-foreground">Authorise to enable syncing</p>}
          {revokeHint}
        </div>
      )}
      {!linked && unlink.isSuccess && (
        <div className="flex flex-col gap-1">
          <p className="text-sm text-muted-foreground">AniList Tracker removed</p>
          {revokeHint}
        </div>
      )}
      {canAuthorise && (
        <Button asChild variant={authorised ? "outline" : "default"} className="self-start">
          <a href={`/auth/anilist/start?return=${returnTo}`}>{authorised ? "Re-authorise on AniList" : "Authorise on AniList"}</a>
        </Button>
      )}
      <form
        className="flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          link.mutate();
        }}
      >
        <label htmlFor="anilist-username" className="text-sm font-medium">
          {linked ? "Change AniList username" : "AniList username"}
        </label>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input id="anilist-username" value={username} onChange={(e) => setUsername(e.target.value)} />
          <Button type="submit" disabled={link.isPending || username.trim() === ""} className="shrink-0">
            {link.isPending ? "Checking username" : linked ? "Change" : "Link"}
          </Button>
        </div>
        {link.isError && <p className="text-sm text-destructive">{link.error.message}</p>}
        {unlink.isError && <p className="text-sm text-destructive">{unlink.error.message}</p>}
      </form>
    </div>
  );
}
