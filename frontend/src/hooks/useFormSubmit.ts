import { useState } from "react";
import { isApiError } from "../api/client";

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
        isApiError(err)
          ? err.message
          : err instanceof Error
            ? err.message
            : "Something went wrong",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return { handleSubmit, isSubmitting, submitError };
}
