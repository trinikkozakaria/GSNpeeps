import { useAuth } from "../../auth/hooks/useAuth";
import { EmployeeStatusBadge } from "../../employees/components/EmployeeStatusBadge";
import { useEmployeeDetail } from "../../employees/hooks/useEmployees";

const labelGender = { L: "Laki-laki", P: "Perempuan" };

const DetailItem = ({ label, children }) => (
  <div>
    <dt className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</dt>
    <dd className="mt-1 text-sm text-slate-100">{children || "Belum diisi"}</dd>
  </div>
);

export const ProfilSayaPage = () => {
  const auth = useAuth();
  const employee = useEmployeeDetail(auth.role, auth.user.employee_id);

  if (employee.isPending) {
    return <p role="status" className="text-slate-300">Memuat profil…</p>;
  }
  if (employee.isError) {
    return (
      <section role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-5 text-red-100">
        <h1 className="text-xl font-bold">Profil tidak dapat dibuka</h1>
        <p className="mt-2">{employee.error.message}</p>
      </section>
    );
  }

  const data = employee.data;
  document.title = `${data.nama} — GSNpeeps`;

  return (
    <section aria-labelledby="profil-saya-title">
      <div className="mt-1 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm text-slate-400">{data.nip}</p>
          <h1 id="profil-saya-title" className="mt-1 text-3xl font-bold">{data.nama}</h1>
          <p className="mt-2 text-slate-300">{data.jabatan || "Jabatan belum ditetapkan"} · {data.departemen || "Departemen belum ditetapkan"}</p>
        </div>
        <EmployeeStatusBadge status={data.status} />
      </div>

      <div className="mt-7 grid gap-6">
        <section className="rounded-xl border border-white/10 bg-white/[0.03] p-5" aria-labelledby="identity-heading">
          <h2 id="identity-heading" className="text-lg font-bold">Identitas dan pekerjaan</h2>
          <dl className="mt-5 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            <DetailItem label="Email">{data.email}</DetailItem>
            <DetailItem label="Jenis kelamin">{labelGender[data.jenis_kelamin]}</DetailItem>
            <DetailItem label="Tanggal lahir">{data.tanggal_lahir}</DetailItem>
            <DetailItem label="Tanggal bergabung">{data.tanggal_join}</DetailItem>
            <DetailItem label="Status pernikahan">{data.status_pernikahan}</DetailItem>
            <DetailItem label="Nomor KTP">{data.ktp?.nomor_ktp}</DetailItem>
          </dl>
        </section>

        <section className="rounded-xl border border-white/10 bg-white/[0.03] p-5" aria-labelledby="address-heading">
          <h2 id="address-heading" className="text-lg font-bold">Alamat</h2>
          {data.alamat ? (
            <p className="mt-4 text-slate-300">
              {[data.alamat.jalan, data.alamat.kelurahan, data.alamat.kecamatan, data.alamat.kota, data.alamat.provinsi]
                .filter(Boolean)
                .join(", ")}
            </p>
          ) : (
            <p className="mt-4 text-slate-400">Alamat belum diisi.</p>
          )}
        </section>

        <section className="rounded-xl border border-white/10 bg-white/[0.03] p-5" aria-labelledby="contract-heading">
          <h2 id="contract-heading" className="text-lg font-bold">Kontrak</h2>
          {data.kontrak.length === 0 ? (
            <p className="mt-4 text-slate-400">Belum ada riwayat kontrak.</p>
          ) : (
            <ul className="mt-4 divide-y divide-white/10">
              {data.kontrak.map((contract) => (
                <li key={contract.nomor_kontrak} className="grid gap-2 py-4 sm:grid-cols-3">
                  <span className="font-semibold">{contract.nomor_kontrak}</span>
                  <span className="text-slate-300">{contract.jenis_kontrak}</span>
                  <span className="text-slate-300">{contract.tanggal_mulai} — {contract.tanggal_berakhir}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </section>
  );
};