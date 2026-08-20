import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { confirmSchema, type ConfirmFormValues } from "../../schemas/confirm";
import Button from "../objects/Button.tsx";

export function DeleteAccountSection({ onClose }: { onClose: () => void }) {
  const form = useForm<ConfirmFormValues>({
    resolver: zodResolver(confirmSchema),
    mode: "onBlur",
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: ConfirmFormValues) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        <div className="space-y-2">
          <FormField label="Password" name="password" type="password" validateOnChange />
          <FormField
            label="Confirm password"
            name="confirmPassword"
            type="password"
            validateOnChange
          />
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="secondary" type="submit" disabled={!form.formState.isValid}>
            Delete Account
          </Button>
          <Button variant="primary" type="button" onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </Form>
  );
}
