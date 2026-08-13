import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { selfResetPasswordRequest } from "../api/auth-api";
import { AuthCard } from "../components/AuthCard";
import { selfResetPasswordSchema } from "../schemas/auth-schema";

export const ResetPasswordPage = () => {
  document.title = "Pulihkan password — GSNpeeps";
  const navigate = useNavigate();
  const [formError, setFormError] = useState("");
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(selfResetPasswordSchema),
    defaultValues: {
      email: "",
      current_password: "",
      new_password: "",
      new_password_confirmation: "",
    },
  });

  const onSubmit = async (values) => {
    setFormError("");
    try {
      await selfResetPasswordRequest(values);
      navigate("/login", {
        replace: true,
        state: { message: "Password berhasil dipulihkan. Silakan login kembali." },
      });
    } catch (error) {
      Object.entries(error.fields ?? {}).forEach(([field, message]) => {
        if (field in values) {
          setError(field, { type: "server", message });
        }
      });
      setFormError(
        error.code === "TOO_MANY_REQUESTS" ||
          error.code === "RATE_LIMITED" ||
          error.code === "ACCOUNT_LOCKED"
          ? "Terlalu banyak percobaan. Tunggu sebentar sebelum mencoba kembali."
          : "Email atau password saat ini tidak valid.",
      );
    }
  };

  return (
    <AuthCard
      title="Pulihkan password"
      description="Verifikasi email dan password saat ini. Fitur ini tidak dapat digunakan jika password benar-benar terlupa."
      footer={
        <Link to="/login" className="text-sm font-semibold text-cyan-700 hover:text-cyan-900">
          Kembali ke login
        </Link>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
        {formError ? (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700">
            {formError}
          </div>
        ) : null}
        <FormInput id="reset-email" label="Email kerja" type="email" autoComplete="username" registration={register("email")} error={errors.email?.message} disabled={isSubmitting} />
        <FormInput id="current-password" label="Password saat ini" type="password" autoComplete="current-password" registration={register("current_password")} error={errors.current_password?.message} disabled={isSubmitting} />
        <FormInput id="new-password" label="Password baru" type="password" autoComplete="new-password" description="Minimal 12 karakter." registration={register("new_password")} error={errors.new_password?.message} disabled={isSubmitting} />
        <FormInput id="confirm-password" label="Konfirmasi password baru" type="password" autoComplete="new-password" registration={register("new_password_confirmation")} error={errors.new_password_confirmation?.message} disabled={isSubmitting} />
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "Memproses…" : "Pulihkan password"}
        </Button>
      </form>
    </AuthCard>
  );
};
