import { createContext, useContext } from "react";

type FormContextValue = {
  isEditing?: boolean;
};

export const FormContext = createContext<FormContextValue>({});

export function useFormConfig() {
  return useContext(FormContext);
}
