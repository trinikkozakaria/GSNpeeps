import { useState } from "react";
import { Link, useParams } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { isPendingStatus } from "../../../lib/request-status";
import { useAuth } from "../../auth/hooks/useAuth";
import {
  useDecideLeaveRequest,
  useDelegateLeaveRequest,
  useLeaveRequestDetail,
} from "../../leave/hooks/useLeave";
import { useDecideOvertimeRequest, useOvertimeDetail } from "../../overtime/hooks/useOvertime";
import { ApprovalTimeline } from "../components/ApprovalTimeline";
import { DecisionDialog } from "../components/DecisionDialog";
import { RequestStatusBadge } from "../components/RequestStatusBadge";

// Tahap aktif yang boleh diputus tiap role. Backend tetap otoritas; ini hanya UX.
const decidableStatusForRole = {
  atasan: "menunggu_atasan",
  hr: "menunggu_hr",
  top_management: "menunggu_top_management",
};

const Field = ({ label, children }) => (
  <div className="border-b border-slate-900/10 py-3">
    <dt className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</dt>
    <dd className="mt-1 text-sm text-slate-900">{children ?? "—"}</dd>
  </div>
);

export const ApprovalDetailPage = ({ kind }) => {
  const { id } = useParams();
  const auth = useAuth();
  const isLeave = kind === "ketidakhadiran";

  const leaveDetail = useLeaveRequestDetail(auth.role, isLeave ? id : undefined);
  const overtimeDetail = useOvertimeDetail(auth.role, isLeave ? undefined : id);
  const detail = isLeave ? leaveDetail : overtimeDetail;

  const decideLeave = useDecideLeaveRequest(auth.role, id);
  const delegateLeave = useDelegateLeaveRequest(auth.role, id);
  const decideOvertime = useDecideOvertimeRequest(auth.role, id);

  const [dialogMode, setDialogMode] = useState(null);
  const [dialogError, setDialogError] = useState("");
  const [notice, setNotice] = useState("");

  const data = detail.data;
  const canDecide = Boolean(data) && data.status === decidableStatusForRole[auth.role];
  // Delegasi hanya tersedia untuk ketidakhadiran; kontrak tidak menyediakan delegasi lembur.
  const canDelegate = isLeave && canDecide && auth.role === "atasan";
  const busy = decideLeave.isPending || delegateLeave.isPending || decideOvertime.isPending;

  const handleDecision = async (note) => {
    setDialogError("");
    try {
      if (dialogMode === "delegate") {
        await delegateLeave.mutateAsync({ delegate_to: data.employee_id, catatan: note });
        setNotice("Pengajuan telah didelegasikan ke HR.");
      } else {
        const payload = { keputusan: dialogMode, catatan: note || undefined };
        if (isLeave) await decideLeave.mutateAsync(payload);
        else await decideOvertime.mutateAsync(payload);
        setNotice(
          dialogMode === "setujui" ? "Keputusan persetujuan tersimpan." : "Pengajuan telah ditolak.",
        );
      }
      setDialogMode(null);
    } catch (error) {
      if (error?.status === 409) {
        // Pengajuan sudah diputus pihak lain; muat ulang agar status terbaru terlihat.
        setDialogMode(null);
        setNotice(
          "Pengajuan ini sudah diproses oleh pihak lain. Detail dimuat ulang dengan status terbaru.",
        );
        await detail.refetch();
        return;
      }
      setDialogError(error?.message ?? "Keputusan belum dapat disimpan.");
    }
  };

  if (detail.isPending) {
    return (
      <p role="status" className="text-slate-600">
        Memuat detail pengajuan…
      </p>
    );
  }
  if (detail.isError) {
    return (
      <section role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-5 text-red-700">
        <h1 className="text-xl font-bold">Pengajuan tidak dapat dibuka</h1>
        <p className="mt-2">
          Pengajuan tidak ditemukan atau tidak dapat diakses dengan hak akses Anda.
        </p>
        <Link to="/app/persetujuan" className="mt-4 inline-block font-semibold text-cyan-700">
          Kembali ke antrean
        </Link>
      </section>
    );
  }

  return (
    <section aria-labelledby="approval-detail-title">
      <Link to="/app/persetujuan" className="text-sm font-semibold text-cyan-700">
        ← Kembali ke antrean
      </Link>
      <div className="mt-5 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 id="approval-detail-title" className="text-3xl font-bold">
            {data.nama_karyawan}
          </h1>
          <p className="mt-2 text-slate-600">
            {isLeave ? data.jenis_izin : "Pengajuan lembur"}
          </p>
        </div>
        <RequestStatusBadge status={data.status} />
      </div>

      {notice && (
        <p role="status" className="mt-5 rounded-lg border border-cyan-300/30 bg-cyan-300/10 p-4 text-sm text-cyan-800">
          {notice}
        </p>
      )}

      <div className="mt-7 grid gap-6 lg:grid-cols-2">
        <section aria-labelledby="request-detail-heading" className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <h2 id="request-detail-heading" className="text-lg font-bold">
            Detail pengajuan
          </h2>
          <dl className="mt-4">
            {isLeave ? (
              <>
                <Field label="Periode">
                  {formatDate(data.tanggal_mulai)} — {formatDate(data.tanggal_selesai)}
                </Field>
                <Field label="Jumlah hari">{data.jumlah_hari} hari</Field>
                <Field label="Lokasi tujuan">{data.lokasi_tujuan}</Field>
                <Field label="Keperluan tugas">{data.keterangan_lokasi}</Field>
              </>
            ) : (
              <>
                <Field label="Tanggal">{formatDate(data.tanggal)}</Field>
                <Field label="Waktu">
                  {data.waktu_mulai} – {data.waktu_selesai}
                </Field>
                {/* Durasi berasal dari server; tidak ada kalkulasi kompensasi di client. */}
                <Field label="Total jam">{data.total_jam} jam</Field>
              </>
            )}
            <Field label="Alasan">{data.alasan}</Field>
            <Field label="Dokumen pendukung">
              {data.dokumen_url ? (
                <a
                  href={data.dokumen_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium text-cyan-700 underline hover:text-cyan-900"
                >
                  Buka dokumen
                  <span className="sr-only"> (tab baru)</span>
                </a>
              ) : (
                "Tidak ada dokumen"
              )}
            </Field>
          </dl>
        </section>

        <section aria-labelledby="timeline-heading" className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <h2 id="timeline-heading" className="text-lg font-bold">
            Riwayat approval
          </h2>
          <div className="mt-4">
            <ApprovalTimeline history={data.approval_history} />
          </div>
        </section>
      </div>

      {canDecide ? (
        <div className="mt-7 flex flex-wrap gap-3">
          <Button onClick={() => setDialogMode("setujui")} disabled={busy}>
            Setujui
          </Button>
          <Button variant="secondary" onClick={() => setDialogMode("tolak")} disabled={busy}>
            Tolak
          </Button>
          {canDelegate && (
            <Button variant="secondary" onClick={() => setDialogMode("delegate")} disabled={busy}>
              Delegasikan ke HR
            </Button>
          )}
        </div>
      ) : (
        <p className="mt-7 text-sm text-slate-500">
          {isPendingStatus(data.status)
            ? "Pengajuan sedang berada pada tahap approver lain."
            : "Pengajuan sudah final dan tidak dapat diputus lagi."}
        </p>
      )}

      <DecisionDialog
        open={Boolean(dialogMode)}
        mode={dialogMode}
        busy={busy}
        error={dialogError}
        onSubmit={handleDecision}
        onCancel={() => {
          setDialogMode(null);
          setDialogError("");
        }}
      />
    </section>
  );
};
