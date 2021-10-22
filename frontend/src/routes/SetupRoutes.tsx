import React from 'react'
import { Route, Switch, useRouteMatch } from 'react-router'
import AnilistAuthCode from '../views/AnilistAuthCode'
import AnilistSetup from '../views/AnilistSetup'
import PlexSetup from '../views/PlexSetup'

export default function SetupRoutes() {
    const match = useRouteMatch();

    return (
        <Switch>
            <Route path={`${match.url}/anilist-setup/auth-code`}>
                <AnilistAuthCode />
            </Route>
            <Route exact path={`${match.url}/anilist-setup`}>
                <AnilistSetup />
            </Route>
            <Route path={`${match.url}/plex-setup`}>
                <PlexSetup />
            </Route>
        </Switch>
    )
}
