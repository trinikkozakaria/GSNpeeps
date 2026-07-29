import { forwardRef } from "react";

export const Input = forwardRef(({ className = "", ...props }, ref) => (
  <input
    ref={ref}
    className={`min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 py-2 text-white outline-none placeholder:text-slate-500 focus:border-cyan-300 focus:ring-2 focus:ring-cyan-300/20 disabled:cursor-not-allowed disabled:opacity-60 ${className}`}
    {...props}
  />
));

Input.displayName = "Input";

