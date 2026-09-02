// runs before the test files (via `setupFiles` in vite.config.ts).
// side-effect import: registers jest-dom's matchers onto `expect`, so you get
// readable DOM assertions like toBeInTheDocument(), toHaveTextContent(), etc.
import "@testing-library/jest-dom";
// initializes the i18next singleton (en default + zod locale config) so
// components using useTranslation() render in a deterministic language.
import "../i18n";
