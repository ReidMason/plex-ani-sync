import React, { useState } from 'react'
import { AnilistAnime } from '../interfaces/Interfaces';
import AnilistService from '../services/AnilistService';
import Button from './Button'

export default function DeletePlanningList() {
    const [planningListToRemove, setPlanningListToRemove] = useState<Array<AnilistAnime>>([]);
    const [loading, setLoading] = useState(false);

    const getPlanningAnimeToRemove = () => {
        setLoading(true);
        AnilistService.getPlanningAnimeToRemove().then((response) => {
            setPlanningListToRemove(response.data);
        }).finally(() => {
            setLoading(false);
        })
    }

    return (
        <div>
            <Button loading={loading} onClick={getPlanningAnimeToRemove}>Delete planning list</Button>
            {(!loading && planningListToRemove.length > 0) &&
                <div>
                    <p> Are you sure?
                        This will delete: {planningListToRemove.length} entries</p>
                </div>
            }
            {loading &&
                <p>Calculating anime to remove...</p>}
        </div >
    )
}
