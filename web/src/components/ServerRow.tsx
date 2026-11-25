import type { ReactNode } from "react";

export function ServerRow({ url, action }: { url: string; action?: ReactNode }) {
  return (
    <li className="flex items-center justify-between gap-4 px-4 py-3">
      <span className="flex min-w-0 items-center gap-3">
        <span className="size-2 shrink-0 rounded-full bg-success" aria-hidden="true" />
        <span className="truncate font-mono text-sm">{url}</span>
      </span>
      {action}
    </li>
  );
}
