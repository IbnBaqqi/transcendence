// runs before the test files (via `setupFiles` in vite.config.ts).
// side-effect import: registers jest-dom's matchers onto `expect`, so you get
// readable DOM assertions like toBeInTheDocument(), toHaveTextContent(), etc.
import "@testing-library/jest-dom";
