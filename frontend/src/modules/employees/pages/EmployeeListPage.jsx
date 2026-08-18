import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { useAuth } from "../../auth/hooks/useAuth";
import { EmployeeExportMenu } from "../components/EmployeeExportMenu";
import { EmployeeBulkUpload } from "../components/EmployeeBulkUpload";
import { EmployeeTable } from "../components/EmployeeTable";
import { useDepartments, useEmployees } from "../hooks/useEmployees";

const parsePositiveInteger = (value, fallback) => {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
};

export const EmployeeListPage = () => {
  document.title = "Employee Database — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const [searchInput, setSearchInput] = useState(params.get("search") ?? "");
  const filters = useMemo(
    () => ({
      search: params.get("search") || undefined,
      department_id: params.get("department_id") || undefined,
      status: params.get("status") || undefined,
      page: parsePositiveInteger(params.get("page"), 1),
      limit: 10,
    }),
    [params],
  );
  const departments = useDepartments();
  const employees = useEmployees(auth.role, filters);

  useEffect(() => {
    const normalized = searchInput.trim();
    const timer = window.setTimeout(() => {
      setParams((current) => {
        const currentSearch = current.get("search") ?? "";
        if (normalized === currentSearch) return current;

        const next = new URLSearchParams(current);
        if (normalized) next.set("search", normalized);
        else next.delete("search");
        next.delete("page");
        return next;
      }, { replace: true });
    }, 350);
    return () => window.clearTimeout(timer);
  }, [searchInput, setParams]);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    if (key !== "page") next.delete("page");
    setParams(next);
  };

  const clearFilters = () => {
    setSearchInput("");
    setParams({});
  };

  const data = employees.data;
  const isFiltered = Boolean(filters.search || filters.department_id || filters.status);

  return (
    <section aria-labelledby="employee-title">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Organisasi</p>
          <h1 id="employee-title" className="mt-2 text-3xl font-bold">Employee Database</h1>
          <p className="mt-2 text-slate-600">
            Data aktif dan nonaktif dipisahkan melalui filter status. Top Management memiliki akses baca saja.
          </p>
        </div>
        {auth.role === "hr" && (
          <div className="flex flex-wrap items-center gap-3">
            <Link
              to="/app/karyawan/baru"
              className="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-700 px-4 py-2 text-sm font-semibold text-white hover:bg-cyan-800"
            >
              Tambah karyawan
            </Link>
            <EmployeeExportMenu filters={filters} label="Export hasil filter" />
            <EmployeeBulkUpload />
          </div>
        )}
      </div>

      <div className="mt-7 grid gap-4 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-4 md:grid-cols-3">
        <label className="text-sm font-medium text-slate-700">
          Cari nama atau NIP
          <input
            type="search"
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          Departemen
          <select
            value={filters.department_id ?? ""}
            onChange={(event) => setFilter("department_id", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          >
            <option value="">Semua departemen</option>
            {(departments.data ?? []).map((department) => (
              <option key={department.id} value={department.id}>{department.nama}</option>
            ))}
          </select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          Status
          <select
            value={filters.status ?? ""}
            onChange={(event) => setFilter("status", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          >
            <option value="">Semua status</option>
            <option value="aktif">Aktif</option>
            <option value="nonaktif">Nonaktif</option>
          </select>
        </label>
        {isFiltered && (
          <Button type="button" variant="secondary" onClick={clearFilters} className="justify-self-start">
            Hapus semua filter
          </Button>
        )}
      </div>

      <div className="mt-6" aria-live="polite">
        {employees.isPending && <p role="status" className="text-slate-600">Memuat data karyawan…</p>}
        {employees.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Data karyawan belum dapat dimuat. {employees.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => employees.refetch()}>Coba lagi</Button>
          </div>
        )}
        {data && data.items.length === 0 && (
          <div className="rounded-xl border border-slate-900/10 p-8 text-center text-slate-600">
            {isFiltered ? "Tidak ada karyawan yang cocok dengan filter." : "Belum ada data karyawan."}
          </div>
        )}
        {data && data.items.length > 0 && (
          <>
            <p className="mb-3 text-sm text-slate-500">{data.meta.total_data} karyawan ditemukan</p>
            <EmployeeTable employees={data.items} />
            <Pagination
              meta={data.meta}
              label="Navigasi halaman daftar karyawan"
              onPageChange={(page) => setFilter("page", String(page))}
            />
          </>
        )}
      </div>
    </section>
  );
};
