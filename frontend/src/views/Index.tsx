import React, { useEffect, useState } from 'react'
import { io } from 'socket.io-client';
import Button from '../components/Button';
import SchedulerService from '../services/SchedulerService';
import { getBaseUrl } from '../utils';

interface SyncLog {
    seriesTitle: string;
}

export default function Index() {
    const [nextRunTime, setNextRunTime] = useState<string>("");
    const [syncRunning, setSyncrunning] = useState<boolean>(false);
    const [processedShowsLog, setProcessedShowsLog] = useState<Array<SyncLog>>([])

    const update_things = () => {
        SchedulerService.getNextRunTime().then((response) => {
            setNextRunTime(response.data.nextRunTime);
            setSyncrunning(response.data.syncRunning);
        })
    }

    useEffect(() => {
        var socket = io(getBaseUrl());

        socket.on('sync_process_logs', function (response) {
            setProcessedShowsLog(response.updates.reverse());
        })

        update_things();
        setInterval(update_things, 1000);
    }, [])

    const startSync = () => {
        SchedulerService.forceRunSync().then(() => {
            SchedulerService.getNextRunTime().then((response) => {
                setNextRunTime(response.data.nextRunTime);
                setSyncrunning(response.data.syncRunning);
            })
        });
    }

    return (
        <div className="bg-gray-700 h-full pt-24 flex flex-col items-center gap-4">
            <h1 className="text-center text-4xl font-semibold">Plex Anilist sync</h1>
            {nextRunTime &&
                <div className="text-center">
                    <h2 className="text-2xl">Next run time</h2>
                    <h2 className="text-2xl">{nextRunTime}</h2>
                </div>
            }
            {syncRunning && <h2 className="text-2xl">Sync running</h2>}

            <Button loading={syncRunning} onClick={startSync}>Sync now</Button>
            <div className="flex flex-col-reverse w-3/12 text-center h-52 relative">
                <div className="absolute top-0 bg-gradient-to-b from-gray-700 h-3/6 w-full"></div>
                {processedShowsLog.map((syncLog) => (
                    <p key={syncLog.seriesTitle} className="truncate overflow-ellipsis text-gray-300">{syncLog.seriesTitle}</p>
                ))}
            </div>
        </div>
    )
}
