import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormTextArea } from "./FormTextArea";
import { bioSchema, type BioFormValues } from "../../schemas/common";
import Button from "../objects/Button.tsx";
import { useState } from "react";

export function BioSection() {
  const [isEditing, setEditing] = useState(false);

  const form = useForm<BioFormValues>({
    resolver: zodResolver(bioSchema),
    mode: "onBlur",
    // TODO: blocked by #109 Add hooks to fetch data from backend (or maybe local frontend e.g. from Profile.tsx?)
  });

  const handleSubmit = (data: BioFormValues) => {
    console.log(data);
    // TODO: blocked by #109 Save to API here
    setEditing(false);
  };
  // TODO: blocked by #109 Add hooks to save data to backend

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={isEditing}>
      <div className="space-y-2">
        <div className="flex flex-row gap-4">
          <FormTextArea name="bio" validateOnChange />
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
              Edit Text
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
