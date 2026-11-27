import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { linkAniList, unlinkAniList, useAniListUsername } from "@/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function AniListLink({ onLinked }: { onLinked?: () => void }) {
  const queryClient = useQueryClient();
  const current = useAniListUsername();
  const [username, setUsername] = useState("");
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["anilist"] });
  const link = useMutation({
    mutationFn: () => linkAniList(username.trim()),
    onSuccess: async () => {
      setUsername("");
      await refresh();
      onLinked?.();
    },
  });
  const unlink = useMutation({ mutationFn: unlinkAniList, onSuccess: refresh });
  const linked = current.data;

  return (
    <div className="flex flex-col gap-3">
      {linked && (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-4 py-3">
          <p className="text-sm">
            Linked as <span className="font-medium">{linked}</span>
          </p>
          <Button variant="outline" size="sm" disabled={unlink.isPending} onClick={() => unlink.mutate()}>
            Remove
          </Button>
        </div>
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
