import { Controller } from "react-hook-form";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { contactDetailsSchema, ContactDetailsFormValues } from "../../schemas/contactDetails";

function ContactDetailsSection() {
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
      <FormField label="First name" name="firstName" />
      <FormField label="Last name" name="lastName" />
      <FormField label="Phone" name="phone" type="tel" />
      <FormField label="City" name="location" />
      <button type="submit">Save</button>
      {/* Also need to update this once hooks in place */}
      {/* <button type="submit" disabled={isSubmitting}> */}
      {/*   {isSubmitting? "Saving..." : "Save"} */}
      {/* </button> */}
    </Form>
  );
}
