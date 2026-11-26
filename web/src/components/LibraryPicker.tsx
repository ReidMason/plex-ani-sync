import { useMutation, useQueryClient } from "@tanstack/react-query";

import { setAnimeLibraries, useLibraries } from "@/api";

export function LibraryPicker({ mediaSourceId }: { mediaSourceId: number }) {
  const queryClient = useQueryClient();
  const libraries = useLibraries(mediaSourceId);
  const save = useMutation({
    mutationFn: (keys: string[]) => setAnimeLibraries(mediaSourceId, keys),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["libraries", mediaSourceId] });
      queryClient.removeQueries({ queryKey: ["anime"] });
    },
  });

  if (libraries.isPending) return <p className="text-sm text-muted-foreground">Loading libraries</p>;
  if (libraries.isError) return <p className="text-sm text-destructive">{libraries.error.message}</p>;
  if (libraries.data.length === 0) return <p className="text-sm text-muted-foreground">No show libraries on this server.</p>;

  const toggle = (key: string, checked: boolean) => {
    const keys = libraries.data.filter((l) => (l.key === key ? checked : l.selected)).map((l) => l.key);
    save.mutate(keys);
  };

  return (
    <fieldset disabled={save.isPending} className="flex flex-col gap-2">
      <legend className="mb-1 text-sm text-muted-foreground">Libraries that contain anime</legend>
      {libraries.data.map((l) => (
        <label key={l.key} className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={l.selected} onChange={(e) => toggle(l.key, e.target.checked)} />
          {l.title}
        </label>
      ))}
      {save.isError && <p className="text-sm text-destructive">{save.error.message}</p>}
    </fieldset>
  );
}
