import { useState } from "react";

interface PinDisplayProps {
    pin: string;
}

export default function PinDisplay({ pin }: PinDisplayProps) {
    const [copyMessage, setCopyMessage] = useState("");

    const copyPin = () => {
        const clipboard = navigator.clipboard;
        if (clipboard) {
            clipboard.writeText(pin).then(function () {
            }).then(() => {
                setCopyMessage("Pin copied to clipboard")
            })
                .catch(() => {
                    setCopyMessage("Failed to copy to clipboard")
                    console.log("Failed to copy to clipboard");
                });
        } else {
            console.log("Failed to copy to clipboard");
            setCopyMessage("Failed to copy to clipboard")
        }

    }

    return (
        <div>
            <div className="flex items-center justify-center mb-5">
                <p className="text-lg absolute">{copyMessage}</p>
            </div>
            <div
                className="flex flex-col gap-1 w-full items-center justify-center"
                onClick={copyPin}
            >
                <p className="border border-indigo-100 bg-indigo-200 rounded p-2 text-3xl font-semibold text-center tracking-widest">{pin}</p>
                {/* {pinArray.map((pinChar: string, index: number) => (
                <p className="border border-indigo-100 bg-indigo-200 rounded p-2 text-3xl font-semibold w-14 text-center" key={index}>{pinChar.trim()}</p>
            ))} */}
            </div>
        </div>

    )
}
