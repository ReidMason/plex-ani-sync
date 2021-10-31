import axios, { AxiosResponse } from 'axios';
import { createRequestUrl } from '../utils';

interface anilistAuthenticatedReponse {
    anilistAuthenticated: boolean,
    token: string | null
}

const AnilistService = {
    setAnilistToken(token: string) {
        return axios.post(createRequestUrl("/api/anilist/setAnilistToken"), { token });
    },

    anilistAuthenticated(): Promise<AxiosResponse<anilistAuthenticatedReponse>> {
        return axios.get<anilistAuthenticatedReponse>(createRequestUrl("/api/anilist/anilistAuthenticated"))
    }
}

export default AnilistService
