import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form } from "./Form";
import { FormField } from "./FormField";
import { loginSchema, type LoginFormSchema } from "../../schemas/login";
import Button from "../objects/Button.tsx";
import { useAuth } from "../../hooks/useAuth";
import { isApiError } from "../../api/client";
import OAuthButtons from "./OAuthButtons";

export function LoginSection({ onClose }: { onClose: () => void }) {
  const { login } = useAuth();
  const form = useForm<LoginFormSchema>({
    resolver: zodResolver(loginSchema),
    mode: "onBlur",
  });
  const {
    formState: { errors, isValid, isSubmitting },
  } = form;

  const handleSubmit = async (data: LoginFormSchema) => {
    form.clearErrors("root");
    try {
      await login(data.email, data.password);
      onClose();
    } catch (err) {
      // The backend deliberately sends one message for a wrong email and a
      // wrong password, and rate limits speak for themselves - trust its copy.
      form.setError("root", {
        message: isApiError(err) ? err.message : "Something went wrong. Please try again.",
      });
    }
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        <OAuthButtons />
        <div className="text-muted flex items-center gap-3 text-xs">
          <span className="border-line h-px flex-1 border-t" />
          or sign in with email
          <span className="border-line h-px flex-1 border-t" />
        </div>
        <div className="space-y-2">
          <FormField label="Email" name="email" validateOnChange />
          <FormField label="Password" name="password" type="password" validateOnChange />
        </div>
        {errors.root?.message && <p className="text-berry-500 text-sm">{errors.root.message}</p>}
        <div className="flex flex-row gap-2">
          <Button variant="primary" type="submit" disabled={!isValid || isSubmitting}>
            {isSubmitting ? "Logging in…" : "Log In"}
          </Button>
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </Form>
  );
}
