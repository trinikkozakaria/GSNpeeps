import { useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";

import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { PhotoUploadField } from "../../../components/form/PhotoUploadField";
import { Button } from "../../../components/ui/Button";
import { useAuth } from "../../auth/hooks/useAuth";
import { EmployeeDetailSections } from "../components/EmployeeDetailSections";
import { EmployeeDocuments } from "../components/EmployeeDocuments";
import { EmployeeExportMenu } from "../components/EmployeeExportMenu";
import { EmployeeStatusBadge } from "../components/EmployeeStatusBadge";
import { useDeactivateEmployee, useEmployeeDetail, useResetEmployeePassword, useUploadEmployeePhoto } from "../hooks/useEmployees";

// Pesan tidak membedakan "tidak ada" dan "tidak boleh diakses" agar keberadaan record tidak
// bocor kepada role yang tidak berhak.
const detailErrorMessage = (error) => {
  if (error?.status === 403 || error?.status === 404) {
    return "Data karyawan tidak ditemukan atau tidak dapat diakses dengan hak akses Anda.";
  }
  return error?.message ?? "Detail karyawan belum dapat dimuat.";
};

export const EmployeeDetailPage = () => {
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const auth = useAuth();
  const employee = useEmployeeDetail(auth.role, id);
  const deactivate = useDeactivateEmployee(auth.role, id);
  const uploadPhoto = useUploadEmployeePhoto(auth.role, id);
	const resetPassword = useResetEmployeePassword(auth.role, id);
  const [actionError, setActionError] = useState("");
  const [isConfirmOpen, setConfirmOpen] = useState(false);
	const [resetOpen, setResetOpen] = useState(false);
	const [newPassword, setNewPassword] = useState("");
	const [passwordConfirmation, setPasswordConfirmation] = useState("");
  const isHR = auth.role === "hr";
  const backTarget = location.state?.from?.startsWith("/app/karyawan")
    ? location.state.from
    : "/app/karyawan";

  if (employee.isPending) {
    return (
      <p role="status" className="text-slate-600">
        Memuat detail karyawan…
      </p>
    );
  }
  if (employee.isError) {
    return (
      <section role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-5 text-red-700">
        <h1 className="text-xl font-bold">Detail tidak dapat dibuka</h1>
        <p className="mt-2">{detailErrorMessage(employee.error)}</p>
        <Link to={backTarget} className="mt-4 inline-block font-semibold text-cyan-700">
          Kembali ke daftar
        </Link>
      </section>
    );
  }

  const data = employee.data;
  const handleDeactivate = async () => {
    setActionError("");
    try {
      await deactivate.mutateAsync();
      setConfirmOpen(false);
      navigate("/app/karyawan?status=nonaktif", { replace: true });
    } catch (error) {
      setActionError(error.message);
    }
  };
	const handleResetPassword = async (event) => {
		event.preventDefault(); setActionError("");
		if (newPassword !== passwordConfirmation) { setActionError("Konfirmasi password tidak sama."); return; }
		try { await resetPassword.mutateAsync({ new_password: newPassword, new_password_confirmation: passwordConfirmation }); setNewPassword(""); setPasswordConfirmation(""); setResetOpen(false); } catch (error) { setActionError(error.message); }
	};
  document.title = `${data.nama} — GSNpeeps`;

  return (
    <section aria-labelledby="employee-detail-title">
      <Link to={backTarget} className="text-sm font-semibold text-cyan-700">
        ← Kembali ke daftar
      </Link>
      <div className="mt-5 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm text-slate-500">{data.nip}</p>
          <h1 id="employee-detail-title" className="mt-1 text-3xl font-bold">
            {data.nama}
          </h1>
          <p className="mt-2 text-slate-600">
            {data.jabatan || "Jabatan belum ditetapkan"} ·{" "}
            {data.departemen || "Departemen belum ditetapkan"}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <EmployeeStatusBadge status={data.status} />
          {isHR && (
            <>
              <Link
                to={`/app/karyawan/${id}/edit`}
                className="inline-flex min-h-10 items-center rounded-lg border border-slate-900/15 px-4 text-sm font-semibold"
              >
                Edit
              </Link>
				<Button variant="secondary" onClick={() => setResetOpen(true)}>Reset Password</Button>
              {data.status === "aktif" && (
                <Button variant="secondary" onClick={() => setConfirmOpen(true)}>
                  Nonaktifkan
                </Button>
              )}
            </>
          )}
        </div>
      </div>

      {isHR && (
        <div className="mt-5">
          <EmployeeExportMenu employeeId={id} label="Export karyawan ini" />
        </div>
      )}

      {isHR && (
        <div className="mt-6 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <h2 className="text-base font-semibold text-slate-900">Foto profil</h2>
          <div className="mt-4">
            <PhotoUploadField
              idPrefix="employee-photo"
              currentPhotoUrl={data.foto_profil_url}
              onUpload={async (file) => {
                const result = await uploadPhoto.mutateAsync(file);
                return result.foto_profil_url;
              }}
            />
          </div>
        </div>
      )}

      {actionError && !isConfirmOpen && (
        <p role="alert" className="mt-4 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
          {actionError}
        </p>
      )}

      <EmployeeDetailSections employee={data} />

      <div className="mt-6">
        <EmployeeDocuments scope={auth.role} employeeId={id} canUpload={isHR} />
      </div>

      <ConfirmDialog
        open={isConfirmOpen}
        title={`Nonaktifkan ${data.nama}?`}
        description="Karyawan dinonaktifkan, bukan dihapus permanen. Seluruh data dan riwayat tetap tersimpan, tetapi akun tidak dapat digunakan dan sesi aktif langsung dicabut."
        confirmLabel="Nonaktifkan karyawan"
        destructive
        busy={deactivate.isPending}
        error={actionError}
        onCancel={() => {
          setActionError("");
          setConfirmOpen(false);
        }}
        onConfirm={handleDeactivate}
      />
		{resetOpen && (
		  <div role="dialog" aria-modal="true" aria-labelledby="reset-password-title" className="fixed inset-0 z-50 grid place-items-center bg-slate-950/40 p-4">
			<form onSubmit={handleResetPassword} className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
			  <h2 id="reset-password-title" className="text-xl font-bold">Reset password {data.nama}</h2>
			  <p className="mt-2 text-sm text-slate-600">Sesi aktif karyawan akan dicabut dan akun terkunci akan dibuka.</p>
			  <label className="mt-4 block text-sm font-medium">Password baru<input type="password" minLength="12" maxLength="128" required value={newPassword} onChange={(event) => setNewPassword(event.target.value)} className="mt-1 block min-h-10 w-full rounded-lg border px-3" /></label>
			  <label className="mt-3 block text-sm font-medium">Konfirmasi password<input type="password" minLength="12" maxLength="128" required value={passwordConfirmation} onChange={(event) => setPasswordConfirmation(event.target.value)} className="mt-1 block min-h-10 w-full rounded-lg border px-3" /></label>
			  {actionError && <p role="alert" className="mt-3 text-sm text-rose-700">{actionError}</p>}
			  <div className="mt-5 flex justify-end gap-3"><Button type="button" variant="secondary" onClick={() => setResetOpen(false)}>Batal</Button><Button type="submit" disabled={resetPassword.isPending}>{resetPassword.isPending ? "Mereset…" : "Reset Password"}</Button></div>
			</form>
		  </div>
		)}
    </section>
  );
};
