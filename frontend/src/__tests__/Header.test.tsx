import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import Header from "../components/layout/Header";
import { ModalProvider } from "../providers/ModalProvider";

test("renders the brand name", () => {
  render(
    <ModalProvider>
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    </ModalProvider>,
  );
  expect(screen.getByText("Forage Marketplace")).toBeInTheDocument();
});
