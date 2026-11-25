import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

class Unauthenticated extends Error {}

interface Me {
  displayName: string;
  plex?: { username: string; needsReSignIn: boolean };
}

interface Pin {
  id: number;
  code: string;
  authUrl: string;
}

async function fetchMe(): Promise<Me> {
  const res = await fetch("/api/me");
  if (res.status === 401) throw new Unauthenticated();
  if (!res.ok) throw new Error(`current user failed: ${res.status}`);
  return res.json();
}

interface MediaSource {
  id: number;
  url: string;
}

const addServerErrors: Record<string, string> = {
  invalid_url: "Enter an http or https address, such as 192.168.1.5:32400",
  unreachable: "Could not reach that server. Check the address and that Plex is running",
  not_plex_server: "That address answered, but it is not a Plex server",
  no_access: "That server rejected your Plex account. Make sure you have access to it",
  already_added: "That server is already added",
  plex_not_linked: "Sign in with Plex first",
};

async function fetchMediaSources(): Promise<MediaSource[]> {
  const res = await fetch("/api/media-sources");
  if (!res.ok) throw new Error(`listing media sources failed: ${res.status}`);
  return res.json();
}

async function removeMediaSource(id: number): Promise<void> {
  const res = await fetch(`/api/media-sources/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`Removing the server failed (${res.status})`);
}

async function addMediaSource(url: string): Promise<void> {
  const res = await fetch("/api/media-sources", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  if (res.ok) return;
  const { error } = await res.json().catch(() => ({ error: "" }));
  throw new Error(addServerErrors[error] ?? `Adding the server failed (${res.status})`);
}

const pollIntervalMs = 2000;

async function createPin(): Promise<Pin> {
  const res = await fetch("/api/plex/pins", { method: "POST" });
  if (!res.ok) throw new Error(`creating plex pin failed: ${res.status}`);
  return res.json();
}

async function pollPin(pin: Pin): Promise<"pending" | "linked"> {
  const res = await fetch(`/api/plex/pins/${pin.id}/poll?code=${encodeURIComponent(pin.code)}`, { method: "POST" });
  if (!res.ok) throw new Error(`polling plex pin failed: ${res.status}`);
  return (await res.json()).status;
}

function PlexSignIn({ onLinked }: { onLinked: () => void }) {
  const signIn = useMutation({
    mutationFn: async () => {
      const popup = window.open("about:blank", "_blank");
      try {
        const pin = await createPin();
        if (popup) popup.location.href = pin.authUrl;
        else window.open(pin.authUrl, "_blank");
        for (;;) {
          await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
          if ((await pollPin(pin)) === "linked") return;
        }
      } finally {
        popup?.close();
      }
    },
    onSuccess: onLinked,
  });

  return (
    <>
      <Button className="mt-4" disabled={signIn.isPending} onClick={() => signIn.mutate()}>
        {signIn.isPending ? "Waiting for Plex" : "Sign in with Plex"}
      </Button>
      {signIn.isError && <p className="mt-2 text-destructive">Plex sign-in failed, try again</p>}
    </>
  );
}

function AddServer({ onAdded }: { onAdded: () => void }) {
  const [url, setUrl] = useState("");
  const add = useMutation({ mutationFn: () => addMediaSource(url), onSuccess: onAdded });

  return (
    <form
      className="mt-4 flex flex-col gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        add.mutate();
      }}
    >
      <label htmlFor="server-url" className="text-sm font-medium">
        Plex server address
      </label>
      <Input
        id="server-url"
        placeholder="192.168.1.5:32400 or plex.example.com"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
      />
      <Button type="submit" disabled={add.isPending || url.trim() === ""}>
        {add.isPending ? "Checking server" : "Add Plex server"}
      </Button>
      {add.isError && <p className="text-destructive">{add.error.message}</p>}
    </form>
  );
}

function Wizard({ me, onChanged }: { me: Me; onChanged: () => void }) {
  return (
    <>
      <p className="mt-2 font-medium">Set up your media source</p>
      {me.plex ? (
        <>
          <p className="text-muted-foreground">Step 2 of 2: add a Plex server</p>
          <AddServer onAdded={onChanged} />
        </>
      ) : (
        <>
          <p className="text-muted-foreground">Step 1 of 2: sign in with Plex</p>
          <PlexSignIn onLinked={onChanged} />
        </>
      )}
    </>
  );
}

function Dashboard({ me, sources, onChanged }: { me: Me; sources: MediaSource[]; onChanged: () => void }) {
  const remove = useMutation({ mutationFn: removeMediaSource, onSuccess: onChanged });

  return (
    <>
      <p className="text-muted-foreground">Plex: {me.plex?.username}</p>
      {me.plex?.needsReSignIn && (
        <div className="mt-2">
          <p className="text-destructive">Plex rejected your sign-in. Sign in with Plex again to keep syncing.</p>
          <PlexSignIn onLinked={onChanged} />
        </div>
      )}
      <h2 className="mt-4 font-medium">Media sources</h2>
      <ul>
        {sources.map((s) => (
          <li key={s.id} className="flex items-center justify-between gap-4 text-muted-foreground">
            {s.url}
            <Button variant="outline" disabled={remove.isPending} onClick={() => remove.mutate(s.id)}>
              Remove
            </Button>
          </li>
        ))}
      </ul>
      {remove.isError && <p className="text-destructive">{remove.error.message}</p>}
      <AddServer onAdded={onChanged} />
    </>
  );
}

export function App() {
  const queryClient = useQueryClient();
  const me = useQuery({ queryKey: ["me"], queryFn: fetchMe, retry: false });
  const logout = useMutation({
    mutationFn: () => fetch("/auth/logout", { method: "POST" }),
    onSuccess: () => window.location.assign("/auth/login"),
  });
  const sources = useQuery({ queryKey: ["media-sources"], queryFn: fetchMediaSources, enabled: me.isSuccess });
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ["me"] });
    void queryClient.invalidateQueries({ queryKey: ["media-sources"] });
  };
  const unauthenticated = me.error instanceof Unauthenticated;

  useEffect(() => {
    if (unauthenticated) window.location.assign("/auth/login");
  }, [unauthenticated]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-secondary p-8 font-sans text-foreground">
      <Card className="p-8">
        <h1 className="text-2xl font-semibold">Plex Ani Sync</h1>
        {me.isSuccess ? (
          <>
            <p className="text-muted-foreground">Signed in as {me.data.displayName}</p>
            {sources.isSuccess &&
              (sources.data.length === 0 ? (
                <Wizard me={me.data} onChanged={refresh} />
              ) : (
                <Dashboard me={me.data} sources={sources.data} onChanged={refresh} />
              ))}
            {sources.isError && <p className="text-destructive">Could not load media sources</p>}
            <Button className="mt-4" onClick={() => logout.mutate()}>
              Sign out
            </Button>
          </>
        ) : (
          <p className="text-muted-foreground">
            {me.isError && !unauthenticated ? "Server is unreachable" : "Signing in"}
          </p>
        )}
      </Card>
    </main>
  );
}
