import React, { useEffect, useState } from 'react'
import Button from '../components/Button';
import PinDisplay from '../components/PinDisplay'
import PlexService from '../services/PlexService';
import LoadingSpinner from '../components/LoadingSpinner';
import { useHistory } from 'react-router';

function sleep(ms: number) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

export default function PlexSetup() {
    const [plexPin, setPlexPin] = useState<string>("");
    const [pinAuthenticated, setPinAuthenticated] = useState<boolean>(false);

    const history = useHistory();

    useEffect(() => {
        const waitForPlexAuthentication = async () => {
            while (!plexTokenfilled) {
                const response = await PlexService.tokenFilled()
                plexTokenfilled = response.data.tokenFilled;
                if (plexTokenfilled) {
                    setPinAuthenticated(true);
                    setTimeout(() => (history.push("/")), 3000);
                    return;
                }
                await sleep(1000);
            }
        }

        const checkPinAuthenticated = async () => {
            const response = await PlexService.tokenFilled();
            pinAlreadyAuthenticated = response.data.tokenFilled;

            if (!pinAlreadyAuthenticated) {
                const response = await PlexService.getPlexPin();
                setPlexPin(response.data.pin);
            }
        }

        var pinAlreadyAuthenticated = false;
        var plexTokenfilled = false;

        checkPinAuthenticated();

        if (!pinAlreadyAuthenticated)
            waitForPlexAuthentication()
    }, [history])

    const openPlexLink = () => {
        window.open('https://www.plex.tv/link/', '_blank');
    }

    return (
        <div className="bg-gray-700 h-full flex justify-center items-center">
            <div className="w-full h-full text-center p-8">
                {pinAuthenticated ?
                    <div className="flex flex-col items-center justify-center h-full">
                        <h1 className="text-4xl">
                            Pin authenticated!
                        </h1>
                        <p>You will be redirected shortly...</p>
                    </div>
                    :
                    <div className="flex flex-col items-center">
                        <div className="flex items-center justify-center text-indigo-700 bg-indigo-300 rounded-full h-52 w-52 mb-4">
                            <svg xmlns="http://www.w3.org/2000/svg" className="h-40 w-40 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 11V7a4 4 0 118 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z" />
                            </svg>
                        </div>

                        <h1 className="text-4xl mb-6 font-semibold">Plex pin</h1>
                        <h1 className="text-xl mb-6">Enter the pin below to link your plex account</h1>

                        <div className="mb-4">
                            {plexPin ?
                                <PinDisplay pin={plexPin} />
                                :
                                <div className="w-14 h-14">
                                    <LoadingSpinner />
                                </div>
                            }
                        </div>

                        <Button onClick={openPlexLink}>Open Plex link</Button>
                    </div>
                }
            </div>
        </div>
    )
}
