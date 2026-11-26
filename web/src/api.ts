import { useQuery } from "@tanstack/react-query";

export class Unauthenticated extends Error {}

export interface Me {
  displayName: string;
  plex?: { username?: string; needsReSignIn: boolean };
}

export interface MediaSource {
  id: number;
  url: string;
  kind: "plex";
}

export interface Library {
  key: string;
  title: string;
  selected: boolean;
}

interface AnimeEpisode {
  number: number;
  title: string;
  watched: boolean;
  lastWatchedAt: string | null;
}

export interface AnimeShow {
  mediaSourceId: number;
  title: string;
  seasons: { number: number; episodes: AnimeEpisode[] }[];
}

interface Pin {
  id: number;
  code: string;
  authUrl: string;
}

const addServerErrors: Record<string, string> = {
  invalid_url: "Enter an http or https address, such as 192.168.1.5:32400",
  unreachable: "Could not reach that server. Check the address and that Plex is running",
  not_plex_server: "That address answered, but it is not a Plex server",
  no_access: "That server rejected your Plex account. Make sure you have access to it",
  already_added: "That server is already added",
  plex_not_linked: "Sign in to Plex first",
};

async function fetchMe(): Promise<Me> {
  const res = await fetch("/api/me");
  if (res.status === 401) throw new Unauthenticated();
  if (!res.ok) throw new Error(`current user failed: ${res.status}`);
  return res.json();
}

async function fetchMediaSources(): Promise<MediaSource[]> {
  const res = await fetch("/api/media-sources");
  if (!res.ok) throw new Error(`listing media sources failed: ${res.status}`);
  return res.json();
}

export async function removeMediaSource(id: number): Promise<void> {
  const res = await fetch(`/api/media-sources/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`Removing the server failed (${res.status})`);
}

export async function addMediaSource(url: string): Promise<void> {
  const res = await fetch("/api/media-sources", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  if (res.ok) return;
  const { error } = await res.json().catch(() => ({ error: "" }));
  throw new Error(addServerErrors[error] ?? `Adding the server failed (${res.status})`);
}

export async function createPin(): Promise<Pin> {
  const res = await fetch("/api/plex/pins", { method: "POST" });
  if (!res.ok) throw new Error(`creating plex pin failed: ${res.status}`);
  return res.json();
}

export async function pollPin(pin: Pin, signal: AbortSignal): Promise<"pending" | "linked"> {
  const res = await fetch(`/api/plex/pins/${pin.id}/poll?code=${encodeURIComponent(pin.code)}`, {
    method: "POST",
    signal,
  });
  if (!res.ok) throw new Error(`polling plex pin failed: ${res.status}`);
  return (await res.json()).status;
}

export function useMe() {
  return useQuery({ queryKey: ["me"], queryFn: fetchMe, retry: false });
}

export function useMediaSources(enabled: boolean) {
  return useQuery({ queryKey: ["media-sources"], queryFn: fetchMediaSources, enabled });
}

export function plexIsLinked(me: Me): boolean {
  return !!me.plex && !me.plex.needsReSignIn;
}

export function useLibraries(mediaSourceId: number) {
  return useQuery({
    queryKey: ["libraries", mediaSourceId],
    queryFn: async (): Promise<Library[]> => {
      const res = await fetch(`/api/media-sources/${mediaSourceId}/libraries`);
      if (!res.ok) throw new Error(`Could not load libraries (${res.status})`);
      return res.json();
    },
  });
}

export async function setAnimeLibraries(mediaSourceId: number, keys: string[]): Promise<void> {
  const res = await fetch(`/api/media-sources/${mediaSourceId}/anime-libraries`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ keys }),
  });
  if (!res.ok) throw new Error(`Saving libraries failed (${res.status})`);
}

export function useAnime() {
  return useQuery({
    queryKey: ["anime"],
    queryFn: async (): Promise<AnimeShow[]> => {
      const res = await fetch("/api/anime");
      if (!res.ok) throw new Error(`Could not load anime (${res.status})`);
      return res.json();
    },
  });
}
