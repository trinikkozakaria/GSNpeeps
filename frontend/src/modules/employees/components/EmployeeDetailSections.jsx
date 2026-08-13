import { calculateAge, formatCurrency, formatDate, formatPeriod } from "../../../lib/format";
import { DefinitionList, DetailItem, DetailSection, EmptyNote } from "./DetailSection";

const genderLabel = { L: "Laki-laki", P: "Perempuan" };
const maritalLabel = { lajang: "Lajang", menikah: "Menikah", cerai: "Cerai" };
const contractStatusLabel = { aktif: "Aktif", berakhir: "Berakhir" };
const bpjsLabel = { kesehatan: "BPJS Kesehatan", ketenagakerjaan: "BPJS Ketenagakerjaan" };

/**
 * Bagian Point 1-11 detail karyawan. Dipakai bersama oleh halaman detail HR dan Profil Saya
 * karena keduanya memakai schema EmployeeDetail yang sama.
 *
 * Setiap bagian hanya dirender bila response memuat datanya, sehingga field yang tidak
 * diizinkan untuk suatu role tidak pernah muncul sebagai bagian kosong yang membingungkan.
 */
export const EmployeeDetailSections = ({ employee }) => (
  <div className="mt-7 grid gap-6">
    <DetailSection title="Identitas dan pekerjaan">
      <DefinitionList>
        <DetailItem label="NIP">{employee.nip}</DetailItem>
        <DetailItem label="Email">{employee.email}</DetailItem>
        <DetailItem label="Jenis kelamin">{genderLabel[employee.jenis_kelamin]}</DetailItem>
        <DetailItem label="Tanggal lahir">{formatDate(employee.tanggal_lahir)}</DetailItem>
        <DetailItem label="Usia">{calculateAge(employee.tanggal_lahir)} tahun</DetailItem>
        <DetailItem label="Tanggal bergabung">{formatDate(employee.tanggal_join)}</DetailItem>
        <DetailItem label="Status pernikahan">
          {employee.status_pernikahan ? maritalLabel[employee.status_pernikahan] : null}
        </DetailItem>
        <DetailItem label="Departemen">{employee.departemen}</DetailItem>
        <DetailItem label="Jabatan">{employee.jabatan}</DetailItem>
        <DetailItem label="Nomor KTP">{employee.ktp?.nomor_ktp}</DetailItem>
      </DefinitionList>
    </DetailSection>

    <DetailSection title="Alamat">
      {employee.alamat ? (
        <DefinitionList columns={2}>
          <DetailItem label="Jalan">{employee.alamat.jalan}</DetailItem>
          <DetailItem label="Kelurahan">{employee.alamat.kelurahan}</DetailItem>
          <DetailItem label="Kecamatan">{employee.alamat.kecamatan}</DetailItem>
          <DetailItem label="Kota">{employee.alamat.kota}</DetailItem>
          <DetailItem label="Provinsi">{employee.alamat.provinsi}</DetailItem>
        </DefinitionList>
      ) : (
        <EmptyNote>Alamat belum diisi.</EmptyNote>
      )}
    </DetailSection>

    <DetailSection title="Kontrak" description="Seluruh riwayat kontrak, terbaru lebih dahulu.">
      {employee.kontrak.length === 0 ? (
        <EmptyNote>Belum ada riwayat kontrak.</EmptyNote>
      ) : (
        <ul className="divide-y divide-slate-900/10">
          {employee.kontrak.map((contract) => (
            <li key={contract.nomor_kontrak} className="grid gap-2 py-4 sm:grid-cols-4">
              <span className="font-semibold text-slate-900">{contract.nomor_kontrak}</span>
              <span className="text-slate-600">{contract.jenis_kontrak}</span>
              <span className="text-slate-600">
                {formatDate(contract.tanggal_mulai)} — {formatDate(contract.tanggal_berakhir)}
              </span>
              <span className="text-slate-600">
                {contractStatusLabel[contract.status] ?? contract.status}
              </span>
            </li>
          ))}
        </ul>
      )}
    </DetailSection>

    <DetailSection title="BPJS dan NPWP">
      {employee.bpjs.length === 0 && !employee.npwp ? (
        <EmptyNote>Data BPJS dan NPWP belum diisi.</EmptyNote>
      ) : (
        <DefinitionList columns={2}>
          {employee.bpjs.map((item) => (
            <DetailItem key={item.jenis} label={bpjsLabel[item.jenis] ?? item.jenis}>
              {item.nomor}
            </DetailItem>
          ))}
          {employee.npwp && <DetailItem label="NPWP">{employee.npwp.nomor_npwp}</DetailItem>}
        </DefinitionList>
      )}
    </DetailSection>

    <DetailSection title="Kontak darurat">
      {employee.kontak_darurat.length === 0 ? (
        <EmptyNote>Kontak darurat belum diisi.</EmptyNote>
      ) : (
        <ul className="divide-y divide-slate-900/10">
          {employee.kontak_darurat.map((contact) => (
            <li key={`${contact.nama}-${contact.nomor_telepon}`} className="grid gap-2 py-4 sm:grid-cols-3">
              <span className="font-semibold text-slate-900">{contact.nama}</span>
              <span className="text-slate-600">{contact.hubungan || "Hubungan belum diisi"}</span>
              <span className="text-slate-600">{contact.nomor_telepon}</span>
            </li>
          ))}
        </ul>
      )}
    </DetailSection>

    <DetailSection title="Pendidikan">
      {employee.pendidikan.length === 0 ? (
        <EmptyNote>Riwayat pendidikan belum diisi.</EmptyNote>
      ) : (
        <ul className="divide-y divide-slate-900/10">
          {employee.pendidikan.map((education, index) => (
            <li
              key={`${education.jenjang ?? "jenjang"}-${education.institusi ?? index}`}
              className="grid gap-2 py-4 sm:grid-cols-3"
            >
              <span className="font-semibold text-slate-900">{education.jenjang || "Jenjang belum diisi"}</span>
              <span className="text-slate-600">{education.institusi || "Institusi belum diisi"}</span>
              <span className="text-slate-600">
                {education.tahun_masuk ? `Masuk ${education.tahun_masuk} · ` : ""}
                {education.tahun_lulus ? `Lulus ${education.tahun_lulus}` : "Sedang pendidikan"}
              </span>
            </li>
          ))}
        </ul>
      )}
    </DetailSection>

    <DetailSection title="Riwayat jabatan">
      {employee.riwayat_jabatan.length === 0 ? (
        <EmptyNote>Riwayat jabatan belum diisi.</EmptyNote>
      ) : (
        <ul className="divide-y divide-slate-900/10">
          {employee.riwayat_jabatan.map((history, index) => (
            <li key={`${history.tanggal_mulai}-${index}`} className="grid gap-2 py-4 sm:grid-cols-3">
              <span className="font-semibold text-slate-900">
                {history.jabatan?.nama || "Jabatan belum tercatat"}
              </span>
              <span className="text-slate-600">
                {history.departemen?.nama || "Departemen belum tercatat"}
              </span>
              <span className="text-slate-600">
                {formatDate(history.tanggal_mulai)} —{" "}
                {history.tanggal_selesai ? formatDate(history.tanggal_selesai) : "sekarang"}
              </span>
            </li>
          ))}
        </ul>
      )}
    </DetailSection>

    <DetailSection
      title="Gaji bulan berjalan"
      description="Hanya periode bulan berjalan yang ditampilkan; riwayat gaji penuh tidak tersedia di sini."
    >
      {employee.gaji_berjalan ? (
        <DefinitionList columns={2}>
          <DetailItem label="Periode">{formatPeriod(employee.gaji_berjalan.periode)}</DetailItem>
          <DetailItem label="Gaji pokok">{formatCurrency(employee.gaji_berjalan.gaji_pokok)}</DetailItem>
          <DetailItem label="Tunjangan">{formatCurrency(employee.gaji_berjalan.tunjangan)}</DetailItem>
          <DetailItem label="Potongan">{formatCurrency(employee.gaji_berjalan.potongan)}</DetailItem>
          <DetailItem label="Take home pay">
            {formatCurrency(employee.gaji_berjalan.take_home_pay)}
          </DetailItem>
        </DefinitionList>
      ) : (
        <EmptyNote>Belum ada data gaji untuk bulan berjalan.</EmptyNote>
      )}
    </DetailSection>
  </div>
);
