import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

class Unauthenticated extends Error {}

async function fetchMe(): Promise<{ displayName: string }> {
  const res = await fetch("/api/me");
  if (res.status === 401) throw new Unauthenticated();
  if (!res.ok) throw new Error(`current user failed: ${res.status}`);
  return res.json();
}

export function App() {
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
