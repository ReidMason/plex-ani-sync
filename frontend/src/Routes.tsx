import { Switch, Route } from 'react-router-dom';
import MainRoutes from './routes/MainRoutes';
import SetupRoutes from './routes/SetupRoutes';

// Nest the routes properly
// We only want to run the auth check for anilist and plex on the main routes
// So don't run the auth check on the setup routes

export default function Routes() {
    return (
        <Switch>
            <Route path="/setup">
                <SetupRoutes />
            </Route>
            <MainRoutes />
        </Switch>
    )
}
