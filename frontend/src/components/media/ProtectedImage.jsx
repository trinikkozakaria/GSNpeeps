import { useEffect, useState } from "react";
import { protectedMediaRequest } from "../../lib/api/client";

const imagePlaceholder = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='120' viewBox='0 0 160 120'%3E%3Crect width='160' height='120' fill='%23f1f5f9'/%3E%3Cpath d='M52 78l18-20 14 14 10-10 18 16H52z' fill='%2394a3b8'/%3E%3Ccircle cx='62' cy='45' r='7' fill='%2394a3b8'/%3E%3C/svg%3E";

export const useProtectedMedia = (path) => {
  const [state, setState] = useState({ url: "", status: path ? "loading" : "idle" });

  useEffect(() => {
    if (!path) {
      setState({ url: "", status: "idle" });
      return undefined;
    }
    let active = true;
    let objectURL = "";
    const controller = new AbortController();
    setState({ url: "", status: "loading" });
    protectedMediaRequest(path, controller.signal)
      .then((blob) => {
        if (!active || !(blob instanceof Blob)) return;
        objectURL = URL.createObjectURL(blob);
        setState({ url: objectURL, status: "ready" });
      })
      .catch((error) => {
        if (active && error?.code !== "ERR_CANCELED") setState({ url: "", status: "error" });
      });
    return () => {
      active = false;
      controller.abort();
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [path]);

  return state;
};

export const ProtectedImage = ({ path, alt = "", className = "", ...props }) => {
  const media = useProtectedMedia(path);
  if (media.status === "ready") return <img src={media.url} alt={alt} className={className} {...props} />;
  if (media.status === "loading") {
    return <img src={imagePlaceholder} alt={alt} aria-busy="true" className={className} {...props} />;
  }
  return (
    <span className="inline-flex flex-col items-center gap-1 text-center text-xs text-red-700">
      <img src={imagePlaceholder} alt={alt} className={className} {...props} />
      <span role="alert">Gambar gagal dimuat</span>
    </span>
  );
};

export const ProtectedDownloadLink = ({ path, fileName, children, className = "" }) => {
  const media = useProtectedMedia(path);
  if (media.status === "error") {
    return <span role="alert" className="text-sm font-medium text-red-700">Berkas gagal dimuat</span>;
  }
  return (
    <a
      href={media.status === "ready" ? media.url : "#"}
      download={fileName}
      target="_blank"
      rel="noopener noreferrer"
      aria-disabled={media.status !== "ready"}
      aria-busy={media.status === "loading"}
      onClick={(event) => {
        if (media.status !== "ready") event.preventDefault();
      }}
      className={`${className} ${media.status !== "ready" ? "pointer-events-none opacity-60" : ""}`}
    >
      {children}
      {media.status === "loading" && <span className="sr-only"> (memuat berkas)</span>}
    </a>
  );
};
