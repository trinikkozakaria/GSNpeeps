const variants = {
  primary: "bg-cyan-300 text-slate-950 hover:bg-cyan-200 focus-visible:outline-cyan-300",
  secondary:
    "border border-white/15 bg-white/5 text-white hover:bg-white/10 focus-visible:outline-white",
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

