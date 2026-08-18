import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { changePasswordSchema, type ChangePasswordFormValues } from "../../schemas/changePassword";
import Button from "../objects/Button.tsx";
import { useState } from "react";

export function ChangePasswordSection() {
  const [isEditing, setEditing] = useState(false);

  const form = useForm<ChangePasswordFormValues>({
    resolver: zodResolver(changePasswordSchema),
    mode: "onBlur",
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: ChangePasswordFormValues) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    setEditing(false);
  };
  // TODO: blocked by #109 Add hooks to save data to backend

  return (
    <Form form={form} onSubmit={handleSubmit} className="max-w-64" isEditing={isEditing}>
      <div className="space-y-2">
        {isEditing ? (
          <>
            <div className="flex flex-col gap-4">
              <FormField
                label="Current password"
                name="currentPassword"
                type="password"
                validateOnChange
              />
              <FormField label="New password" name="newPassword" type="password" validateOnChange />
              <FormField
                label="Confirm password"
                name="confirmPassword"
                type="password"
                validateOnChange
              />
            </div>
            <div className="flex flex-row gap-2">
              <Button variant="primary" type="submit" disabled={!form.formState.isValid}>
                {/* TODO: blocked by #109 Insert API here */}
                Save
              </Button>
              {/* TODO: blocked by #109 Using states, once forms are live we can make cancel only appear if user is in edit mode */}
              <Button
                variant="secondary"
                onClick={() => {
                  form.reset();
                  setEditing(false);
                }}
              >
                Cancel
              </Button>
            </div>
          </>
        ) : (
          <>
            <div>********</div>
            <Button variant="primary" onClick={() => setEditing(true)}>
              Edit Password
            </Button>
          </>
        )}
      </div>
    </Form>
  );
}
