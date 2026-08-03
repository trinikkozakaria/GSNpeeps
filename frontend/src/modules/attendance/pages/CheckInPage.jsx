import { useEffect, useState } from "react";

import { FileField } from "../../../components/form/FileField";
import { Button } from "../../../components/ui/Button";
import { attendanceStatusLabel, workModeLabel } from "../../../lib/request-status";
import { useAuth } from "../../auth/hooks/useAuth";
import { cameraStates, useCameraCapture } from "../hooks/useCameraCapture";
import { geolocationStates, useGeolocation } from "../hooks/useGeolocation";
import { useOfficeLocations, useRecordAttendance } from "../hooks/useAttendance";
import { validatePhotoFile, workModes } from "../schemas/attendance-schema";

// Pesan untuk error domain absensi. Radius dan waktu tetap ditentukan server; pesan di sini
// hanya menjelaskan langkah berikutnya tanpa mengungkap konfigurasi kantor.
const submitErrorMessage = (error) => {
  switch (error?.code) {
    case "OUT_OF_RADIUS":
      return "Lokasi Anda berada di luar radius kantor yang dipilih. Dekati kantor atau pilih mode kerja WFH/WFA bila memang bekerja dari luar kantor.";
    case "DUPLICATE_CHECKIN":
      return "Check-in hari ini sudah tercatat. Gunakan check-out bila ingin mengakhiri hari kerja.";
    case "CHECKOUT_WITHOUT_CHECKIN":
      return "Check-out memerlukan check-in aktif pada hari yang sama.";
    case "NON_WORKING_DAY":
      return "Absensi reguler hanya tersedia Senin sampai Jumat.";
    case "INVALID_OFFICE_LOCATION":
      return "Lokasi kantor yang dipilih tidak aktif. Pilih kantor lain.";
    case "FILE_TOO_LARGE":
      return "Ukuran foto melebihi batas 5 MB.";
    case "UNSUPPORTED_FILE_TYPE":
      return "Format foto ditolak server. Gunakan JPG atau PNG.";
    default:
      return error?.message ?? "Absensi belum dapat dicatat. Coba lagi.";
  }
};

const StatusLine = ({ label, children }) => (
  <div className="flex flex-wrap items-baseline justify-between gap-2 border-b border-white/10 py-2">
    <dt className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</dt>
    <dd className="text-sm text-slate-100">{children}</dd>
  </div>
);

export const CheckInPage = () => {
  document.title = "Kehadiran — GSNpeeps";
  const auth = useAuth();
  const camera = useCameraCapture();
  const geolocation = useGeolocation();
  const record = useRecordAttendance();

  const [attendanceType, setAttendanceType] = useState("check_in");
  const [workMode, setWorkMode] = useState("WFO");
  const [officeLocationID, setOfficeLocationID] = useState("");
  const [photo, setPhoto] = useState(null);
  // Sumber foto menentukan kontrol mana yang ditampilkan; berkas unggahan tetap dikelola
  // FileField agar pesan validasinya tidak hilang saat berkas tidak valid dipilih.
  const [photoSource, setPhotoSource] = useState("");
  const [photoError, setPhotoError] = useState("");
  const [formError, setFormError] = useState("");
  const [result, setResult] = useState(null);

  // Master lokasi hanya diambil ketika WFO dipilih.
  const offices = useOfficeLocations(workMode === "WFO");
  const isFallbackVisible =
    camera.state === cameraStates.unsupported ||
    camera.state === cameraStates.denied ||
    camera.state === cameraStates.error;

  // Foto hasil kamera dilepas ketika komponen dilepas agar tidak tertahan di memory.
  useEffect(() => () => setPhoto(null), []);

  const handleCapture = async () => {
    const captured = await camera.capture();
    if (captured) {
      setPhoto(captured);
      setPhotoSource("camera");
      setPhotoError("");
    }
  };

  const handleRetake = () => {
    setPhoto(null);
    setPhotoSource("");
    setPhotoError("");
    camera.reset();
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    setFormError("");
    setResult(null);

    const photoProblem = validatePhotoFile(photo);
    setPhotoError(photoProblem);
    if (photoProblem) return;

    if (workMode === "WFO" && !officeLocationID) {
      setFormError("Pilih lokasi kantor untuk mode kerja WFO.");
      return;
    }

    // Lokasi diminta tepat sebelum kirim agar koordinat sedekat mungkin dengan waktu absensi.
    const coordinates = await geolocation.request();
    if (!coordinates) return;

    try {
      const created = await record.mutateAsync({
        tipe: attendanceType,
        mode_kerja: workMode,
        gps_lat: coordinates.latitude,
        gps_long: coordinates.longitude,
        office_location_id: workMode === "WFO" ? officeLocationID : undefined,
        foto: photo,
      });
      setResult(created);
      setPhoto(null);
      setPhotoSource("");
      camera.reset();
      geolocation.reset();
    } catch (error) {
      setFormError(submitErrorMessage(error));
    }
  };

  const isSubmitting = record.isPending || geolocation.state === geolocationStates.requesting;

  return (
    <section aria-labelledby="checkin-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Absensi</p>
      <h1 id="checkin-title" className="mt-2 text-3xl font-bold">
        Kehadiran
      </h1>
      <p className="mt-2 max-w-2xl text-slate-300">
        Waktu absensi diambil dari jam server, bukan dari jam perangkat Anda. Koordinat wajib
        untuk semua mode kerja.
      </p>

      <form onSubmit={handleSubmit} noValidate className="mt-7 grid gap-6 lg:grid-cols-2">
        <div className="grid gap-5 rounded-xl border border-white/10 bg-white/[0.03] p-5">
          <fieldset>
            <legend className="text-sm font-medium text-slate-200">Jenis absensi</legend>
            <div className="mt-3 flex flex-wrap gap-3">
              {[
                { value: "check_in", label: "Check-in" },
                { value: "check_out", label: "Check-out" },
              ].map((option) => (
                <label key={option.value} className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    name="tipe"
                    value={option.value}
                    checked={attendanceType === option.value}
                    onChange={() => setAttendanceType(option.value)}
                    disabled={isSubmitting}
                    className="h-4 w-4"
                  />
                  {option.label}
                </label>
              ))}
            </div>
          </fieldset>

          <fieldset>
            <legend className="text-sm font-medium text-slate-200">Mode kerja</legend>
            <div className="mt-3 flex flex-wrap gap-3">
              {workModes.map((mode) => (
                <label key={mode} className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    name="mode_kerja"
                    value={mode}
                    checked={workMode === mode}
                    onChange={() => setWorkMode(mode)}
                    disabled={isSubmitting}
                    className="h-4 w-4"
                  />
                  {workModeLabel[mode]}
                </label>
              ))}
            </div>
            <p className="mt-2 text-xs text-slate-400">
              {workMode === "WFO"
                ? "WFO diverifikasi server terhadap lokasi kantor yang Anda pilih."
                : "WFH dan WFA tetap merekam koordinat tanpa verifikasi jarak kantor."}
            </p>
          </fieldset>

          {workMode === "WFO" && (
            <div>
              <label htmlFor="office-location" className="mb-2 block text-sm font-medium text-slate-200">
                Lokasi kantor
              </label>
              <select
                id="office-location"
                value={officeLocationID}
                onChange={(event) => setOfficeLocationID(event.target.value)}
                disabled={isSubmitting}
                className="min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
              >
                <option value="">Pilih kantor</option>
                {(offices.data ?? []).map((office) => (
                  <option key={office.id} value={office.id}>
                    {office.nama}
                  </option>
                ))}
              </select>
              {offices.data && offices.data.length === 0 && (
                <p className="mt-2 text-xs text-amber-200">
                  Belum ada lokasi kantor aktif. Hubungi HR atau gunakan mode WFH/WFA.
                </p>
              )}
            </div>
          )}
        </div>

        <div className="grid gap-4 rounded-xl border border-white/10 bg-white/[0.03] p-5">
          <h2 className="text-base font-semibold text-white">Foto absensi</h2>

          {camera.state === cameraStates.idle && !photo && (
            <Button type="button" variant="secondary" onClick={camera.start}>
              Nyalakan kamera
            </Button>
          )}
          {camera.state === cameraStates.requestingCamera && (
            <p role="status" className="text-sm text-slate-300">
              Meminta izin kamera…
            </p>
          )}

          {/* Elemen video tetap ada di DOM agar ref tersedia saat stream dipasang. */}
          <video
            ref={camera.videoRef}
            playsInline
            muted
            aria-label="Pratinjau kamera"
            className={
              camera.state === cameraStates.cameraReady
                ? "w-full rounded-lg border border-white/15"
                : "hidden"
            }
          />
          {camera.state === cameraStates.cameraReady && (
            <div className="flex flex-wrap gap-3">
              <Button type="button" onClick={handleCapture}>
                Ambil foto
              </Button>
              <Button type="button" variant="secondary" onClick={camera.reset}>
                Batalkan kamera
              </Button>
            </div>
          )}

          {camera.message && (
            <p role="alert" className="rounded-lg border border-amber-300/30 bg-amber-300/10 p-3 text-sm text-amber-100">
              {camera.message}
            </p>
          )}

          {isFallbackVisible && (
            <FileField
              id="attendance-photo"
              label="Unggah foto absensi"
              accept=".jpg,.jpeg,.png"
              description="JPG atau PNG maksimal 5 MB. Sertakan watermark dari aplikasi kamera Anda bila tersedia; waktu resmi tetap diambil dari server."
              file={photo}
              error={photoError}
              disabled={isSubmitting}
              onFileChange={(selected) => {
                setPhoto(selected);
                setPhotoSource(selected ? "upload" : "");
                setPhotoError(selected ? validatePhotoFile(selected) : "");
              }}
            />
          )}

          {photo && photoSource === "camera" && (
            <div>
              <p className="text-sm text-slate-200">
                Foto siap dikirim: <span className="font-medium">{photo.name}</span>
              </p>
              <Button type="button" variant="secondary" className="mt-3" onClick={handleRetake} disabled={isSubmitting}>
                Ambil ulang
              </Button>
            </div>
          )}
          {photoError && !isFallbackVisible && (
            <p role="alert" className="text-sm text-rose-300">
              {photoError}
            </p>
          )}

          {geolocation.message && (
            <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-200">
              {geolocation.message}
            </p>
          )}
          {geolocation.state === geolocationStates.requesting && (
            <p role="status" className="text-sm text-slate-300">
              Membaca lokasi Anda…
            </p>
          )}

          {formError && (
            <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-200">
              {formError}
            </p>
          )}

          <div>
            <Button type="submit" disabled={isSubmitting || !photo}>
              {isSubmitting ? "Mengirim…" : "Kirim absensi"}
            </Button>
          </div>
        </div>
      </form>

      {result && (
        <section
          aria-labelledby="checkin-result-title"
          className="mt-7 rounded-xl border border-emerald-300/30 bg-emerald-300/10 p-5"
        >
          <h2 id="checkin-result-title" className="text-lg font-bold text-emerald-100">
            Absensi tercatat
          </h2>
          <dl className="mt-4">
            <StatusLine label="Waktu server">{new Date(result.waktu).toLocaleString("id-ID")}</StatusLine>
            <StatusLine label="Jenis">
              {result.tipe === "check_in" ? "Check-in" : "Check-out"}
            </StatusLine>
            <StatusLine label="Mode kerja">{workModeLabel[result.mode_kerja]}</StatusLine>
            <StatusLine label="Status">
              {attendanceStatusLabel[result.status] ?? result.status}
            </StatusLine>
            {typeof result.distance_meters === "number" && (
              <StatusLine label="Jarak dari kantor">
                {Math.round(result.distance_meters)} meter
              </StatusLine>
            )}
          </dl>
          <p className="mt-3 text-xs text-emerald-100/80">
            Dicatat untuk {auth.user?.nama ?? "akun Anda"} menggunakan waktu server.
          </p>
        </section>
      )}
    </section>
  );
};
