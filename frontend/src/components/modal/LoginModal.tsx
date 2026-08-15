import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "../forms/Form";
import { FormField } from "../forms/FormField";
import { loginSchema, type LoginFormSchema } from "../../schemas/login";
import Button from "../objects/Button.tsx";
import { useModal } from "../../providers/ModalProvider";

export function LoginModal() {
  const { closeModal } = useModal();

  const form = useForm<LoginFormSchema>({
    resolver: zodResolver(loginSchema),
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: LoginFormSchema) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    closeModal();
  };
  // TODO: blocked by #109 Add hooks to save data to backend

  return (
    <div className="p-6">
      <h2 className="mb-4 text-lg font-semibold">Log in</h2>
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
            {/* TODO: blocked by #109 Using states, once forms are live we can make cancel only appear if user is in edit mode */}
            <Button variant="secondary" type="button" onClick={closeModal}>
              Cancel
            </Button>
          </div>
        </div>
      </Form>
    </div>
  );
}
