import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import Header from "../components/layout/Header";

test("renders the brand name", () => {
  render(<Header />, { wrapper: MemoryRouter });
  expect(screen.getByText("Forage Marketplace")).toBeInTheDocument();
});
