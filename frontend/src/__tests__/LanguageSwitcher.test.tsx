import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { LanguageSwitcher } from "../components/layout/LanguageSwitcher";

test("renders the three supported languages", () => {
  render(<LanguageSwitcher />);
  expect(screen.getByRole("combobox")).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "English" })).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "Suomi" })).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "Svenska" })).toBeInTheDocument();
});

test("switches the active language on change", async () => {
  const user = userEvent.setup();
  render(<LanguageSwitcher />);
  const select = screen.getByRole("combobox");
  expect(select).toHaveProperty("value", "en");
  expect(select).toHaveAccessibleName("Language");

  await user.selectOptions(select, "fi");
  await waitFor(() => expect(select).toHaveProperty("value", "fi"));
  // the aria-label follows the active locale
  expect(select).toHaveAccessibleName("Kieli");
  // ...and so does the <html lang> attribute (a11y + browser translation)
  expect(document.documentElement.lang).toBe("fi");
});
