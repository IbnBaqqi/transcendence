import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { loginSchema, type LoginFormSchema } from "../../schemas/login";
import Button from "../objects/Button.tsx";

export function LoginSection({ onCancel }: { onCancel: () => void }) {
  const form = useForm<LoginFormSchema>({
    resolver: zodResolver(loginSchema),
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: LoginFormSchema) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
  };

  return (
    <Form form={form} onSubmit={handleSubmit}>
      <div className="space-y-4">
        <div className="space-y-2">
          <FormField label="Email" name="email" isEditing={true} />
          <FormField label="Password" name="password" type="password" isEditing={true} />
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" type="submit">
            {/* TODO: blocked by #109 Insert API here */}
            Log In
          </Button>
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    </Form>
  );
}
