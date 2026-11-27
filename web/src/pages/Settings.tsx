import { AniListLink } from "@/components/AniListLink";
import { PageHeader } from "@/components/PageHeader";
import { Card } from "@/components/ui/card";

export function Settings() {
  return (
    <>
      <PageHeader title="Settings" description="Where your watch progress is tracked. Your AniList list must be public." />
      <Card className="p-5">
        <AniListLink />
      </Card>
    </>
  );
}
