import { useMutation } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { createPin, pollPin } from "@/api";
import { Button } from "@/components/ui/button";

const pollIntervalMs = 2000;

export function PlexSignIn({ onLinked }: { onLinked: () => void }) {
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  const signIn = useMutation({
    mutationFn: async () => {
      controller.current?.abort();
      const { signal } = (controller.current = new AbortController());
      const popup = window.open("about:blank", "_blank");
      try {
        const pin = await createPin();
        if (popup) popup.location.href = pin.authUrl;
        else window.open(pin.authUrl, "_blank");
        while (!signal.aborted) {
          await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
          if (signal.aborted) return;
          if ((await pollPin(pin, signal)) === "linked") return;
        }
      } finally {
        popup?.close();
      }
    },
    onSuccess: () => {
      if (!controller.current?.signal.aborted) onLinked();
    },
  });

  return (
    <div className="flex flex-col items-start gap-2">
      <Button disabled={signIn.isPending} onClick={() => signIn.mutate()}>
        {signIn.isPending ? "Waiting for Plex" : "Sign in to Plex"}
      </Button>
      {signIn.isError && <p className="text-sm text-destructive">Plex sign-in failed, try again</p>}
    </div>
  );
}
