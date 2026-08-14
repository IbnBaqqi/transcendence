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
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: ChangePasswordFormValues) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    setEditing(false);
  };
  // TODO: blocked by #109 Add hooks to save data to backend

  return (
    <Form form={form} onSubmit={handleSubmit}>
      <div className="space-y-2">
        {isEditing ? (
          <>
            <div className="flex flex-row gap-4">
              <FormField label="Current password" name="currentPassword" isEditing={isEditing} />
              <FormField label="New password" name="newPassword" isEditing={isEditing} />
              <FormField label="Confirm password" name="confirmPassword" isEditing={isEditing} />
            </div>
            <div className="flex flex-row gap-2">
              <Button variant="primary" type="submit">
                {/* TODO: blocked by #109 Insert API here */}
                Save
              </Button>
              {/* TODO: blocked by #109 Using states, once forms are live we can make cancel only appear if user is in edit mode */}
              <Button variant="secondary" type="button" onClick={() => {
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
            <Button variant="primary" type="button" onClick={() => setEditing(true)}>
              Edit Password
            </Button>
          </>
        )}
      </div>
    </Form>
  );
}

