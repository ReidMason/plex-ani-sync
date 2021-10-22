import axios, { AxiosResponse } from 'axios';

interface anilistAuthenticatedReponse {
    anilistAuthenticated: boolean,
    token: string | null
}

const baseUrl = "http://10.128.0.101:5000"

const AnilistService = {
    setAnilistToken(token: string) {
        return axios.post(`${baseUrl}/api/anilist/setAnilistToken`, { token });
    },

    anilistAuthenticated(): Promise<AxiosResponse<anilistAuthenticatedReponse>> {
        return axios.get<anilistAuthenticatedReponse>(`${baseUrl}/api/anilist/anilistAuthenticated`)
    }
}

export default AnilistService
