export function AniListSetupNotice({ className = "" }: { className?: string }) {
  return (
    <div role="status" className={`rounded-xl border border-border bg-muted/40 p-4 text-sm ${className}`}>
      AniList authorisation isn't set up on this server, so your list can only be read. Ask the operator to set{" "}
      <code>ANILIST_CLIENT_ID</code> and <code>ANILIST_CLIENT_SECRET</code>. The README explains how to register the AniList client.
    </div>
  );
}
