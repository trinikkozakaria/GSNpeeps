import { Link, useLocation } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { EmployeeStatusBadge } from "./EmployeeStatusBadge";

export const EmployeeTable = ({ employees }) => {
  const location = useLocation();
  const from = `${location.pathname}${location.search}`;

  const columns = [
    {
      key: "karyawan",
      header: "Karyawan",
      render: (employee) => (
        <>
          <span className="block font-semibold text-white">{employee.nama}</span>
          <span className="mt-1 block text-xs text-slate-400">
            {employee.nip} · {employee.email || "Email belum tersedia"}
          </span>
        </>
      ),
    },
    {
      key: "departemen",
      header: "Departemen",
      cellClassName: "text-slate-300",
      render: (employee) => employee.departemen || "Belum ditetapkan",
    },
    {
      key: "jabatan",
      header: "Jabatan",
      cellClassName: "text-slate-300",
      render: (employee) => employee.jabatan || "Belum ditetapkan",
    },
    {
      key: "status",
      header: "Status",
      render: (employee) => <EmployeeStatusBadge status={employee.status} />,
    },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (employee) => (
        <Link
          to={`/app/karyawan/${employee.id}`}
          state={{ from }}
          className="font-semibold text-cyan-300 hover:text-cyan-200"
        >
          Lihat detail
          <span className="sr-only"> {employee.nama}</span>
        </Link>
      ),
    },
  ];

  return (
    <DataTable
      caption="Daftar karyawan"
      columns={columns}
      rows={employees}
      rowKey={(employee) => employee.id}
      emptyMessage="Belum ada data karyawan."
    />
  );
};
