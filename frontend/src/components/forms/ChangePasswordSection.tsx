import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { changePasswordSchema, type ChangePasswordFormValues } from "../../schemas/changePassword";
import Button from "../objects/Button.tsx";
import { useState } from "react";
import { useChangePassword } from "../../api/profile";
import { isApiError } from "../../api/client";

export function ChangePasswordSection() {
  const { t } = useTranslation();
  const [isEditing, setEditing] = useState(false);
  const changePassword = useChangePassword();

  const form = useForm<ChangePasswordFormValues>({
    resolver: zodResolver(changePasswordSchema),
    mode: "onBlur",
  });
  const {
    formState: { errors, isValid, isSubmitting },
  } = form;

  const handleSubmit = async (data: ChangePasswordFormValues) => {
    form.clearErrors("root");
    try {
      await changePassword.mutateAsync({
        current_password: data.currentPassword,
        new_password: data.newPassword,
      });
      form.reset();
      setEditing(false);
    } catch (err) {
      form.setError("root", {
        message: isApiError(err) ? err.message : t("common.somethingWentWrong"),
      });
    }
  };

  return (
    <Form form={form} onSubmit={handleSubmit} className="max-w-64" isEditing={isEditing}>
      <div className="space-y-2">
        {isEditing ? (
          <>
            <div className="flex flex-col gap-4">
              <FormField
                label={t("forms.currentPassword")}
                name="currentPassword"
                type="password"
                validateOnChange
              />
              <FormField
                label={t("forms.newPassword")}
                name="newPassword"
                type="password"
                validateOnChange
              />
              <FormField
                label={t("forms.confirmPassword")}
                name="confirmPassword"
                type="password"
                validateOnChange
              />
            </div>
            {errors.root?.message && (
              <p role="alert" className="text-danger text-sm">
                {errors.root.message}
              </p>
            )}
            <div className="flex flex-row gap-2">
              <Button variant="primary" type="submit" disabled={!isValid || isSubmitting}>
                {isSubmitting ? t("common.saving") : t("common.save")}
              </Button>
              <Button
                variant="secondary"
                onClick={() => {
                  form.reset();
                  setEditing(false);
                }}
              >
                {t("common.cancel")}
              </Button>
            </div>
          </>
        ) : (
          <>
            <div>********</div>
            <Button variant="primary" onClick={() => setEditing(true)}>
              {t("forms.editPassword")}
            </Button>
          </>
        )}
      </div>
    </Form>
  );
}
