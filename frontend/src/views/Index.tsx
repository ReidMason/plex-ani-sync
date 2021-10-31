import React, { useEffect, useState } from 'react'
import Button from '../components/Button';
import SchedulerService from '../services/SchedulerService';

export default function Index() {
    const [nextRunTime, setNextRunTime] = useState<string>("");
    const [syncRunning, setSyncrunning] = useState<boolean>(false);

    const update_things = () => {
        SchedulerService.getNextRunTime().then((response) => {
            setNextRunTime(response.data.nextRunTime);
            setSyncrunning(response.data.syncRunning);
        })
    }

    useEffect(() => {
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
        </div>
    )
}
