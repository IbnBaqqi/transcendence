import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { contactDetailsSchema, type ContactDetailsFormValues } from "../../schemas/contactDetails";
import Button from "../objects/Button.tsx";
import { useEffect, useState } from "react";
import { useOwnProfile, useUpdateOwnProfile } from "../../api/profile";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";

export function ContactDetailsSection() {
  const { t } = useTranslation();
  const [isEditing, setEditing] = useState(false);
  const { user } = useAuth();
  const { data: profile } = useOwnProfile({ enabled: Boolean(user) });
  const update = useUpdateOwnProfile();

  const form = useForm<ContactDetailsFormValues>({
    resolver: zodResolver(contactDetailsSchema),
    mode: "onBlur",
  });
  const {
    formState: { errors, isValid, isSubmitting },
  } = form;

  // Seed the fields once the profile lands (and after any refetch), but never
  // clobber what the user is typing mid-edit.
  useEffect(() => {
    if (profile && !isEditing) {
      form.reset({
        firstname: profile.firstname ?? "",
        lastname: profile.lastname ?? "",
        phone_number: profile.phone_number ?? "",
        location: profile.location ?? "",
      });
    }
  }, [profile, isEditing, form]);

  const handleSubmit = async (data: ContactDetailsFormValues) => {
    form.clearErrors("root");
    try {
      await update.mutateAsync({
        firstname: data.firstname,
        lastname: data.lastname,
        phone_number: data.phone_number,
        location: data.location,
      });
      setEditing(false);
    } catch (err) {
      form.setError("root", {
        message: isApiError(err) ? err.message : t("common.somethingWentWrong"),
      });
    }
  };

  // max-w-md rather than max-w-fit: once the grid stacks, fit-content shrinks
  // the whole form to one input's width and leaves the fields floating in the
  // viewport. 28rem is within a few pixels of today's two-column width.
  return (
    <Form form={form} onSubmit={handleSubmit} className="max-w-md" isEditing={isEditing}>
      <div className="space-y-2">
        {/* One column below sm: at 320px a fixed two-up grid gives each input
            ~136px against a content width of ~206px. */}
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField label={t("forms.firstName")} name="firstname" validateOnChange />
          <FormField label={t("forms.lastName")} name="lastname" />
          <FormField label={t("forms.phone")} name="phone_number" type="tel" />
          <FormField label={t("forms.city")} name="location" validateOnChange />
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
              {t("forms.editDetails")}
            </Button>
          )}
        </div>
      </div>
    </Form>
  );
}
