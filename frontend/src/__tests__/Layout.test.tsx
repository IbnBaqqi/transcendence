import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

// Throwing is controlled from the test rather than by a render counter:
// React retries a failed concurrent render synchronously, so "throw the first
// time only" succeeds on that retry and the fallback never appears.
let shouldThrow = true;
function MaybeThrows() {
  if (shouldThrow) {
    throw new Error("page blew up during render");
  }
  return <p>second time lucky</p>;
}

function renderWithPage(element: React.ReactNode) {
  return render(
    <AuthContext.Provider value={AUTH_STUB}>
      <ModalProvider>
        <MemoryRouter initialEntries={["/boom"]}>
          <Routes>
            <Route element={<Layout />}>
              {/* Home is what the header's link points at, so the recovery
                  test can navigate the way a real user would. */}
              <Route path="/" element={<p>home page content</p>} />
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
  shouldThrow = true;
  vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Layout", () => {
  test("catches a page that throws without taking the shell down", () => {
    renderWithPage(<Boom />);

    // The boundary inside <main> caught it, and the header survived - without
    // the nested boundary the only one is above the router, and this link
    // would be gone too.
    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Metsätori" })).toBeInTheDocument();
  });

  // Both header links point at "/", so if Home is the page that throws, a
  // boundary keyed on pathname never retries and the user has no way out.
  // location.key changes on every navigation, including to the current path.
  test("recovers when the user re-navigates to the page they are already on", async () => {
    const user = userEvent.setup();
    render(
      <AuthContext.Provider value={AUTH_STUB}>
        <ModalProvider>
          <MemoryRouter initialEntries={["/"]}>
            <Routes>
              <Route element={<Layout />}>
                <Route path="/" element={<MaybeThrows />} />
              </Route>
            </Routes>
          </MemoryRouter>
        </ModalProvider>
      </AuthContext.Provider>,
    );

    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument();

    // The page is healthy now; only a boundary that actually retries will
    // notice. Keyed on pathname it would not, since "/" has not changed.
    shouldThrow = false;
    await user.click(screen.getByRole("link", { name: "Home" }));

    expect(screen.getByText("second time lucky")).toBeInTheDocument();
    expect(screen.queryByText(/something went wrong/i)).not.toBeInTheDocument();
  });

  // A surviving header is only useful if it works. hasError is one-way and
  // this boundary outlives child routes, so without key={pathname} in Layout
  // the fallback stays on screen after navigating and every link looks dead.
  test("recovers when the user navigates away", async () => {
    const user = userEvent.setup();
    renderWithPage(<Boom />);

    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "Home" }));

    expect(screen.getByText("home page content")).toBeInTheDocument();
    expect(screen.queryByText(/something went wrong/i)).not.toBeInTheDocument();
  });

  test("renders the page normally when nothing throws", () => {
    renderWithPage(<p>all good</p>);

    expect(screen.getByText("all good")).toBeInTheDocument();
    expect(screen.queryByText(/something went wrong/i)).not.toBeInTheDocument();
  });
});
