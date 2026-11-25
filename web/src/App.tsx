import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

class Unauthenticated extends Error {}

interface Me {
  displayName: string;
  plex?: { username: string };
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

export function App() {
  const queryClient = useQueryClient();
  const me = useQuery({ queryKey: ["me"], queryFn: fetchMe, retry: false });
  const logout = useMutation({
    mutationFn: () => fetch("/auth/logout", { method: "POST" }),
    onSuccess: () => window.location.assign("/auth/login"),
  });
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
            {me.data.plex ? (
              <p className="text-muted-foreground">Plex: {me.data.plex.username}</p>
            ) : (
              <PlexSignIn onLinked={() => void queryClient.invalidateQueries({ queryKey: ["me"] })} />
            )}
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
