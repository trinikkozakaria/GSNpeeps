import { z } from "zod";

export const roleSchema = z.enum(["karyawan", "atasan", "hr", "top_management"]);

const emailSchema = z
  .string()
  .trim()
  .min(1, "Email wajib diisi")
  .email("Format email tidak valid")
  .max(255, "Email terlalu panjang");

const currentPasswordSchema = z
  .string()
  .min(8, "Password minimal 8 karakter")
  .max(128, "Password maksimal 128 karakter");

const newPasswordSchema = z
  .string()
  .min(12, "Password baru minimal 12 karakter")
  .max(128, "Password baru maksimal 128 karakter");

export const loginSchema = z
  .object({
    email: emailSchema,
    password: currentPasswordSchema,
  })
  .strict();

export const changePasswordSchema = z
  .object({
    current_password: currentPasswordSchema,
    new_password: newPasswordSchema,
    new_password_confirmation: newPasswordSchema,
  })
  .strict()
  .superRefine((value, context) => {
    if (value.new_password !== value.new_password_confirmation) {
      context.addIssue({
        code: "custom",
        path: ["new_password_confirmation"],
        message: "Konfirmasi password harus sama",
      });
    }
    if (value.current_password === value.new_password) {
      context.addIssue({
        code: "custom",
        path: ["new_password"],
        message: "Password baru harus berbeda dari password saat ini",
      });
    }
  });

export const selfResetPasswordSchema = z
  .object({
    email: emailSchema,
    current_password: currentPasswordSchema,
    new_password: newPasswordSchema,
    new_password_confirmation: newPasswordSchema,
  })
  .strict()
  .superRefine((value, context) => {
    if (value.new_password !== value.new_password_confirmation) {
      context.addIssue({
        code: "custom",
        path: ["new_password_confirmation"],
        message: "Konfirmasi password harus sama",
      });
    }
    if (value.current_password === value.new_password) {
      context.addIssue({
        code: "custom",
        path: ["new_password"],
        message: "Password baru harus berbeda dari password saat ini",
      });
    }
  });

export const authUserSchema = z
  .object({
    id: z.string().uuid(),
    employee_id: z.string().uuid(),
    nama: z.string().min(1),
    email: z.string().email(),
    role: roleSchema,
    foto_profil_url: z.string().nullable().optional(),
  })
  .strict();

export const loginDataSchema = z
  .object({
    token: z.string().min(1),
    token_type: z.literal("Bearer"),
    expires_in: z.literal(28800),
    user: authUserSchema,
  })
  .strict();

export const photoUpdateDataSchema = z
  .object({
    foto_profil_url: z.string(),
  })
  .strict();

export const passwordChangedDataSchema = z
  .object({
    password_changed: z.literal(true),
    account_locked: z.literal(false),
    sessions_revoked: z.literal(true),
  })
  .strict();

