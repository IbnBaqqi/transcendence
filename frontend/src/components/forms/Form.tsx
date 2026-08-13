import { FormProvider, type UseFormReturn, type FieldValues } from "react-hook-form";
import { ReactNode } from "react";

type FormProps<T extends FieldValues> = {
  form: UseFormReturn<T>;
  onSubmit: (data: T) => void;
  children: ReactNode;
  className?: string;
};

export function Form<T extends FieldValues>({ form, onSubmit, children, className }: FormProps<T>) {
  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className={className}>
        {children}
      </form>
    </FormProvider>
  );
}
