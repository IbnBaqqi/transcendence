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
    mode: "onBlur",
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
    <Form form={form} onSubmit={handleSubmit} className="max-w-fit" isEditing={isEditing}>
      <div className="space-y-2">
        <div className="grid grid-cols-2 gap-4">
          <FormField label="First name" name="firstName" validateOnChange/>
          <FormField label="Last name" name="lastName" />
          <FormField label="Phone" name="phone" type="tel" />
          <FormField label="City" name="city" validateOnChange />
        </div>
        <div className="flex flex-row gap-2">
          {isEditing ? (
            <>
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
