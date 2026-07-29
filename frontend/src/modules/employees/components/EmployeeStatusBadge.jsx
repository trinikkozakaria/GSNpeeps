export const EmployeeStatusBadge = ({ status }) => {
  const active = status === "aktif";
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
        active ? "bg-emerald-400/15 text-emerald-300" : "bg-slate-700 text-slate-300"
      }`}
    >
      {active ? "Aktif" : "Nonaktif"}
    </span>
  );
};
