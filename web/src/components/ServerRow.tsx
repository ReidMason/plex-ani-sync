import type { ReactNode } from "react";

export function ServerRow({ url, action, children }: { url: string; action?: ReactNode; children?: ReactNode }) {
  return (
    <li className="px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <span className="flex min-w-0 items-center gap-3">
          <span className="size-2 shrink-0 rounded-full bg-success" aria-hidden="true" />
          <span className="truncate font-mono text-sm">{url}</span>
        </span>
        {action}
      </div>
      {children && <div className="mt-3 pl-5">{children}</div>}
    </li>
  );
}
