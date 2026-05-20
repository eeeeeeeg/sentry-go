import { Input, Select } from "@headlessui/react";
import clsx from "clsx";
import { type InputHTMLAttributes, type SelectHTMLAttributes } from "react";

const fieldClass =
  "h-9 min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

export function TextInput({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <Input className={clsx(fieldClass, className)} {...props} />;
}

export function SelectInput({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <Select className={clsx(fieldClass, className)} {...props} />;
}
