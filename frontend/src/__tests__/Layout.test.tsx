import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";

const AUTH_STUB: AuthContextValue = {
  user: null,
  isLoading: false,
  login: vi.fn(),
  signup: vi.fn(),
  logout: vi.fn(),
};

function Boom(): never {
  throw new Error("page blew up during render");
}

function renderWithPage(element: React.ReactNode) {
  return render(
    <AuthContext.Provider value={AUTH_STUB}>
      <ModalProvider>
        <MemoryRouter initialEntries={["/boom"]}>
          <Routes>
            <Route element={<Layout />}>
              <Route path="/boom" element={element} />
            </Route>
          </Routes>
        </MemoryRouter>
      </ModalProvider>
    </AuthContext.Provider>,
  );
}

// React logs the caught error to console.error; silence it so a passing test
// doesn't print a stack trace that reads like a failure.
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Layout", () => {
  test("keeps the shell usable when a page throws during render", () => {
    renderWithPage(<Boom />);

    // The boundary inside <main> caught it...
    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument();
    // ...and the header survived, so the user can navigate away instead of
    // facing a blank app. Without the nested boundary the only one is above
    // the router and this link would be gone too.
    expect(screen.getByRole("link", { name: "Metsätori" })).toBeInTheDocument();
  });

  test("renders the page normally when nothing throws", () => {
    renderWithPage(<p>all good</p>);

    expect(screen.getByText("all good")).toBeInTheDocument();
    expect(screen.queryByText(/something went wrong/i)).not.toBeInTheDocument();
  });
});
