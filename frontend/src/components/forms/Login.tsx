// NOTE: We maybe don't need this file, the form can be in /modal/LoginModal.tsx
// or alternatively we pass data about the modal state down to the form which then
// passes back up from the form, needs checking

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { loginSchema, type LoginFormSchema } from "../../schemas/login";
import Button from "../objects/Button.tsx";
import { useState } from "react";

export function LoginSection() {
  const [isEditing, setEditing] = useState(false);

  const form = useForm<LoginFormSchema>({
    resolver: zodResolver(loginSchema),
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: LoginFormSchema) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    setEditing(false);
  };
  // TODO: blocked by #109 Add hooks to save data to backend

  return (
    <Form form={form} onSubmit={handleSubmit}>
      <div className="space-y-2">
        <FormField label="Email" name="email" isEditing={isEditing} />
        <FormField label="Password" name="password" isEditing={isEditing} />
        <div className="flex flex-row gap-2">
          <Button variant="primary" type="submit">
            {/* TODO: blocked by #109 Insert API here */}
            Log In
          </Button>
          {/* TODO: blocked by #109 Using states, once forms are live we can make cancel only appear if user is in edit mode */}
          <Button
            variant="secondary"
            type="button"
            onClick={() => {
              form.reset();
              setEditing(false);
            }}
          >
            Cancel
          </Button>
        </div>
      </div>
    </Form>
  );
}
