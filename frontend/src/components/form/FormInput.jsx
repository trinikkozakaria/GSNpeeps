import { useState } from "react";

import { Button } from "../ui/Button";
import { Input } from "../ui/Input";

export const FormInput = ({
  id,
  label,
  error,
  description,
  registration,
  type = "text",
  ...props
}) => {
  const [isPasswordVisible, setIsPasswordVisible] = useState(false);
  const isPassword = type === "password";
  const descriptionID = description ? `${id}-description` : undefined;
  const errorID = error ? `${id}-error` : undefined;
  const describedBy = [descriptionID, errorID].filter(Boolean).join(" ") || undefined;

  return (
    <div>
      <label htmlFor={id} className="mb-2 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <div className={isPassword ? "flex gap-2" : undefined}>
        <Input
          id={id}
          type={isPassword && isPasswordVisible ? "text" : type}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          {...registration}
          {...props}
        />
        {isPassword ? (
          <Button
            type="button"
            variant="secondary"
            className="shrink-0"
            aria-label={isPasswordVisible ? `Sembunyikan ${label}` : `Tampilkan ${label}`}
            onClick={() => setIsPasswordVisible((visible) => !visible)}
          >
            {isPasswordVisible ? "Sembunyikan" : "Tampilkan"}
          </Button>
        ) : null}
      </div>
      {description ? (
        <p id={descriptionID} className="mt-2 text-xs text-slate-500">
          {description}
        </p>
      ) : null}
      {error ? (
        <p id={errorID} role="alert" className="mt-2 text-sm text-rose-700">
          {error}
        </p>
      ) : null}
    </div>
  );
};

