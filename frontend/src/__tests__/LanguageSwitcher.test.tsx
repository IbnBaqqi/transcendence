import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { LanguageSwitcher } from "../components/objects/LanguageSwitcher";
import i18next from "../i18n";

beforeEach(async () => {
  // the switch test changes language; start every test from the default
  if (i18next.language !== "en") await i18next.changeLanguage("en");
});

test("renders as a collapsed flag button with a language label", () => {
  render(<LanguageSwitcher />);

  expect(screen.getByRole("button", { name: "Language" })).toBeInTheDocument();
  expect(screen.queryByRole("group")).not.toBeInTheDocument();
});

test("expands to a group of flags and native language names on click", async () => {
  const user = userEvent.setup();
  render(<LanguageSwitcher />);

  await user.click(screen.getByRole("button", { name: "Language" }));

  expect(screen.getByRole("group", { name: "Language" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /English/ })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Suomi/ })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Svenska/ })).toBeInTheDocument();
});

test("marks the active language, restores focus, and follows the locale", async () => {
  const user = userEvent.setup();
  render(<LanguageSwitcher />);

  await user.click(screen.getByRole("button", { name: "Language" }));
  await user.click(screen.getByRole("button", { name: /Suomi/ }));

  // selects, closes, and syncs <html lang> + the trigger's accessible name
  expect(document.documentElement.lang).toBe("fi");
  expect(screen.queryByRole("group")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Kieli" })).toBeInTheDocument();
  // the menu just unmounted with the clicked option, so focus must return
  expect(screen.getByRole("button", { name: "Kieli" })).toHaveFocus();

  await user.click(screen.getByRole("button", { name: "Kieli" }));
  expect(screen.getByRole("button", { name: /Suomi/ })).toHaveAttribute("aria-current", "true");
});

test("closes on an outside click", async () => {
  const user = userEvent.setup();
  render(<LanguageSwitcher />);

  await user.click(screen.getByRole("button", { name: "Language" }));
  expect(screen.getByRole("group")).toBeInTheDocument();

  await user.click(document.body);
  await waitFor(() => expect(screen.queryByRole("group")).not.toBeInTheDocument());
});

test("closes on Escape", async () => {
  const user = userEvent.setup();
  render(<LanguageSwitcher />);

  await user.click(screen.getByRole("button", { name: "Language" }));
  expect(screen.getByRole("group")).toBeInTheDocument();

  await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("group")).not.toBeInTheDocument());
});
