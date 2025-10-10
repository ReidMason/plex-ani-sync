import {getPlexAuthUrl} from "@/lib/api/plexAuth/PlexAuthApi";
import {useEffect, useState} from "react";
import {Button} from "@/components/ui/button";

function getForwardUrl(): string {
  const url = new URL(window.location.href);
  url.pathname = url.pathname.replace(/\/$/, "") + "/pin";
  return url.toString();
}

export default function PlexSetup() {
  const [authUrl, setAuthUrl] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>("");

  const fetchAuthUrl = async () => {
    setIsLoading(true);

    const response = await getPlexAuthUrl(getForwardUrl());
    if (response.success) {
      const data = await response.data;
      setAuthUrl(data.authUrl);
    } else {
      setError(response.error.message);
    }

    setIsLoading(false);
  };

  return (
    <div>
      <h1>Plex Setup</h1>
      {!authUrl && !isLoading && !error && (
        <Button
          onClick={() => {
            fetchAuthUrl();
          }}
        >
          Get Auth URL
        </Button>
      )}

      {authUrl && !isLoading && !error && (
        <Button asChild>
          <a href={authUrl}>Authenticate with Plex</a>
        </Button>
      )}

      {error && !isLoading && <p>{error}</p>}
      {isLoading && <p>Loading...</p>}
    </div>
  );
}
