import { Button } from "../../../components/ui/Button";
import { useAuth } from "../../auth/hooks/useAuth";
import { EmployeeDetailSections } from "../../employees/components/EmployeeDetailSections";
import { EmployeeStatusBadge } from "../../employees/components/EmployeeStatusBadge";
import { ProtectedImage } from "../../../components/media/ProtectedImage";
import { useMyProfile } from "../hooks/useProfile";

/**
 * Profil Saya bersifat read-only. Tidak ada tombol edit maupun pengajuan perubahan data
 * karena self-service edit berada di luar scope; perubahan data ditempuh melalui HR.
 */
export const MyProfilePage = () => {
  document.title = "Profil Saya — GSNpeeps";
  const auth = useAuth();
  const profile = useMyProfile(auth.user?.id);

  return (
    <section aria-labelledby="my-profile-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Data pribadi</p>
      <h1 id="my-profile-title" className="mt-2 text-3xl font-bold">
        Profil Saya
      </h1>
      <p className="mt-2 max-w-2xl text-slate-600">
        Data ini hanya dapat dibaca. Jika ada informasi yang perlu diperbaiki, ajukan
        perubahan melalui HR agar tercatat pada Audit Log.
      </p>

      <div className="mt-6" aria-live="polite">
        {profile.isPending && (
          <p role="status" className="text-slate-600">
            Memuat profil…
          </p>
        )}
        {profile.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Profil belum dapat dimuat. {profile.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => profile.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {profile.data && (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <span className="h-16 w-16 overflow-hidden rounded-full bg-slate-100">{profile.data.foto_profil_url ? <ProtectedImage path={profile.data.foto_profil_url} alt={`Foto profil ${profile.data.nama}`} className="h-full w-full object-cover" /> : null}</span>
              <h2 className="text-2xl font-bold">{profile.data.nama}</h2>
              <EmployeeStatusBadge status={profile.data.status} />
            </div>
            <p className="mt-1 text-slate-600">
              {profile.data.jabatan || "Jabatan belum ditetapkan"} ·{" "}
              {profile.data.departemen || "Departemen belum ditetapkan"}
            </p>
            <EmployeeDetailSections employee={profile.data} />
          </>
        )}
      </div>
    </section>
  );
};
