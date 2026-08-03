import { Button } from "../../../components/ui/Button";
import { groupPermissionsByModule, readableAction } from "../utils/access-labels";

/**
 * Matriks permission satu role.
 *
 * Ketika `canEdit` bernilai false, kontrol mutation tidak dirender sama sekali, bukan sekadar
 * dinonaktifkan. Ini menghindari kesan bahwa Top Management memiliki kewenangan yang sedang
 * terkunci sementara. Backend tetap menjadi penentu: UI ini hanya cermin kebijakan.
 */
export const PermissionMatrix = ({ permissions, roleId, canEdit, pendingKey, onToggle }) => {
  const groups = groupPermissionsByModule(permissions, roleId);

  if (groups.length === 0) {
    return (
      <div className="rounded-xl border border-white/10 p-8 text-center text-slate-300">
        Belum ada kapabilitas terdaftar untuk role ini.
      </div>
    );
  }

  return (
    <ul className="grid gap-3">
      {groups.map((group) => (
        <li key={group.modul} className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
          <h3 className="text-sm font-semibold uppercase tracking-wider text-cyan-200">
            {group.label}
          </h3>
          <ul className="mt-3 grid gap-2 sm:grid-cols-2">
            {group.actions.map((permission) => {
              const key = `${permission.role_id}:${permission.modul}:${permission.aksi}`;
              const isPending = pendingKey === key;
              return (
                <li
                  key={permission.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-white/10 px-3 py-2"
                >
                  <span className="text-sm text-slate-200">{readableAction(permission.aksi)}</span>
                  <span className="flex items-center gap-3">
                    {/* Status ditulis sebagai teks dan ikon, tidak hanya diwakili warna. */}
                    <span
                      className={`text-sm font-semibold ${
                        permission.is_allowed ? "text-emerald-200" : "text-slate-400"
                      }`}
                    >
                      <span aria-hidden="true">{permission.is_allowed ? "✓ " : "✕ "}</span>
                      {permission.is_allowed ? "Diizinkan" : "Tidak diizinkan"}
                    </span>
                    {canEdit && (
                      <Button
                        variant="secondary"
                        disabled={Boolean(pendingKey)}
                        onClick={() => onToggle(permission)}
                      >
                        {isPending
                          ? "Menyimpan…"
                          : permission.is_allowed
                            ? "Cabut"
                            : "Izinkan"}
                      </Button>
                    )}
                  </span>
                </li>
              );
            })}
          </ul>
        </li>
      ))}
    </ul>
  );
};
