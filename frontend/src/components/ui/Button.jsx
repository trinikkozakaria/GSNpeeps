const variants = {
  primary: "bg-cyan-700 text-white hover:bg-cyan-800 focus-visible:outline-cyan-700",
  secondary:
    "border border-slate-900/15 bg-slate-900/5 text-slate-900 hover:bg-slate-900/10 focus-visible:outline-slate-900",
};

export const Button = ({
  children,
  className = "",
  variant = "primary",
  type = "button",
  ...props
}) => (
  <button
    type={type}
    className={`inline-flex min-h-11 items-center justify-center rounded-lg px-4 py-2 text-sm font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${className}`}
    {...props}
  >
    {children}
  </button>
);

