import { useEffect, useRef } from "react";

import { Button } from "../../../components/ui/Button";
import { calculateAge, formatDate } from "../../../lib/format";
import {
  DefinitionList,
  DetailItem,
  DetailSection,
} from "../../employees/components/DetailSection";
import { useEmployeeDetail } from "../../employees/hooks/useEmployees";

const genderLabel = { L: "Laki-laki", P: "Perempuan" };
const maritalLabel = { lajang: "Lajang", menikah: "Menikah", cerai: "Cerai" };

export const OrganizationEmployeeDetailModal = ({ employeeId, employeeName, onClose }) => {
  const dialogRef = useRef(null);
  const closeButtonRef = useRef(null);
  const detail = useEmployeeDetail("hr", employeeId);

  useEffect(() => {
    const previousFocus = document.activeElement;
    closeButtonRef.current?.focus();

    return () => previousFocus?.focus();
  }, []);

  const handleKeyDown = (event) => {
    if (event.key === "Escape") {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== "Tab") return;

    const focusable = dialogRef.current?.querySelectorAll(
      "button:not([disabled]), [href], [tabindex]:not([tabindex='-1'])",
    );
    if (!focusable || focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };

  const employee = detail.data;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="organization-employee-detail-title"
        onClick={(event) => event.stopPropagation()}
        onKeyDown={handleKeyDown}
        className="max-h-[90vh] w-full max-w-4xl overflow-y-auto rounded-xl border border-slate-900/10 bg-white p-6 shadow-xl"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-cyan-700">
              Detail karyawan
            </p>
            <h2
              id="organization-employee-detail-title"
              className="mt-1 text-xl font-bold text-slate-900"
            >
              {employee?.nama ?? employeeName}
            </h2>
          </div>
          <Button
            ref={closeButtonRef}
            variant="secondary"
            className="min-w-10 px-3"
            aria-label="Tutup detail karyawan"
            onClick={onClose}
          >
            ×
          </Button>
        </div>

        {detail.isPending && (
          <p role="status" className="mt-6 text-sm text-slate-600">
            Memuat detail karyawan…
          </p>
        )}

        {detail.isError && (
          <div role="alert" className="mt-6 rounded-lg border border-red-400/30 bg-red-400/10 p-4 text-sm text-red-700">
            <p>Detail karyawan belum dapat dimuat. {detail.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => detail.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}

        {employee && (
          <div className="mt-6">
            <DetailSection title="Identitas dan pekerjaan">
              <DefinitionList>
                <DetailItem label="NIP">{employee.nip}</DetailItem>
                <DetailItem label="Email">{employee.email}</DetailItem>
                <DetailItem label="Jenis kelamin">
                  {genderLabel[employee.jenis_kelamin]}
                </DetailItem>
                <DetailItem label="Tanggal lahir">{formatDate(employee.tanggal_lahir)}</DetailItem>
                <DetailItem label="Usia">{calculateAge(employee.tanggal_lahir)} tahun</DetailItem>
                <DetailItem label="Tanggal bergabung">{formatDate(employee.tanggal_join)}</DetailItem>
                <DetailItem label="Status pernikahan">
                  {employee.status_pernikahan
                    ? maritalLabel[employee.status_pernikahan]
                    : null}
                </DetailItem>
                <DetailItem label="Departemen">{employee.departemen}</DetailItem>
                <DetailItem label="Jabatan">{employee.jabatan}</DetailItem>
              </DefinitionList>
            </DetailSection>
          </div>
        )}
      </div>
    </div>
  );
};
