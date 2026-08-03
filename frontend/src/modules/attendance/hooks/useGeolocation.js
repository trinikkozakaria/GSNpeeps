import { useCallback, useRef, useState } from "react";

/**
 * Membaca posisi sekali per permintaan. Tidak ada watchPosition maupun polling berkelanjutan
 * agar perangkat tidak terus mengaktifkan GPS selama halaman terbuka.
 */
export const geolocationStates = {
  idle: "idle",
  requesting: "requesting_location",
  ready: "ready",
  error: "error",
};

const geolocationMessage = (error) => {
  switch (error?.code) {
    case 1:
      return "Izin lokasi ditolak. Aktifkan izin lokasi untuk mencatat absensi.";
    case 2:
      return "Lokasi tidak tersedia saat ini. Periksa sinyal GPS lalu coba lagi.";
    case 3:
      return "Permintaan lokasi melebihi batas waktu. Coba lagi.";
    default:
      return "Lokasi tidak dapat dibaca. Coba lagi.";
  }
};

export const useGeolocation = () => {
  const [state, setState] = useState(geolocationStates.idle);
  const [position, setPosition] = useState(null);
  const [message, setMessage] = useState("");
  const pendingRef = useRef(false);

  const request = useCallback(async () => {
    if (pendingRef.current) return null;
    setMessage("");

    if (!navigator.geolocation) {
      setState(geolocationStates.error);
      setMessage("Browser ini tidak mendukung lokasi. Absensi memerlukan koordinat.");
      return null;
    }

    pendingRef.current = true;
    setState(geolocationStates.requesting);
    try {
      const result = await new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, {
          enableHighAccuracy: true,
          timeout: 15000,
          maximumAge: 0,
        });
      });
      const coordinates = {
        latitude: result.coords.latitude,
        longitude: result.coords.longitude,
        accuracy: result.coords.accuracy,
      };
      setPosition(coordinates);
      setState(geolocationStates.ready);
      return coordinates;
    } catch (error) {
      setState(geolocationStates.error);
      setMessage(geolocationMessage(error));
      return null;
    } finally {
      pendingRef.current = false;
    }
  }, []);

  const reset = useCallback(() => {
    setState(geolocationStates.idle);
    setPosition(null);
    setMessage("");
  }, []);

  return { state, position, message, request, reset };
};
