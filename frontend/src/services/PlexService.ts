import axios, { AxiosResponse } from 'axios';

interface plexPinResponse {
    pin: string
}

interface plexAuthenticatedReponse {
    plexAuthenticated: boolean,
    token: string | null
}

const baseUrl = "http://10.128.0.101:5000"

const PlexService = {
    getPlexPin(): Promise<AxiosResponse<plexPinResponse>> {
        return axios.get<plexPinResponse>(`${baseUrl}/api/plex/getPin`);
    },

    plexAuthenticated(): Promise<AxiosResponse<plexAuthenticatedReponse>> {
        return axios.get<plexAuthenticatedReponse>(`${baseUrl}/api/plex/plexAuthenticated`)
    }
}

export default PlexService
