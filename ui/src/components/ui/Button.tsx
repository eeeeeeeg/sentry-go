import { Button as HeadlessButton } from "@headlessui/react";
import clsx from "clsx";
import { type ButtonHTMLAttributes } from "react";

type ButtonVariant = "default" | "primary" | "ghost";

export function Button({ className, variant = "default", ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }) {
  return (
    <HeadlessButton
      className={clsx(
        "inline-flex h-9 items-center justify-center gap-2 rounded-md px-3 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50",
        variant === "primary" && "bg-slate-900 font-semibold text-white hover:bg-slate-800",
        variant === "default" && "border border-slate-300 bg-white text-slate-700 hover:bg-slate-50",
        variant === "ghost" && "text-slate-600 hover:bg-slate-100",
        className,
      )}
      {...props}
    />
  );
}
