/**
 * Org chart dirender sebagai daftar bersarang sehingga hubungan atasan-bawahan dapat
 * ditelusuri screen reader dan keyboard tanpa memerlukan grafik terpisah.
 */
const OrganizationBranch = ({ nodes }) => (
  <ul className="ml-4 border-l border-slate-900/10 pl-4">
    {nodes.map((node) => (
      <li key={node.employee_id} className="mt-3">
        <div className="rounded-lg border border-slate-900/10 bg-slate-900/[0.03] p-3">
          <p className="font-semibold text-slate-900">{node.nama}</p>
          <p className="text-xs text-slate-500">
            {node.jabatan || "Jabatan belum ditetapkan"} ·{" "}
            {node.departemen || "Departemen belum ditetapkan"}
          </p>
        </div>
        {node.bawahan.length > 0 && <OrganizationBranch nodes={node.bawahan} />}
      </li>
    ))}
  </ul>
);

const countMembers = (nodes) =>
  nodes.reduce((total, node) => total + 1 + countMembers(node.bawahan), 0);

export const OrganizationChart = ({ nodes }) => {
  if (nodes.length === 0) {
    return (
      <p className="text-sm text-slate-500">
        Belum ada karyawan aktif untuk menyusun struktur organisasi pada periode ini.
      </p>
    );
  }

  return (
    <div>
      <p className="text-sm text-slate-500">
        {countMembers(nodes)} karyawan aktif dalam {nodes.length} jalur pelaporan teratas.
      </p>
      <div className="mt-4 overflow-x-auto">
        <OrganizationBranch nodes={nodes} />
      </div>
    </div>
  );
};
