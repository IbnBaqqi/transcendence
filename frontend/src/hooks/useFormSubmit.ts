import { useState } from "react";
import { isApiError } from "../api/client";
import i18next from "../i18n";

export function useFormSubmit<T>(submitFn: (data: T) => Promise<void>) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const handleSubmit = async (data: T) => {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      await submitFn(data);
    } catch (err) {
      setSubmitError(
        isApiError(err) || err instanceof Error
          ? err.message
          : i18next.t("common.somethingWentWrong"),
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return { handleSubmit, isSubmitting, submitError };
}
