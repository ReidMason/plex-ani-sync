import React, { useState } from 'react'
import Button from '../components/Button';
import Input from '../components/Input';

export default function AnilistSetup() {
    const [clientId, setClientId] = useState<string>("");
    const [instructionsVisible, setInstructionsVisible] = useState<boolean>(false);

    const domain = window.location.protocol + '//' + window.location.host;

    const redirectToAuthPage = () => {
        window.location.href = `https://anilist.co/api/v2/oauth/authorize?client_id=${clientId}&response_type=token`;
    }

    return (
        <div className="bg-gray-700 pt-24 h-screen flex flex-col gap-4 items-center p-4">
            <h1 className="text-4xl">Anilist setup</h1>
            <div className="flex flex-col gap-4">
                <li className="flex flex-col justify-center items-center">
                    <span className="text-xl text-center mb-2">Enter the ID of your Anilist api client</span>
                    <Input placeholder="Client Id" value={clientId} setValue={setClientId} />
                </li>

                <li className="flex justify-center">
                    <Button disabled={clientId == null} onClick={redirectToAuthPage}>Authorize application</Button>
                </li>
            </div>

            <span className="underline text-blue-400 cursor-pointer" onClick={() => (setInstructionsVisible(true))}>How to create an Anilist API client?</span>
            {instructionsVisible &&
                <ul className="bg-gray-400 flex flex-col gap-2 rounded px-4 py-2 list-decimal list-inside">
                    <li>Go to the Anilist&nbsp;
                        <a className="text-blue-400 underline" href="https://anilist.co/settings/developer" target="_blank" rel="noreferrer">developer portal</a>
                    </li>

                    <li>Click "create new client"</li>
                    <li>Set the name as "Plex ani sync"</li>
                    <li>Set the redirect url as "{domain}/setup/anilist-setup/auth-code"</li>
                </ul>}
        </div >
    )
}
