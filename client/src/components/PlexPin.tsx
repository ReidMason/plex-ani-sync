import {getPlexPin} from "@/lib/api/plexAuth/PlexAuthApi";
import {useEffect, useState} from "react";

export default function PlexPin() {
  const [pinId, setPinId] = useState<string>("");

  useEffect(() => {
    const fetchPin = async () => {
      const urlParams = new URLSearchParams(window.location.search);
      const pinIdFromUrl = urlParams.get("pinId");

      if (pinIdFromUrl) {
        setPinId(pinIdFromUrl);
        const response = await getPlexPin(pinIdFromUrl);
        if (response.success) {
          console.log(response.data);
        } else {
          console.log(response.error);
        }
      }
    };

    fetchPin();
  }, []);

  return (
    <div>
      <h1>Plex Pin</h1>
      {pinId ? <p>Pin ID: {pinId}</p> : <p>No pin ID found in URL</p>}
    </div>
  );
}
