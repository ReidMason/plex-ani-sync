import React, { useEffect, useState } from 'react'

interface EnterTransitionProps {
    children: React.ReactNode;
    className?: String;
    duration?: string;
}

export default function EnterTransition({ children, className, duration }: EnterTransitionProps) {
    const [triggerAnimations, setTriggerAnimations] = useState<boolean>(false)
    duration = duration ?? "duration-150"

    useEffect(() => {
        setTimeout(() => (setTriggerAnimations(true)), 100);
    }, []);

    return (
        <div className={`transform ${duration} ${className} transition ${triggerAnimations ? "opacity-100 translate-y-0" : "opacity-0 translate-y-6"}`}>
            {children}
        </div>
    )
}
