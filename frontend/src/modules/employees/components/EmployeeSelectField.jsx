export const EmployeeSelectField = ({
  id,
  label,
  registration,
  error,
  children,
  disabled,
  description,
}) => {
  const describedBy = [
    description ? `${id}-description` : "",
    error ? `${id}-error` : "",
  ].filter(Boolean).join(" ") || undefined;

  return (
    <div>
      <label htmlFor={id} className="mb-2 block text-sm font-medium text-slate-700">{label}</label>
      <select
        id={id}
        aria-invalid={Boolean(error)}
        aria-describedby={describedBy}
        disabled={disabled}
        className="min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300 disabled:opacity-60"
        {...registration}
      >
        {children}
      </select>
      {description && <p id={`${id}-description`} className="mt-2 text-xs text-slate-500">{description}</p>}
      {error && <p id={`${id}-error`} role="alert" className="mt-2 text-sm text-rose-700">{error}</p>}
    </div>
  );
};
