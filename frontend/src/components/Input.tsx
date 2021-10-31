import React from 'react'

interface InputProps {
    placeholder?: string;
    value: string | number;
    setValue: Function;
    type?: "text" | "password" | "number";
    disabled?: boolean;
    autoComplete?: string;
    className?: string;
    labelText?: string;
}


export default function Input({ placeholder, value, setValue, type, disabled, autoComplete, className, labelText }: InputProps) {
    const inputType = type ?? "text";
    const inputDisabled = disabled ?? false
    const inputAutocomplete = autoComplete ?? "off";

    return (
        <div className="flex flex-col gap-1">
            <label className={`text-lg ml-1 text-snow-1 transition-all transform translate-y-0 opacity-100 peer-placeholder-shown:opacity-0 peer-placeholder-shown:translate-y-10 peer-focus:opacity-100 peer-focus:translate-y-0`}>{labelText}</label>
            <input className={`${className} peer z-10 text-snow-3 p-2 rounded bg-night-4 border border-night-4 disabled:cursor-not-allowed disabled:opacity-50`} type={inputType} placeholder={placeholder} disabled={inputDisabled} value={value} onChange={(e) => (setValue(e.target.value))} autoComplete={inputAutocomplete} />
        </div>
    )
}
