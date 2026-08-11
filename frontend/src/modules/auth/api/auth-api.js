import { apiClient } from "../../../lib/api/client";
import {
  authUserSchema,
  loginDataSchema,
  passwordChangedDataSchema,
  photoUpdateDataSchema,
} from "../schemas/auth-schema";

export const loginRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/auth/login", payload, { signal });
  return loginDataSchema.parse(envelope.data);
};

export const logoutRequest = async (signal) => {
  await apiClient.post("/auth/logout", undefined, { signal });
};

export const currentUserRequest = async (signal) => {
  const envelope = await apiClient.get("/auth/me", { signal });
  return authUserSchema.parse(envelope.data);
};

export const updateMyPhotoRequest = async (file, signal) => {
  const form = new FormData();
  form.append("foto", file);
  const envelope = await apiClient.put("/auth/me/foto", form, { signal });
  return photoUpdateDataSchema.parse(envelope.data);
};

export const changePasswordRequest = async (payload, signal) => {
  const envelope = await apiClient.patch("/auth/me/password", payload, { signal });
  return passwordChangedDataSchema.parse(envelope.data);
};

export const selfResetPasswordRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/auth/reset-password", payload, { signal });
  return passwordChangedDataSchema.parse(envelope.data);
};

