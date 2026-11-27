import { Navigate, Outlet, createRootRoute, createRoute, createRouter, useLocation } from "@tanstack/react-router";
import { useEffect } from "react";

import { Unauthenticated, plexIsLinked, useMe, useMediaSources } from "@/api";
import { AppShell } from "@/components/AppShell";
import { Anime } from "@/pages/Anime";
import { MediaSources } from "@/pages/MediaSources";
import { Overview } from "@/pages/Overview";
import { Settings } from "@/pages/Settings";
import { SetupAniList, SetupPlex, SetupServers } from "@/pages/Setup";

function Splash({ children }: { children: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center text-muted-foreground">
      <p>{children}</p>
    </main>
  );
}

const rootRoute = createRootRoute({ component: Gate });
const overviewRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: () => <Page page="overview" /> });
const animeRoute = createRoute({ getParentRoute: () => rootRoute, path: "/anime", component: Anime });
const mediaSourcesRoute = createRoute({ getParentRoute: () => rootRoute, path: "/media-sources", component: () => <Page page="media-sources" /> });
const setupPlexRoute = createRoute({ getParentRoute: () => rootRoute, path: "/setup/plex", component: () => <Page page="setup-plex" /> });
const setupServersRoute = createRoute({ getParentRoute: () => rootRoute, path: "/setup/servers", component: () => <Page page="setup-servers" /> });
const settingsRoute = createRoute({ getParentRoute: () => rootRoute, path: "/settings", component: Settings });
const setupAniListRoute = createRoute({ getParentRoute: () => rootRoute, path: "/setup/anilist", component: SetupAniList });

export const router = createRouter({
  routeTree: rootRoute.addChildren([overviewRoute, animeRoute, mediaSourcesRoute, settingsRoute, setupPlexRoute, setupServersRoute, setupAniListRoute]),
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

type PageName = "overview" | "media-sources" | "setup-plex" | "setup-servers";

function Gate() {
  const me = useMe();
  const sources = useMediaSources(me.isSuccess);
  const { pathname } = useLocation();
  const unauthenticated = me.error instanceof Unauthenticated;

  useEffect(() => {
    if (unauthenticated) window.location.assign("/auth/login");
  }, [unauthenticated]);

  if (me.isError) return <Splash>{unauthenticated ? "Signing in" : "Server is unreachable"}</Splash>;
  if (sources.isError) return <Splash>Could not load media sources</Splash>;
  if (!me.isSuccess || !sources.isSuccess) return <Splash>Loading</Splash>;

  const linked = plexIsLinked(me.data);
  const setupStep = linked ? "/setup/servers" : "/setup/plex";
  const inSetup = pathname.startsWith("/setup");
  const onAniListStep = linked && pathname === "/setup/anilist";
  if (sources.data.length === 0 ? pathname !== setupStep : inSetup && pathname !== setupStep && !onAniListStep) {
    return <Navigate to={setupStep} replace />;
  }

  return inSetup ? <Outlet /> : <AppShell me={me.data} />;
}

function Page({ page }: { page: PageName }) {
  const me = useMe().data;
  const sources = useMediaSources(false).data;
  if (!me || !sources) return null;
  switch (page) {
    case "overview":
      return <Overview me={me} sources={sources} />;
    case "media-sources":
      return <MediaSources me={me} sources={sources} />;
    case "setup-plex":
      return <SetupPlex />;
    case "setup-servers":
      return <SetupServers sources={sources} />;
  }
}
