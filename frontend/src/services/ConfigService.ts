import axios, { AxiosResponse } from 'axios';
import Config from '../interfaces/Config';
import { createRequestUrl } from '../utils';

const baseUrl = process.env.REACT_APP_API_BASE_URL

const ConfigService = {
    getconfig(): Promise<AxiosResponse<Config>> {
        return axios.get<Config>(createRequestUrl("/api/config/getConfig"));
    }
}

export default ConfigService
