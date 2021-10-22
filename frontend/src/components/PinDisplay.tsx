interface PinDisplayProps {
    pin: string;
}

export default function PinDisplay({ pin }: PinDisplayProps) {
    const pinArray = pin.split('');

    const copyPin = () => {
        navigator.clipboard.writeText(pin).then(function () {
            console.log("Pin successfully copied");

        }, function (err) {
            console.error('Could not copy pin: ', err);
        });
    }

    return (
        <div
            className="flex gap-1 w-full items-center justify-center"
            onClick={copyPin}
        >
            {pinArray.map((pinChar: string, index: number) => (
                <span className="border border-indigo-100 bg-indigo-200 rounded p-2 text-3xl font-semibold w-14 text-center" key={index}>{pinChar}</span>
            ))}
            {/* <div className="absolute ml-80">
                <button className="border border-gray-500 rounded px-4 py-2" onClick={copyPin}>
                    <motion.div
                        animate={{ rotate: pinCopied ? 360 : 0 }}
                    >
                        {pinCopied ?
                            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                            </svg>
                            :
                            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                            </svg>
                        }
                    </motion.div>
                </button>
            </div> */}
        </div>
    )
}
