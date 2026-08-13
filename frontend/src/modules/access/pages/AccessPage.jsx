import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { Button } from "../../../components/ui/Button";
import { roleLabel } from "../../../routes/navigation/navigation";
import { useAuth } from "../../auth/hooks/useAuth";
import { PermissionMatrix } from "../components/PermissionMatrix";
import { usePermissionMatrix, useRoles, useUpdatePermission } from "../hooks/useAccess";
import { readableAction, readableModule } from "../utils/access-labels";

const monitoringRoles = ["hr", "top_management"];

const permissionKey = (permission) =>
  `${permission.role_id}:${permission.modul}:${permission.aksi}`;

export const AccessPage = () => {
  document.title = "AKSES — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const [pendingChange, setPendingChange] = useState(null);
  const [actionError, setActionError] = useState("");

  // Karyawan dan Atasan tidak pernah memicu permintaan modul AKSES, bahkan ketika membuka
  // URL secara langsung sebelum route guard sempat mengalihkan.
  const canRead = monitoringRoles.includes(auth.role);
  const canEdit = auth.role === "hr";

  const roles = useRoles(canRead);
  const permissions = usePermissionMatrix(canRead);
  const updatePermission = useUpdatePermission();

  const selectedRoleId = params.get("role");
  const activeRole =
    roles.data?.find((role) => role.id === selectedRoleId) ?? roles.data?.[0] ?? null;

  useEffect(() => {
    // Menyimpan role terpilih di URL agar tautan matriks dapat dibagikan.
    if (activeRole && activeRole.id !== selectedRoleId) {
      const next = new URLSearchParams(params);
      next.set("role", activeRole.id);
      setParams(next, { replace: true });
    }
  }, [activeRole, params, selectedRoleId, setParams]);

  const confirmToggle = async () => {
    if (!pendingChange) return;
    setActionError("");
    try {
      await updatePermission.mutateAsync({
        role_id: pendingChange.role_id,
        modul: pendingChange.modul,
        aksi: pendingChange.aksi,
        is_allowed: !pendingChange.is_allowed,
      });
      setPendingChange(null);
    } catch (error) {
      setActionError(describePermissionError(error));
    }
  };

  if (!canRead) {
    return (
      <section aria-labelledby="access-title">
        <h1 id="access-title" className="text-3xl font-bold">
          AKSES
        </h1>
        <p role="alert" className="mt-4 text-slate-600">
          Modul ini hanya tersedia untuk HR dan Top Management.
        </p>
      </section>
    );
  }

  return (
    <section aria-labelledby="access-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">
        Administrasi
      </p>
      <h1 id="access-title" className="mt-2 text-3xl font-bold">
        AKSES
      </h1>
      <p className="mt-2 text-slate-600">
        Empat role sistem beserta kapabilitasnya.
        {canEdit
          ? " Perubahan berlaku segera dan tercatat pada Audit Log."
          : " Akses Anda bersifat pemantauan saja."}
      </p>

      <div className="mt-7" aria-live="polite">
        {roles.isPending && (
          <p role="status" className="text-slate-600">
            Memuat daftar role…
          </p>
        )}
        {roles.isError && (
          <div
            role="alert"
            className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700"
          >
            <p>Daftar role belum dapat dimuat. {roles.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => roles.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}

        {roles.data && roles.data.length === 0 && (
          <div className="rounded-xl border border-slate-900/10 p-8 text-center text-slate-600">
            Belum ada role terdaftar.
          </div>
        )}

        {roles.data && roles.data.length > 0 && (
          <>
            <ul className="grid gap-3 sm:grid-cols-2">
              {roles.data.map((role) => (
                <li
                  key={role.id}
                  className="rounded-xl border border-slate-900/10 bg-slate-50 p-4"
                >
                  <p className="font-semibold text-slate-900">{roleLabel[role.nama] ?? role.nama}</p>
                  <p className="mt-1 text-sm text-slate-600">{role.deskripsi}</p>
                </li>
              ))}
            </ul>

            <div
              className="mt-8 flex flex-wrap gap-2"
              role="group"
              aria-label="Pilih role untuk matriks permission"
            >
              {roles.data.map((role) => (
                <Button
                  key={role.id}
                  variant={activeRole?.id === role.id ? "primary" : "secondary"}
                  aria-pressed={activeRole?.id === role.id}
                  onClick={() => {
                    const next = new URLSearchParams(params);
                    next.set("role", role.id);
                    setParams(next);
                  }}
                >
                  {roleLabel[role.nama] ?? role.nama}
                </Button>
              ))}
            </div>
          </>
        )}
      </div>

      <h2 className="mt-8 text-xl font-bold">
        Matriks permission {activeRole ? `— ${roleLabel[activeRole.nama] ?? activeRole.nama}` : ""}
      </h2>

      <div className="mt-4" aria-live="polite">
        {permissions.isPending && (
          <p role="status" className="text-slate-600">
            Memuat matriks permission…
          </p>
        )}
        {permissions.isError && (
          <div
            role="alert"
            className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700"
          >
            <p>Matriks permission belum dapat dimuat. {permissions.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => permissions.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {permissions.data && activeRole && (
          <PermissionMatrix
            permissions={permissions.data}
            roleId={activeRole.id}
            canEdit={canEdit}
            pendingKey={
              updatePermission.isPending && pendingChange ? permissionKey(pendingChange) : null
            }
            onToggle={(permission) => {
              setActionError("");
              setPendingChange(permission);
            }}
          />
        )}
      </div>

      <ConfirmDialog
        open={Boolean(pendingChange)}
        title={
          pendingChange?.is_allowed ? "Cabut kapabilitas ini?" : "Izinkan kapabilitas ini?"
        }
        description={
          pendingChange
            ? `${readableAction(pendingChange.aksi)} pada modul ${readableModule(
                pendingChange.modul,
              )} untuk role ${roleLabel[activeRole?.nama] ?? activeRole?.nama ?? ""}. Perubahan berlaku segera bagi seluruh pengguna role tersebut.`
            : ""
        }
        confirmLabel={pendingChange?.is_allowed ? "Cabut" : "Izinkan"}
        destructive={Boolean(pendingChange?.is_allowed)}
        busy={updatePermission.isPending}
        error={actionError}
        onConfirm={confirmToggle}
        onCancel={() => {
          setPendingChange(null);
          setActionError("");
        }}
      />
    </section>
  );
};

/** Menerjemahkan kegagalan server menjadi kalimat yang dapat ditindaklanjuti HR. */
const describePermissionError = (error) => {
  if (error?.status === 403) {
    return "Hanya HR yang dapat mengubah permission.";
  }
  if (error?.status === 404) {
    return "Role tidak ditemukan. Muat ulang halaman lalu coba lagi.";
  }
  if (error?.status === 409) {
    return "Matriks sudah berubah di tempat lain. Muat ulang halaman lalu coba lagi.";
  }
  if (error?.status === 422) {
    return (
      error?.fields?.is_allowed ??
      "Perubahan ini melanggar batasan akses produk dan tidak dapat disimpan."
    );
  }
  return error?.message ?? "Permission gagal diperbarui.";
};
