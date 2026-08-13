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
    // TODO: Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
    // defaultValues: {
    //   firstName: user.firstName ?? "",
    //   lastName: user.lastName ?? "",
    //   phone: user.phone ?? "",
    //   city: user.city ?? "",
    // },
  });

  const handleSubmit = (data: ContactDetailsFormValues) => {
    console.log(data);
    // Save to API here
    setEditing(false);
  };
  // TODO: Add hooks to save data to backend
  // const { handleSubmit: handleSave, isSubmitting, submitError } = useFormSubmit<ContactDetailsFormValues>(
  // async (data) => {
  //   await api.updateContactDetails(data); // whatever your API call looks like
  // }

  return (
    <Form form={form} onSubmit={handleSubmit}>
      <div className="space-y-1">
        <div className="flex flex-row gap-4">
          <FormField label="First name" name="firstName" isEditing={isEditing} />
          <FormField label="Last name" name="lastName" isEditing={isEditing} />
          <FormField label="Phone" name="phone" type="tel" isEditing={isEditing} />
          <FormField label="City" name="city" isEditing={isEditing} />
        </div>
        <div className="flex flex-row gap-2">
          {isEditing ? (
            <>
              <Button variant="primary" type="submit">
                {/* Insert API here */}
                Save
              </Button>
              {/* TODO(#): Using states, once forms are live we can make cancel only appear if user is in edit mode */}
              <Button variant="secondary" type="button" onClick={() => {
                  form.reset();
                  setEditing(false);
                }}
              >
                Cancel
              </Button>
            </>
          ) : (
            <Button variant="primary" type="button" onClick={() => setEditing(true)}>
              Edit Details
            </Button>
          )}
        </div>
      </div>
      {/* <button type="submit">Save</button> */}
      {/* Also need to update this once hooks in place */}
      {/* <button type="submit" disabled={isSubmitting}> */}
      {/*   {isSubmitting? "Saving..." : "Save"} */}
      {/* </button> */}
    </Form>
  );
}
