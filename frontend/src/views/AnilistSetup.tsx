import React, { useState } from 'react'

export default function AnilistSetup() {
    const [clientId, setClientId] = useState<string | null>(null);

    console.log("In setup component");

    const domain = window.location.hostname;

    const updateClientId = (e: React.ChangeEvent<HTMLInputElement>) => {
        setClientId(e.target.value);
    }

    return (
        <div className="bg-gray-500">
            <p>1. Go to the Anilist
                <a href="https://anilist.co/settings/developer" target="_blank" rel="noreferrer"> developer portal</a>
            </p>

            <p>2. Click "create new client"</p>
            <p>3. Set the name as "Plex ani sync"</p>
            <p>4. Set the redirect url as "http://{domain}/api/anilist/codeRedirect"</p>
            <div>
                <p>5. Enter the client Id:</p>
                <input className="border border-gray-800 ml-4" type="text" onChange={updateClientId} />
            </div>

            <a href={`https://anilist.co/api/v2/oauth/authorize?client_id=${clientId}&response_type=token`}>6. Authorize application </a>
        </div>
    )
}
