import axios, { AxiosResponse } from 'axios';
import { createRequestUrl } from '../utils';

interface plexPinResponse {
    pin: string
}

interface plexAuthenticatedReponse {
    plexAuthenticated: boolean,
    token: string | null
}

interface TokenFilledResponse {
    tokenFilled: boolean
}

interface ServerUrlFilledResponse {
    serverUrlFilled: boolean
}


const PlexService = {
    getPlexPin(): Promise<AxiosResponse<plexPinResponse>> {
        return axios.get<plexPinResponse>(createRequestUrl("/api/plex/getPin"));
    },

    plexAuthenticated(): Promise<AxiosResponse<plexAuthenticatedReponse>> {
        return axios.get<plexAuthenticatedReponse>(createRequestUrl("/api/plex/plexAuthenticated"))
    },

    tokenFilled(): Promise<AxiosResponse<TokenFilledResponse>> {
        return axios.get<TokenFilledResponse>(createRequestUrl("/api/plex/tokenFilled"))
    },

    serverUrlFilled(): Promise<AxiosResponse<ServerUrlFilledResponse>> {
        return axios.get<ServerUrlFilledResponse>(createRequestUrl("/api/plex/serverUrlFilled"))
    },

    setPlexServerUrl(server_url: string): Promise<AxiosResponse> {
        return axios.post(createRequestUrl("/api/plex/setPlexServerUrl"), { server_url })
    }
}

export default PlexService
