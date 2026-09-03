import { FormProvider, type UseFormReturn, type FieldValues } from "react-hook-form";
import { type ReactNode } from "react";
import { FormContext } from "./FormContext";

type FormProps<T extends FieldValues> = {
  form: UseFormReturn<T>;
  // Awaited by react-hook-form, which is what keeps formState.isSubmitting
  // true for the whole handler rather than just its first await.
  onSubmit: (data: T) => void | Promise<void>;
  children: ReactNode;
  className?: string;
  isEditing?: boolean;
};

export function Form<T extends FieldValues>({
  form,
  onSubmit,
  children,
  className,
  isEditing,
}: FormProps<T>) {
  return (
    <FormProvider {...form}>
      <FormContext.Provider value={{ isEditing }}>
        <form onSubmit={form.handleSubmit(onSubmit)} className={className}>
          {children}
        </form>
      </FormContext.Provider>
    </FormProvider>
  );
}
