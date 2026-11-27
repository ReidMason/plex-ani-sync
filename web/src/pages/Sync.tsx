import { useMutation } from "@tanstack/react-query";

import { type SyncPlan, syncNow } from "@/api";
import { PageHeader } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

function Plan({ plan }: { plan: SyncPlan }) {
  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-2">
        <h2 className="font-display text-lg font-semibold">Changes ({plan.changes.length})</h2>
        {plan.changes.length === 0 ? (
          <p className="text-muted-foreground">Nothing to change.</p>
        ) : (
          <Card className="gap-0 py-0">
            <ul className="divide-y divide-border">
              {plan.changes.map((c) => (
                <li key={c.anilistId} className="flex flex-col gap-1 px-4 py-3">
                  <p className="font-medium">{c.title}</p>
                  <p className="text-sm">
                    {c.current ? `${c.current.status} at ${c.current.progress}` : "Not on list"} to {c.proposed.status} at{" "}
                    {c.proposed.progress}
                  </p>
                  <p className="text-sm text-muted-foreground">{c.reason}</p>
                </li>
              ))}
            </ul>
          </Card>
        )}
      </section>
      {plan.skipped.length > 0 && (
        <section className="flex flex-col gap-2">
          <h2 className="font-display text-lg font-semibold">Skipped ({plan.skipped.length})</h2>
          <Card className="gap-0 py-0">
            <ul className="divide-y divide-border">
              {plan.skipped.map((s, i) => (
                <li key={`${s.title}-${i}`} className="flex flex-col gap-1 px-4 py-3">
                  <p className="font-medium">{s.title}</p>
                  <p className="text-sm text-muted-foreground">{s.reason}</p>
                </li>
              ))}
            </ul>
          </Card>
        </section>
      )}
    </div>
  );
}

export function Sync() {
  const sync = useMutation({ mutationFn: syncNow });

  return (
    <>
      <PageHeader title="Sync" description="Preview what your AniList list would look like if it matched Plex. Nothing is written to AniList." />
      <div className="mb-6">
        <Button disabled={sync.isPending} onClick={() => sync.mutate()}>
          {sync.isPending ? "Syncing" : "Sync now"}
        </Button>
      </div>
      {sync.isError && <p className="text-destructive">{sync.error.message}</p>}
      {sync.isSuccess && <Plan plan={sync.data} />}
    </>
  );
}
