import { useId } from "react";

export const DetailSection = ({ title, description, children }) => {
  const headingId = useId();
  return (
    <section
      className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5"
      aria-labelledby={headingId}
    >
      <h2 id={headingId} className="text-lg font-bold">
        {title}
      </h2>
      {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      <div className="mt-5">{children}</div>
    </section>
  );
};

export const DefinitionList = ({ children, columns = 3 }) => (
  <dl
    className={`grid gap-5 sm:grid-cols-2 ${columns === 3 ? "lg:grid-cols-3" : ""}`}
  >
    {children}
  </dl>
);

export const DetailItem = ({ label, children }) => (
  <div>
    <dt className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</dt>
    <dd className="mt-1 text-sm text-slate-900">
      {children === null || children === undefined || children === "" ? (
        <span className="text-slate-500">Belum diisi</span>
      ) : (
        children
      )}
    </dd>
  </div>
);

export const EmptyNote = ({ children }) => <p className="text-sm text-slate-500">{children}</p>;
