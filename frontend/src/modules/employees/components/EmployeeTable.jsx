import { Link, useLocation } from "react-router-dom";

import { EmployeeStatusBadge } from "./EmployeeStatusBadge";

export const EmployeeTable = ({ employees }) => {
  const location = useLocation();

  return (
    <div className="overflow-x-auto rounded-xl border border-white/10" role="region" aria-label="Daftar karyawan">
      <table className="min-w-full divide-y divide-white/10 text-left text-sm">
        <thead className="bg-white/5 text-xs uppercase tracking-wider text-slate-400">
          <tr>
            <th scope="col" className="px-4 py-3">Karyawan</th>
            <th scope="col" className="px-4 py-3">Departemen</th>
            <th scope="col" className="px-4 py-3">Jabatan</th>
            <th scope="col" className="px-4 py-3">Status</th>
            <th scope="col" className="px-4 py-3"><span className="sr-only">Aksi</span></th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/10">
          {employees.map((employee) => (
            <tr key={employee.id} className="bg-slate-900/40">
              <td className="px-4 py-4">
                <p className="font-semibold text-white">{employee.nama}</p>
                <p className="mt-1 text-xs text-slate-400">{employee.nip} · {employee.email || "Email belum tersedia"}</p>
              </td>
              <td className="px-4 py-4 text-slate-300">{employee.departemen || "Belum ditetapkan"}</td>
              <td className="px-4 py-4 text-slate-300">{employee.jabatan || "Belum ditetapkan"}</td>
              <td className="px-4 py-4"><EmployeeStatusBadge status={employee.status} /></td>
              <td className="px-4 py-4 text-right">
                <Link
                  to={`/app/karyawan/${employee.id}`}
                  state={{ from: `${location.pathname}${location.search}` }}
                  className="font-semibold text-cyan-300 hover:text-cyan-200"
                >
                  Lihat detail
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
