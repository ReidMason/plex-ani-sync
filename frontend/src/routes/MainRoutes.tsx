import { useEffect, useState } from 'react';
import { Route, Switch, useHistory } from 'react-router'
import LoadingSpinner from '../components/LoadingSpinner';
import AnilistService from '../services/AnilistService';
import PlexService from '../services/PlexService';
import Index from '../views/Index';

export default function MainRoutes() {
    const [loading, setLoading] = useState<boolean>(true);
    const history = useHistory();

    useEffect(() => {
        const checkPlexAuthenticated = () => {
            return PlexService.tokenFilled().then((response) => {
                if (!response.data.tokenFilled)
                    history.push("/setup/plex-setup")
                return !response.data.tokenFilled;
            })
        }

        const checkPlexServerUrlFilled = () => {
            return PlexService.serverUrlFilled().then((response) => {
                if (!response.data.serverUrlFilled)
                    history.push("/setup/plex-setup-server-url");
                return !response.data.serverUrlFilled;
            })
        }

        const checkAnilistAuthenticated = () => {
            return AnilistService.anilistAuthenticated().then((response) => {
                if (!response.data.anilistAuthenticated)
                    history.push("/setup/anilist-setup")
            })
        }

        checkPlexServerUrlFilled().then((redirecred: boolean) => {
            if (!redirecred) {
                checkPlexAuthenticated().then((redirected: boolean) => {
                    if (!redirected) {
                        checkAnilistAuthenticated().finally(() => {
                            setLoading(false);
                        });
                    } else {
                        setLoading(false);
                    }
                });
            } else {
                setLoading(false);
            }
        })


    }, [history])

    return (
        <>
            {loading ?
                <div className="bg-gray-700 flex items-center justify-center h-full">
                    <div className="w-24 h-24">
                        <LoadingSpinner />
                    </div>
                </div>
                :
                <Switch>
                    <Route exact path="/">
                        <Index />
                    </Route>
                </Switch>
            }
        </>

    )
}
