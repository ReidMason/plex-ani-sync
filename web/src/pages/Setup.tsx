import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { type MediaSource, useAniListUsername } from "@/api";
import { AddServer } from "@/components/AddServer";
import { AniListLink } from "@/components/AniListLink";
import { Brand } from "@/components/Brand";
import { PlexSignIn } from "@/components/PlexSignIn";
import { ServerRow } from "@/components/ServerRow";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { refreshAll } from "@/refresh";

const steps = ["Sign in to Plex", "Add a server", "Link AniList"];

function StepRail({ current }: { current: 1 | 2 | 3 }) {
  return (
    <ol className="mb-8 flex items-center" aria-label="Setup progress">
      {steps.map((label, i) => {
        const n = i + 1;
        const done = n < current;
        const active = n === current;
        return (
          <li key={label} className="flex flex-1 items-center last:flex-none" aria-current={active ? "step" : undefined}>
            <span
              className={`flex size-8 shrink-0 items-center justify-center rounded-full border text-sm font-semibold ${
                done || active ? "border-primary bg-primary text-primary-foreground" : "border-border text-muted-foreground"
              }`}
            >
              {done ? "✓" : n}
            </span>
            <span className={`ml-3 text-sm font-medium ${active ? "text-foreground" : "text-muted-foreground"}`}>{label}</span>
            {n < steps.length && <span className={`mx-4 h-px flex-1 ${done ? "bg-primary" : "bg-border"}`} aria-hidden="true" />}
          </li>
        );
      })}
    </ol>
  );
}

function SetupFrame({ step, title, description, children }: { step: 1 | 2 | 3; title: string; description: string; children: ReactNode }) {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 p-6">
      <Brand />
      <Card className="w-full max-w-xl gap-0 p-8">
        <StepRail current={step} />
        <h1 className="font-display text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="mt-1 mb-6 text-muted-foreground">{description}</p>
        {children}
      </Card>
    </main>
  );
}

export function SetupPlex() {
  const queryClient = useQueryClient();
  return (
    <SetupFrame step={1} title="Connect your Plex account" description="Plex Ani Sync reads your watch history through your Plex account.">
      <PlexSignIn onLinked={() => refreshAll(queryClient)} />
    </SetupFrame>
  );
}

export function SetupServers({ sources }: { sources: MediaSource[] }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  return (
    <SetupFrame step={2} title="Add your Plex server" description="Enter the address of a server your Plex account can access.">
      {sources.length > 0 && (
        <ul className="mb-6 divide-y divide-border rounded-lg border border-border">
          {sources.map((s) => (
            <ServerRow key={s.id} url={s.url} />
          ))}
        </ul>
      )}
      <AddServer onAdded={() => refreshAll(queryClient)} />
      <Button className="mt-6 self-end" disabled={sources.length === 0} onClick={() => void navigate({ to: "/setup/anilist" })}>
        Continue
      </Button>
    </SetupFrame>
  );
}

export function SetupAniList() {
  const navigate = useNavigate();
  const linked = useAniListUsername().data;
  return (
    <SetupFrame step={3} title="Link your AniList account" description="Enter your public AniList username. Your anime list must be public so it can be read.">
      <AniListLink />
      <div className="mt-6 flex justify-end gap-2">
        <Button variant="outline" onClick={() => void navigate({ to: "/" })}>
          Skip for now
        </Button>
        <Button disabled={!linked} onClick={() => void navigate({ to: "/" })}>
          Done
        </Button>
      </div>
    </SetupFrame>
  );
}
