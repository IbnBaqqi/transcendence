import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { registerSchema, type RegisterFormSchema } from "../../schemas/register";
import Button from "../objects/Button.tsx";

export function RegisterSection({ onClose }: { onClose: () => void }) {
  const form = useForm<RegisterFormSchema>({
    resolver: zodResolver(registerSchema),
    mode: "onBlur",
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: RegisterFormSchema) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        <div className="space-y-2">
          <FormField label="Username" name="username" validateOnChange />
          <FormField label="Email" name="email" validateOnChange />
          <FormField label="Password" name="password" type="password" validateOnChange />
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" type="submit" disabled={!form.formState.isValid}>
            {/* TODO: blocked by #109 Insert API here */}
            Register
          </Button>
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </Form>
  );
}
