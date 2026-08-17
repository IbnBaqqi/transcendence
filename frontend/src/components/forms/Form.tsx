import { FormProvider, type UseFormReturn, type FieldValues } from "react-hook-form";
import { type ReactNode } from "react";
import { FormContext } from "./FormContext";

type FormProps<T extends FieldValues> = {
  form: UseFormReturn<T>;
  onSubmit: (data: T) => void;
  children: ReactNode;
  className?: string;
  width?: string;
  isEditing?: boolean;
};

export function Form<T extends FieldValues>({
  form,
  onSubmit,
  children,
  className,
  width,
  isEditing,
}: FormProps<T>) {
  return (
    <FormProvider {...form}>
      <FormContext.Provider value={{ isEditing }}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className={`${className ?? ""} ${width ?? ""}`}
        >
          {children}
        </form>
      </FormContext.Provider>
    </FormProvider>
  );
}
