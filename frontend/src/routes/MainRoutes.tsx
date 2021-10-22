import { useEffect, useState } from 'react';
import { Route, Switch, useHistory } from 'react-router'
import LoadingSpinner from '../components/LoadingSpinner';
import AnilistService from '../services/AnilistService';
import PlexService from '../services/PlexService';

export default function MainRoutes() {
    const [loading, setLoading] = useState<boolean>(true);
    const history = useHistory();

    useEffect(() => {
        const checkPlexAuthenticated = () => {
            return PlexService.plexAuthenticated().then((response) => {
                if (!response.data.plexAuthenticated)
                    history.push("/setup/plex-setup")
            })
        }

        const checkAnilistAuthenticated = () => {
            return AnilistService.anilistAuthenticated().then((response) => {
                if (!response.data.anilistAuthenticated)
                    history.push("/setup/anilist-setup")
            })
        }

        checkPlexAuthenticated().finally(() => {
            checkAnilistAuthenticated().finally(() => {
                setLoading(false);
            });
        });
    }, [])

    return (
        <>
            {loading ?
                <div className="bg-gray-500 flex items-center justify-center h-full">
                    <div className="w-24 h-24">
                        <LoadingSpinner />
                    </div>
                </div>
                :
                <Switch>
                    <Route exact path="/">
                        <div className="bg-gray-600">
                            <h1>Index</h1>
                        </div>
                    </Route>
                </Switch>
            }
        </>

    )
}
