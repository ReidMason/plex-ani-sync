import React from 'react'
import { ProcessLog } from '../interfaces/Interfaces';
import { CSSTransition, TransitionGroup } from 'react-transition-group'

const classNames = {
    enter: 'transition duration-300 transform translate-y-3 opacity-0',
    enterActive: 'transition duration-300 transform translate-y-0 opacity-100',
    enterDone: 'transition duration-300 transform translate-y-0 opacity-100',
    exit: 'transition duration-300 opacity-0',
}

interface SyncProcessingLogProps {
    processedShowsLog: Array<ProcessLog>
}

export default function SyncProcessLog({ processedShowsLog }: SyncProcessingLogProps) {
    return (
        <div className="flex flex-col w-full text-center relative">
            <div className="absolute top-0 bg-gradient-to-b from-gray-700 h-3/6 w-full z-10"></div>
            <TransitionGroup component="div" className="flex flex-col-reverse h-32">
                {processedShowsLog.map((syncLog) => (
                    <CSSTransition key={syncLog.seriesTitle} timeout={150} classNames={classNames}>
                        <p key={syncLog.seriesTitle} className="truncate overflow-ellipsis text-gray-300">{syncLog.seriesTitle}</p>
                    </CSSTransition>
                ))}
            </TransitionGroup>
        </div>
    )
}
