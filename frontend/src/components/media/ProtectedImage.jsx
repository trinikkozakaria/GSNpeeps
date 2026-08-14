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
        setState({ url: objectURL, status: "ready", contentType: blob.type });
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

export const ProtectedImage = ({ path, alt = "", className = "", showErrorMessage = true, ...props }) => {
  const media = useProtectedMedia(path);
  const isImage = media.status !== "ready" || media.contentType?.startsWith("image/");
  if (media.status === "ready" && isImage) return <img src={media.url} alt={alt} className={className} {...props} />;
  if (media.status === "loading") {
    return <img src={imagePlaceholder} alt={alt} aria-busy="true" className={className} {...props} />;
  }
  return (
    <span className="inline-flex flex-col items-center gap-1 text-center text-xs text-red-700">
      <img src={imagePlaceholder} alt={alt} className={className} {...props} />
      {showErrorMessage && <span role="alert">Gambar gagal dimuat</span>}
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

const extensionOf = (fileName = "") => fileName.slice(fileName.lastIndexOf(".")).toLowerCase();

export const ProtectedDocumentPreview = ({ path, fileName, className = "" }) => {
  const media = useProtectedMedia(path);
  const extension = extensionOf(fileName);
  const isImage = [".jpg", ".jpeg", ".png"].includes(extension);
  const isPDF = extension === ".pdf";

  if (media.status === "loading") {
    return <p role="status" className="text-sm text-slate-500">Memuat pratinjau {fileName}â€¦</p>;
  }
  if (media.status === "error") {
    return <p role="alert" className="text-sm font-medium text-red-700">Berkas gagal dimuat</p>;
  }
  if (media.status !== "ready") return null;

  return (
    <div className={`space-y-2 ${className}`}>
      {isImage && (
        <a href={media.url} target="_blank" rel="noopener noreferrer" aria-label={`Buka pratinjau ${fileName}`}>
          <img
            src={media.url}
            alt={`Pratinjau ${fileName}`}
            className="h-28 w-full max-w-xs rounded-lg border border-slate-200 object-contain bg-slate-50"
          />
        </a>
      )}
      {isPDF && (
        <iframe
          src={media.url}
          title={`Pratinjau ${fileName}`}
          className="h-40 w-full max-w-md rounded-lg border border-slate-200 bg-white"
        />
      )}
      {!isImage && !isPDF && (
        <p className="text-xs text-slate-500">Pratinjau tidak tersedia untuk format ini.</p>
      )}
      <a
        href={media.url}
        download={fileName}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-block text-sm font-medium text-cyan-700 underline hover:text-cyan-900"
      >
        Unduh {fileName}
      </a>
    </div>
  );
};
