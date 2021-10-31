import React, { useState } from 'react'
import { useHistory } from 'react-router';
import Button from '../components/Button';
import Input from '../components/Input';
import PlexService from '../services/PlexService';

export default function PlexSetupServerUrl() {
    const domain = window.location.protocol + '//' + window.location.hostname + ":32400";
    const [plexServerUrl, setPlexServerUrl] = useState<string>(domain);
    const [loading, setLoading] = useState<boolean>(false);

    const history = useHistory();

    const savePlexUrl = () => {
        setLoading(true);
        PlexService.setPlexServerUrl(plexServerUrl.trim()).then(() => {
            history.push("/");
        }).finally(() => {
            setLoading(false);
        })
    }

    return (
        <div className="bg-gray-700 h-full flex justify-center pb-36 items-center flex-col gap-4">
            <h1 className="text-3xl text-center">Enter your Plex server URL</h1>
            <div className="w-full px-8 flex justify-center">
                <Input disabled={loading} placeholder="Plex server URL" value={plexServerUrl} setValue={setPlexServerUrl} />

                {/* <input disabled={loading} value={plexServerUrl} onChange={updatePlexServerUrl} className="w-full max-w-lg px-2 py-1 bg-gray-200 border border-gray-800 shadow-inner rounded"></input> */}
            </div>
            <Button loading={loading} onClick={savePlexUrl}>Save Plex URL</Button>
        </div>
    )
}
