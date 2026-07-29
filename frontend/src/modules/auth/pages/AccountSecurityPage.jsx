import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { changePasswordRequest } from "../api/auth-api";
import { useAuth } from "../hooks/useAuth";
import { changePasswordSchema } from "../schemas/auth-schema";

export const AccountSecurityPage = () => {
  document.title = "Keamanan akun — GSNpeeps";
  const auth = useAuth();
  const navigate = useNavigate();
  const [formError, setFormError] = useState("");
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      current_password: "",
      new_password: "",
      new_password_confirmation: "",
    },
  });

  const onSubmit = async (values) => {
    setFormError("");
    try {
      await changePasswordRequest(values);
      await auth.clearSession();
      navigate("/login", {
        replace: true,
        state: { message: "Password berhasil diganti. Silakan login kembali." },
      });
    } catch (error) {
      Object.entries(error.fields ?? {}).forEach(([field, message]) => {
        if (field in values) {
          setError(field, { type: "server", message });
        }
      });
      setFormError(
        error.code === "INVALID_CREDENTIALS"
          ? "Password saat ini tidak valid."
          : "Password belum dapat diganti. Periksa kembali data Anda.",
      );
    }
  };

  return (
    <section aria-labelledby="security-title" className="max-w-xl">
      <h1 id="security-title" className="text-3xl font-bold">
        Keamanan akun
      </h1>
      <p className="mt-3 text-slate-400">
        Mengganti password akan mencabut seluruh sesi dan mewajibkan login ulang.
      </p>
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-8 space-y-5 rounded-2xl border border-white/10 bg-white/[0.04] p-6">
        {formError ? (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-200">
            {formError}
          </div>
        ) : null}
        <FormInput id="security-current-password" label="Password saat ini" type="password" autoComplete="current-password" registration={register("current_password")} error={errors.current_password?.message} disabled={isSubmitting} />
        <FormInput id="security-new-password" label="Password baru" type="password" autoComplete="new-password" description="Minimal 12 karakter dan berbeda dari password saat ini." registration={register("new_password")} error={errors.new_password?.message} disabled={isSubmitting} />
        <FormInput id="security-confirm-password" label="Konfirmasi password baru" type="password" autoComplete="new-password" registration={register("new_password_confirmation")} error={errors.new_password_confirmation?.message} disabled={isSubmitting} />
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Menyimpan…" : "Ganti password"}
        </Button>
      </form>
    </section>
  );
};

