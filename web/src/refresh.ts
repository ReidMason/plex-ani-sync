import type { QueryClient } from "@tanstack/react-query";

export function refreshAll(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: ["me"] });
  void queryClient.invalidateQueries({ queryKey: ["media-sources"] });
}
