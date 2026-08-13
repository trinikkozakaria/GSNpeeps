export class AppError extends Error {
  constructor({ status = 0, code, message, fields = {}, cause }) {
    super(message);
    this.name = "AppError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.cause = cause;
  }
}

export const normalizeApiError = (error) => {
  if (error instanceof AppError) {
    return error;
  }

  if (error.code === "ERR_CANCELED") {
    return new AppError({ code: "REQUEST_ABORTED", message: "Permintaan dibatalkan.", cause: error });
  }

  if (!error.response) {
    const isTimeout = error.code === "ECONNABORTED";
    return new AppError({
      code: isTimeout ? "REQUEST_TIMEOUT" : "NETWORK_ERROR",
      message: isTimeout
        ? "Waktu permintaan habis. Silakan coba lagi."
        : "Tidak dapat terhubung ke layanan.",
      cause: error,
    });
  }

  const payload = error.response.data?.error;
  return new AppError({
    status: error.response.status,
    code: payload?.code ?? "UNEXPECTED_RESPONSE",
    message: payload?.message ?? "Layanan tidak dapat memproses permintaan.",
    fields: payload?.fields ?? {},
    cause: error,
  });
};

