import { useMutation } from "@tanstack/react-query";
import { Link, Outlet } from "@tanstack/react-router";

import type { Me } from "@/api";
import { Brand } from "@/components/Brand";
import { Button } from "@/components/ui/button";

const navClass =
  "rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground";
const navActiveClass = "bg-accent text-foreground shadow-[inset_2px_0_0_var(--primary)]";

export function AppShell({ me }: { me: Me }) {
  const logout = useMutation({
    mutationFn: () => fetch("/auth/logout", { method: "POST" }),
    onSuccess: () => window.location.assign("/auth/login"),
  });

  return (
    <div className="flex min-h-screen flex-col md:flex-row">
      <aside className="flex flex-col gap-4 border-b border-border bg-sidebar p-4 md:sticky md:top-0 md:h-screen md:w-60 md:shrink-0 md:gap-8 md:border-r md:border-b-0 md:p-5">
        <Brand />
        <nav aria-label="Main" className="flex gap-1 md:flex-col">
          <Link to="/" activeOptions={{ exact: true }} className={navClass} activeProps={{ className: navActiveClass }}>
            Overview
          </Link>
          <Link to="/anime" className={navClass} activeProps={{ className: navActiveClass }}>
            Anime
          </Link>
          <Link to="/media-sources" className={navClass} activeProps={{ className: navActiveClass }}>
            Media sources
          </Link>
          <Link to="/settings" className={navClass} activeProps={{ className: navActiveClass }}>
            Settings
          </Link>
        </nav>
        <div className="flex items-center justify-between gap-3 md:mt-auto md:flex-col md:items-stretch">
          <p className="truncate text-sm text-muted-foreground">
            Signed in as <span className="text-foreground">{me.displayName}</span>
          </p>
          <Button variant="outline" disabled={logout.isPending} onClick={() => logout.mutate()}>
            Sign out
          </Button>
        </div>
      </aside>
      <main className="min-w-0 flex-1 p-6 md:p-10">
        <div className="mx-auto max-w-3xl">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
