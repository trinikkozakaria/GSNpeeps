import { useCallback, useEffect, useRef, useState } from "react";

/**
 * State machine kamera absensi.
 *
 * idle -> requesting_camera -> camera_ready -> preview -> idle
 * requesting_camera -> unsupported | denied | error  (seluruhnya recoverable via fallback)
 *
 * Kamera hanya diminta setelah aksi pengguna, tidak pernah saat render. Seluruh track
 * MediaStream dihentikan ketika capture selesai, dibatalkan, atau komponen unmount agar
 * indikator kamera perangkat tidak menyala setelah alur berakhir.
 */
export const cameraStates = {
  idle: "idle",
  requestingCamera: "requesting_camera",
  cameraReady: "camera_ready",
  preview: "preview",
  unsupported: "unsupported",
  denied: "denied",
  error: "error",
};

const cameraErrorMessage = (error) => {
  if (error?.name === "NotAllowedError" || error?.name === "SecurityError") {
    return "Akses kamera ditolak. Gunakan unggah manual atau izinkan kamera pada browser.";
  }
  if (error?.name === "NotFoundError" || error?.name === "OverconstrainedError") {
    return "Kamera tidak ditemukan pada perangkat ini. Gunakan unggah manual.";
  }
  if (error?.name === "NotReadableError") {
    return "Kamera sedang dipakai aplikasi lain. Tutup aplikasi tersebut atau unggah manual.";
  }
  return "Kamera tidak dapat dibuka. Gunakan unggah manual.";
};

export const useCameraCapture = () => {
  const [state, setState] = useState(cameraStates.idle);
  const [message, setMessage] = useState("");
  const streamRef = useRef(null);
  const videoRef = useRef(null);

  const stopStream = useCallback(() => {
    const stream = streamRef.current;
    if (!stream) return;
    stream.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
  }, []);

  // Menjamin kamera dilepas ketika pengguna meninggalkan halaman.
  useEffect(() => stopStream, [stopStream]);

  const start = useCallback(async () => {
    setMessage("");
    if (!navigator.mediaDevices?.getUserMedia) {
      setState(cameraStates.unsupported);
      setMessage("Browser ini tidak mendukung kamera langsung. Gunakan unggah manual.");
      return;
    }
    setState(cameraStates.requestingCamera);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "user" },
        audio: false,
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        // Autoplay dapat ditolak browser dan tidak diimplementasikan sebagian environment.
        // Kegagalan play tidak boleh menggagalkan alur karena stream sudah aktif.
        try {
          await videoRef.current.play?.();
        } catch {
          // Pratinjau tetap tersedia meskipun autoplay ditolak.
        }
      }
      setState(cameraStates.cameraReady);
    } catch (error) {
      stopStream();
      setState(
        error?.name === "NotAllowedError" || error?.name === "SecurityError"
          ? cameraStates.denied
          : cameraStates.error,
      );
      setMessage(cameraErrorMessage(error));
    }
  }, [stopStream]);

  /**
   * Mengambil satu frame menjadi File JPEG. Kamera langsung dilepas setelah frame diambil
   * sehingga stream tidak hidup selama pengguna meninjau hasil.
   */
  const capture = useCallback(async () => {
    const video = videoRef.current;
    if (!video) return null;

    const canvas = document.createElement("canvas");
    canvas.width = video.videoWidth || 640;
    canvas.height = video.videoHeight || 480;
    const context = canvas.getContext("2d");
    if (!context) return null;
    context.drawImage(video, 0, 0, canvas.width, canvas.height);

    const blob = await new Promise((resolve) => {
      if (!canvas.toBlob) {
        resolve(null);
        return;
      }
      canvas.toBlob(resolve, "image/jpeg", 0.9);
    });
    stopStream();
    if (!blob) {
      setState(cameraStates.error);
      setMessage("Foto tidak dapat diambil. Gunakan unggah manual.");
      return null;
    }
    setState(cameraStates.preview);
    return new File([blob], "absensi.jpg", { type: "image/jpeg" });
  }, [stopStream]);

  const reset = useCallback(() => {
    stopStream();
    setState(cameraStates.idle);
    setMessage("");
  }, [stopStream]);

  return { state, message, videoRef, start, capture, reset, stopStream };
};
