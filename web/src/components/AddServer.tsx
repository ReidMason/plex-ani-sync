import { useMutation } from "@tanstack/react-query";
import { useState } from "react";

import { addMediaSource } from "@/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function AddServer({ onAdded }: { onAdded: () => void }) {
  const [url, setUrl] = useState("");
  const add = useMutation({
    mutationFn: () => addMediaSource(url),
    onSuccess: () => {
      setUrl("");
      onAdded();
    },
  });

  return (
    <form
      className="flex flex-col gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        add.mutate();
      }}
    >
      <label htmlFor="server-url" className="text-sm font-medium">
        Plex server address
      </label>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          id="server-url"
          placeholder="192.168.1.5:32400 or plex.example.com"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <Button type="submit" disabled={add.isPending || url.trim() === ""} className="shrink-0">
          {add.isPending ? "Checking server" : "Add server"}
        </Button>
      </div>
      {add.isError && <p className="text-sm text-destructive">{add.error.message}</p>}
    </form>
  );
}
