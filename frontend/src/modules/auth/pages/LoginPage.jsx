import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { useAuth } from "../hooks/useAuth";
import { loginSchema } from "../schemas/auth-schema";
import { safeReturnPath } from "../utils/safe-return-path";
import { AuthCard } from "../components/AuthCard";

const errorMessage = (error) => {
  switch (error.code) {
    case "INVALID_CREDENTIALS":
      return "Email atau password tidak valid.";
    case "ACCOUNT_LOCKED":
      return "Akun terkunci. Pulihkan password untuk membuka akun.";
    case "TOO_MANY_REQUESTS":
    case "RATE_LIMITED":
      return "Terlalu banyak percobaan. Tunggu sebentar lalu coba kembali.";
    case "NETWORK_ERROR":
    case "REQUEST_TIMEOUT":
      return "Tidak dapat terhubung ke layanan. Periksa koneksi lalu coba kembali.";
    default:
      return "Login belum dapat diproses. Silakan coba kembali.";
  }
};

export const LoginPage = () => {
  document.title = "Login — GSNpeeps";
  const auth = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const [formError, setFormError] = useState("");
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = async (values) => {
    setFormError("");
    try {
      await auth.login(values);
      navigate(safeReturnPath(location.state?.returnTo), { replace: true });
    } catch (error) {
      Object.entries(error.fields ?? {}).forEach(([field, message]) => {
        if (field === "email" || field === "password") {
          setError(field, { type: "server", message });
        }
      });
      setFormError(errorMessage(error));
    }
  };

  return (
    <AuthCard
      title="Masuk ke akun"
      description="Gunakan email kerja dan password GSNpeeps Anda."
      footer={
        <p className="text-sm text-slate-500">
          Akun terkunci?{" "}
          <Link to="/reset-password" className="font-semibold text-cyan-700 hover:text-cyan-900">
            Pulihkan password
          </Link>
        </p>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
        {location.state?.message ? (
          <div role="status" className="rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-3 text-sm text-emerald-700">
            {location.state.message}
          </div>
        ) : null}
        {formError ? (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700">
            {formError}
          </div>
        ) : null}
        <FormInput
          id="email"
          label="Email kerja"
          type="email"
          autoComplete="username"
          registration={register("email")}
          error={errors.email?.message}
          disabled={isSubmitting}
        />
        <FormInput
          id="password"
          label="Password"
          type="password"
          autoComplete="current-password"
          registration={register("password")}
          error={errors.password?.message}
          disabled={isSubmitting}
        />
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "Sedang masuk…" : "Masuk"}
        </Button>
      </form>
    </AuthCard>
  );
};
