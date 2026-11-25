import { useQueryClient } from "@tanstack/react-query";

import type { Me } from "@/api";
import { PlexSignIn } from "@/components/PlexSignIn";
import { refreshAll } from "@/refresh";

export function ReSignInBanner({ me }: { me: Me }) {
  const queryClient = useQueryClient();
  if (!me.plex?.needsReSignIn) return null;
  return (
    <div
      role="alert"
      className="mb-8 flex flex-col gap-3 rounded-xl border border-destructive/40 bg-destructive/10 p-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <p className="text-sm">Plex rejected your sign-in. Sign in to Plex again to keep syncing.</p>
      <PlexSignIn onLinked={() => refreshAll(queryClient)} />
    </div>
  );
}
