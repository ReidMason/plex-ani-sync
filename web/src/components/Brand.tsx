export function Brand() {
  return (
    <div className="flex items-center gap-2.5">
      <svg viewBox="0 0 24 24" className="size-7 text-primary" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M4 9a8 8 0 0 1 14-3l2 2" />
        <path d="M20 4v4h-4" />
        <path d="M20 15a8 8 0 0 1-14 3l-2-2" />
        <path d="M4 20v-4h4" />
      </svg>
      <span className="font-display text-lg font-semibold tracking-tight">Plex Ani Sync</span>
    </div>
  );
}
