import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import {
  useApprovalRequestDetail,
  useCancelApprovalRequest,
  useDecideApprovalRequest,
} from "../hooks/useDocumentApproval";

const statusLabel = {
  berjalan: "Berjalan",
  disetujui: "Disetujui",
  ditolak: "Ditolak",
  dibatalkan: "Dibatalkan",
  bermasalah: "Bermasalah — tahap aktif tidak memiliki pemegang posisi",
};

export const RequestDetailPage = () => {
  const { id } = useParams();
  document.title = "Detail Persetujuan Dokumen — GSNpeeps";
  const navigate = useNavigate();
  const detail = useApprovalRequestDetail(id);
  const decide = useDecideApprovalRequest(id);
  const cancel = useCancelApprovalRequest(id);

  const [note, setNote] = useState("");
  const [actionError, setActionError] = useState("");

  const handleDecide = async (keputusan) => {
    setActionError("");
    if (keputusan === "reject" && note.trim().length < 5) {
      setActionError("Catatan wajib diisi minimal 5 karakter saat menolak.");
      return;
    }
    try {
      await decide.mutateAsync({ keputusan, catatan: note.trim() });
      setNote("");
    } catch (error) {
      setActionError(error.message ?? "Keputusan belum dapat disimpan.");
    }
  };

  const handleCancel = async () => {
    setActionError("");
    try {
      await cancel.mutateAsync();
    } catch (error) {
      setActionError(error.message ?? "Pengajuan belum dapat dibatalkan.");
    }
  };

  if (detail.isPending) {
    return <p role="status" className="text-slate-600">Memuat detail pengajuan…</p>;
  }
  if (detail.isError) {
    return (
      <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
        <p>Detail pengajuan belum dapat dimuat. {detail.error.message}</p>
        <Button className="mt-3" variant="secondary" onClick={() => detail.refetch()}>
          Coba lagi
        </Button>
      </div>
    );
  }

  const data = detail.data;
  const isPending = data.status === "berjalan" || data.status === "bermasalah";
  const noDecisionsYet = data.riwayat.length === 0;

  return (
    <section aria-labelledby="request-detail-title" className="max-w-3xl">
      <Link to="/app/persetujuan-dokumen/saya" className="text-sm font-semibold text-cyan-700">
        ← Kembali ke pengajuan saya
      </Link>
      <h1 id="request-detail-title" className="mt-5 text-3xl font-bold">{data.judul}</h1>
      <p className="mt-1 text-slate-500">
        <a href={data.document_url} target="_blank" rel="noreferrer" className="text-cyan-700 hover:text-cyan-900">
          Buka dokumen ↗
        </a>
        {" · "}Diajukan {formatDate(data.created_at)}
      </p>
      <p className="mt-2 font-semibold">{statusLabel[data.status] ?? data.status}</p>

      {actionError && (
        <div role="alert" className="mt-4 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
          {actionError}
        </div>
      )}

      <div className="mt-6 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6">
        <h2 className="text-lg font-bold">Tahap alur — {data.template_nama}</h2>
        <ol className="mt-4 space-y-3">
          {data.tahap.map((stage) => (
            <li
              key={stage.urutan}
              className={`rounded-lg border p-3 ${
                stage.aktif ? "border-cyan-400 bg-cyan-50" : "border-slate-900/10"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">
                  {stage.urutan + 1}. {stage.nama_tahap}
                </span>
                <span className="text-sm text-slate-500">
                  {stage.selesai ? "Selesai" : stage.aktif ? "Aktif" : "Menunggu"}
                </span>
              </div>
              <p className="mt-1 text-sm text-slate-500">
                Posisi: {stage.position_nama ?? "—"}
                {" · "}
                Pemegang saat ini:{" "}
                {stage.pemegang_posisi.length > 0
                  ? stage.pemegang_posisi.map((holder) => holder.nama).join(", ")
                  : "Tidak ada (kosong)"}
              </p>
            </li>
          ))}
        </ol>
      </div>

      <div className="mt-6 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6">
        <h2 className="text-lg font-bold">Riwayat keputusan</h2>
        {data.riwayat.length === 0 ? (
          <p className="mt-2 text-slate-500">Belum ada keputusan.</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {data.riwayat.map((entry, index) => (
              <li key={index} className="text-sm">
                <span className="font-semibold">{entry.approver_nama}</span> —{" "}
                {entry.keputusan === "approve" ? "Menyetujui" : "Menolak"} tahap{" "}
                {entry.urutan_tahap + 1} pada {formatDate(entry.decided_at)}
                {entry.catatan && <span className="block text-slate-500">Catatan: {entry.catatan}</span>}
              </li>
            ))}
          </ul>
        )}
      </div>

      {isPending && (
        <div className="mt-6 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6">
          <h2 className="text-lg font-bold">Ambil keputusan</h2>
          <p className="mt-1 text-sm text-slate-500">
            Tombol ini terlihat untuk semua orang; server tetap menolak keputusan dari
            pengguna yang bukan approver tahap aktif.
          </p>
          <label htmlFor="decision-note" className="mt-4 block text-sm font-medium">
            Catatan (wajib saat menolak)
          </label>
          <textarea
            id="decision-note"
            value={note}
            onChange={(event) => setNote(event.target.value)}
            rows={3}
            className="mt-1 w-full rounded-lg border border-slate-300 p-2"
            disabled={decide.isPending}
          />
          <div className="mt-3 flex flex-wrap gap-3">
            <Button disabled={decide.isPending} onClick={() => handleDecide("approve")}>
              Setujui
            </Button>
            <Button variant="secondary" disabled={decide.isPending} onClick={() => handleDecide("reject")}>
              Tolak
            </Button>
          </div>
        </div>
      )}

      {isPending && noDecisionsYet && (
        <div className="mt-6">
          <Button variant="secondary" disabled={cancel.isPending} onClick={handleCancel}>
            {cancel.isPending ? "Membatalkan…" : "Batalkan Pengajuan"}
          </Button>
        </div>
      )}
    </section>
  );
};
