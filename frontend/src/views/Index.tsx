import React, { useEffect, useState } from 'react'
import { io } from 'socket.io-client';
import Button from '../components/Button';
import SchedulerService from '../services/SchedulerService';
import { getBaseUrl } from '../utils';
import SyncProcessLog from '../components/SyncProcessLog';
import { ProcessLog, ProcessLogs } from '../interfaces/Interfaces';
import EnterTransition from '../components/EnterTransition';
import DeletePlanningList from '../components/DeletePlanningList';

const logListTransition = {
    hidden: 'transition duration-300 transform -translate-y-3 opacity-0',
    visible: 'transition duration-300 transform translate-y-0 opacity-100',
}

export default function Index() {
    const [nextRunTime, setNextRunTime] = useState<string>("");
    const [syncRunning, setSyncrunning] = useState<boolean>(false);
    const [processedShowsLog, setProcessedShowsLog] = useState<Array<ProcessLog>>([]);

    const update_things = () => {
        SchedulerService.getNextRunTime().then((response) => {
            setNextRunTime(response.data.nextRunTime);
            setSyncrunning(response.data.syncRunning);
        })
    }

    useEffect(() => {
        var socket = io(getBaseUrl());

        socket.on('sync_process_logs', function (response: ProcessLogs) {
            setProcessedShowsLog(response.updates.reverse().slice(0, 5));
            setSyncrunning(response.syncIsRunning);
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
        <div className="bg-gray-700 h-full pt-24">
            <EnterTransition className="flex flex-col items-center gap-4">
                <h1 className="text-center text-4xl font-semibold">Plex Anilist sync</h1>
                {nextRunTime &&
                    <div className="text-center">
                        <h2 className="text-2xl">Next run time</h2>
                        <h2 className="text-2xl">{nextRunTime}</h2>
                    </div>
                }

                <Button loading={syncRunning} onClick={startSync}>Sync now</Button>
                <div className={`w-3/12 flex flex-col items-center ${syncRunning ? logListTransition.visible : logListTransition.hidden}`}>
                    <h2 className="text-2xl">Sync running</h2>
                    <SyncProcessLog processedShowsLog={processedShowsLog} />
                </div>

                {/* <DeletePlanningList /> */}
            </EnterTransition>
        </div>
    )
}
