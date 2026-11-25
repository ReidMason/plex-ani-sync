import { Link } from "@tanstack/react-router";

import { type MediaSource, type Me } from "@/api";
import { PageHeader } from "@/components/PageHeader";
import { ReSignInBanner } from "@/components/ReSignInBanner";
import { ServerRow } from "@/components/ServerRow";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export function Overview({ me, sources }: { me: Me; sources: MediaSource[] }) {
  const needsSignIn = me.plex?.needsReSignIn;
  return (
    <>
      <PageHeader title="Overview" description="Where your watch history comes from." />
      <ReSignInBanner me={me} />
      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="gap-3 p-5">
          <h2 className="text-sm font-medium text-muted-foreground">Plex account</h2>
          <p className="font-display text-xl font-semibold">{me.plex?.username ?? "Linked"}</p>
          <p className="flex items-center gap-2 text-sm">
            <span className={`size-2 rounded-full ${needsSignIn ? "bg-destructive" : "bg-success"}`} aria-hidden="true" />
            {needsSignIn ? "Needs sign-in" : "Connected"}
          </p>
        </Card>
        <Card className="gap-3 p-5">
          <h2 className="text-sm font-medium text-muted-foreground">Media sources</h2>
          <p className="font-display text-xl font-semibold">
            {sources.length} {sources.length === 1 ? "server" : "servers"}
          </p>
          <Link to="/media-sources" className="text-sm text-primary hover:underline">
            Manage servers
          </Link>
        </Card>
      </div>
      <Card className="mt-4 gap-0 py-0">
        <ul className="divide-y divide-border">
          {sources.map((s) => (
            <ServerRow key={s.id} url={s.url} />
          ))}
        </ul>
      </Card>
      <div className="mt-8 flex items-center justify-between gap-4 border-t border-border pt-6">
        <p className="text-sm text-muted-foreground">Changed accounts or servers? Walk through setup again.</p>
        <Button variant="outline" asChild>
          <Link to="/setup/plex">Run setup again</Link>
        </Button>
      </div>
    </>
  );
}
