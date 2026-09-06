import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import { Form } from "./Form";
import { FormTextArea } from "./FormTextArea";
import { bioSchema, type BioFormValues } from "../../schemas/common";
import Button from "../objects/Button.tsx";
import { useEffect, useState } from "react";
import { useOwnProfile, useUpdateOwnProfile } from "../../api/profile";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";

export function BioSection() {
  const { t } = useTranslation();
  const [isEditing, setEditing] = useState(false);
  const { user } = useAuth();
  const { data: profile } = useOwnProfile({ enabled: Boolean(user) });
  const update = useUpdateOwnProfile();

  const form = useForm<BioFormValues>({
    resolver: zodResolver(bioSchema),
    mode: "onBlur",
  });
  const {
    formState: { errors, isValid, isSubmitting },
  } = form;

  // Same seeding rule as ContactDetailsSection: fill when the profile lands,
  // never while the user is editing.
  useEffect(() => {
    if (profile && !isEditing) {
      form.reset({ bio: profile.bio ?? "" });
    }
  }, [profile, isEditing, form]);

  const handleSubmit = async (data: BioFormValues) => {
    form.clearErrors("root");
    try {
      // An emptied textarea sends "", which the backend treats as "clear".
      await update.mutateAsync({ bio: data.bio });
      setEditing(false);
    } catch (err) {
      form.setError("root", {
        message: isApiError(err) ? err.message : t("common.somethingWentWrong"),
      });
    }
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={isEditing}>
      <div className="space-y-2">
        <div className="flex flex-row gap-4">
          <FormTextArea name="bio" maxLength={1000} validateOnChange />
        </div>
        {errors.root?.message && (
          <p role="alert" className="text-danger text-sm">
            {errors.root.message}
          </p>
        )}
        <div className="flex flex-row gap-2">
          {isEditing ? (
            <>
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
            </>
          ) : (
            <Button variant="primary" type="button" onClick={() => setEditing(true)}>
              {t("forms.editText")}
            </Button>
          )}
        </div>
      </div>
    </Form>
  );
}
