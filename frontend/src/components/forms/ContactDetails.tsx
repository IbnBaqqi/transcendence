import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { contactDetailsSchema, type ContactDetailsFormValues } from "../../schemas/contactDetails";
import Button from "../objects/Button.tsx";
import { useState } from "react";

export function ContactDetailsSection() {
  const [isEditing, setEditing] = useState(false);

  const form = useForm<ContactDetailsFormValues>({
    resolver: zodResolver(contactDetailsSchema),
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
    // defaultValues: {
    //   firstName: user.firstName ?? "",
    //   lastName: user.lastName ?? "",
    //   phone: user.phone ?? "",
    //   city: user.city ?? "",
    // },
  });

  const handleSubmit = (data: ContactDetailsFormValues) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    setEditing(false);
  };
  // TODO: blocked by #109 Add hooks to save data to backend
  // const { handleSubmit: handleSave, isSubmitting, submitError } = useFormSubmit<ContactDetailsFormValues>(
  // async (data) => {
  //   await api.updateContactDetails(data); // whatever your API call looks like
  // }

  return (
    <Form form={form} onSubmit={handleSubmit}>
      <div className="space-y-2">
        <div className="flex flex-row gap-4">
          <FormField label="First name" name="firstName" isEditing={isEditing} />
          <FormField label="Last name" name="lastName" isEditing={isEditing} />
          <FormField label="Phone" name="phone" type="tel" isEditing={isEditing} />
          <FormField label="City" name="city" isEditing={isEditing} />
        </div>
        <div className="flex flex-row gap-2">
          {isEditing ? (
            <>
              <Button variant="primary">
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
            </>
          ) : (
            <Button variant="primary" onClick={() => setEditing(true)}>
              Edit Details
            </Button>
          )}
        </div>
      </div>
    </Form>
  );
}
