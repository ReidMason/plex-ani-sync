import { useQuery } from "@tanstack/react-query";

import { Card } from "@/components/ui/card";

export function App() {
  const { isSuccess, isError } = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const res = await fetch("/api/health");
      if (!res.ok) throw new Error(`health check failed: ${res.status}`);
    },
    retry: false,
  });

  return (
    <main className="flex min-h-screen items-center justify-center bg-secondary p-8 font-sans text-foreground">
      <Card className="p-8">
        <h1 className="text-2xl font-semibold">Plex Ani Sync</h1>
        <p className="text-muted-foreground">
          {isSuccess ? "Server is healthy" : isError ? "Server is unreachable" : "Checking server"}
        </p>
      </Card>
    </main>
  );
}
