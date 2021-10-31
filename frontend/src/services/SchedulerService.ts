import axios, { AxiosResponse } from 'axios';
import { createRequestUrl } from '../utils'

interface NextRunTimeResponse {
    nextRunTime: string;
    syncRunning: boolean;
}


const SchedulerService = {
    getNextRunTime(): Promise<AxiosResponse<NextRunTimeResponse>> {
        return axios.get<NextRunTimeResponse>(createRequestUrl("/api/scheduler/getNextRunTime"));
    },

    forceRunSync(): Promise<AxiosResponse> {
        return axios.post(createRequestUrl("/api/scheduler/forceRunSync"));
    }
}

export default SchedulerService
