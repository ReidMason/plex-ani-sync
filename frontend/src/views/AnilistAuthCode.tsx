import { useLocation, useHistory } from "react-router-dom";
import LoadingSpinner from '../components/LoadingSpinner';
import AnilistService from '../services/AnilistService';

function getHashParams(hash: string) {
    const hashParams: any = {};
    var e;
    const a = /\+/g,  // Regex for replacing addition symbol with a space
        r = /([^&;=]+)=?([^&;]*)/g,
        d = function (s: any) { return decodeURIComponent(s.replace(a, " ")); },
        q = hash.substring(1);

    e = r.exec(q)
    while (e) {
        hashParams[d(e[1])] = d(e[2]);
        e = r.exec(q)
    }


    return hashParams;
}

export default function AnilistAuthCode() {
    const history = useHistory();

    const { hash } = useLocation();
    const hashParams = getHashParams(hash);

    console.log("In auth code component");


    const accessToken = hashParams.access_token
    if (accessToken)
        AnilistService.setAnilistToken(accessToken).then(() => {
            console.log("Saved anilist token");

            history.push("/");
        });

    return (
        <div>
            <div className="flex items-center justify-center w-48 h-48">
                <h1>Saving Anilist api token</h1>
                <LoadingSpinner />
            </div>
        </div>
    )
}
