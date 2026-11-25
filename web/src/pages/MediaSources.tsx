import { useMutation, useQueryClient } from "@tanstack/react-query";

import { type MediaSource, type Me, removeMediaSource } from "@/api";
import { AddServer } from "@/components/AddServer";
import { PageHeader } from "@/components/PageHeader";
import { ReSignInBanner } from "@/components/ReSignInBanner";
import { ServerRow } from "@/components/ServerRow";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { refreshAll } from "@/refresh";

export function MediaSources({ me, sources }: { me: Me; sources: MediaSource[] }) {
  const queryClient = useQueryClient();
  const onChanged = () => refreshAll(queryClient);
  const remove = useMutation({ mutationFn: removeMediaSource, onSuccess: onChanged });

  return (
    <>
      <PageHeader title="Media sources" description="Servers your watch history is read from." />
      <ReSignInBanner me={me} />
      <Card className="gap-0 py-0">
        <ul className="divide-y divide-border">
          {sources.map((s) => (
            <ServerRow
              key={s.id}
              url={s.url}
              action={
                <Button variant="outline" size="sm" disabled={remove.isPending} onClick={() => remove.mutate(s.id)}>
                  Remove
                </Button>
              }
            />
          ))}
        </ul>
      </Card>
      {remove.isError && <p className="mt-2 text-sm text-destructive">{remove.error.message}</p>}
      <Card className="mt-6 p-5">
        <AddServer onAdded={onChanged} />
      </Card>
    </>
  );
}
