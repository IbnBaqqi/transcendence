import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { contactDetailsSchema, type ContactDetailsFormValues } from "../../schemas/contactDetails";
import Button from "../objects/Button.tsx";

export function ContactDetailsSection() {
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
          <FormField label="First name" name="firstName" />
          <FormField label="Last name" name="lastName" />
          <FormField label="Phone" name="phone" type="phone" />
          <FormField label="City" name="location" />
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Edit Details
          </Button>
          {/* TODO(#): Using states, once forms are live we can make cancel only appear if user is in edit mode */}
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
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
