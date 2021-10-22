import React from 'react'
import LoadingSpinner from './LoadingSpinner';

interface ButtonProps {
    children: React.ReactNode;
    disabled?: boolean;
    variant?: "dark" | "outline" | "text";
    rounded?: boolean;
    onClick?: Function;
    loading?: boolean;
    className?: string;
}

interface variantStyle {
    dark: string,
    outline: string,
    text: string,
}

const baseStyles: variantStyle = {
    dark: "text-grey-200 bg-indigo-500 ",
    outline: "text-grey-200 bg-transparent border border-2 border-grey-200",
    text: "text-grey-200 bg-transparent",
}

const hoverStyles: variantStyle = {
    dark: "hover:bg-indigo-300",
    outline: "hover:bg-blue-300",
    text: "hover:bg-blue-300",
}

const baseDisabledStyles = "disabled:opacity-50 disabled:cursor-not-allowed";
const disabledStyles: variantStyle = {
    dark: `${baseDisabledStyles}`,
    outline: `${baseDisabledStyles}`,
    text: `${baseDisabledStyles}`,
}

const activeStyles: variantStyle = {
    dark: "active:bg-indigo-300",
    outline: "active:bg-blue-300",
    text: "",
}

export default function Button({ children, disabled, variant, rounded, onClick, loading, className }: ButtonProps) {
    // Disable the button if it's loading
    if (loading)
        disabled = true;

    const variantTheme = variant ?? "dark";
    const roundingStyle = rounded ? "rounded-full" : "rounded-md";
    const baseStyle = baseStyles[variantTheme];
    // Disabled hover styling when button is disabled
    const hoverStyle = !disabled ? hoverStyles[variantTheme] : "";
    const disabledStyle = disabledStyles[variantTheme];
    const activeStyle = activeStyles[variantTheme];
    const variantStyle = `${baseStyle} ${hoverStyle} ${disabledStyle} ${activeStyle}`;

    const onClickFunction = (e: React.MouseEvent) => {
        if (onClick)
            onClick(e)
    }

    return (
        <div className={className}>
            <button
                className={`p-2 px-4 flex justify-center ${variantStyle} ${roundingStyle}`}
                disabled={disabled}
                onClick={onClickFunction}
            >
                <div className="flex items-center gap-2">
                    {loading &&
                        <LoadingSpinner />
                    }
                    {children}
                </div>
            </button >
        </div>
    )
}
