// Outlet is the placeholder where the matched child route renders
import { Outlet, useLocation } from "react-router-dom";
import Header from "./Header";
import Footer from "./Footer";
import { ErrorBoundary } from "./ErrorBoundary";

// A page that throws loses the page, not the app: the only other boundary is
// above the router (main.tsx), so without this one a bad render takes the
// header and nav down with it.
//
// The key is what makes that surviving header useful rather than decorative.
// hasError is one-way and this instance outlives child routes, so a boundary
// with a fixed key keeps showing the fallback after the user has clicked away.
export default function Layout() {
  // location.key changes on every navigation, including to the page you are
  // already on - which pathname does not. See the note above.
  const { key } = useLocation();

  return (
    <div className="bg-surface text-foreground flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        {/* See the note above the component for why this is keyed. */}
        <ErrorBoundary key={key} message="Try another page, or refresh to start over.">
          <Outlet />
        </ErrorBoundary>
      </main>
      <Footer />
    </div>
  );
}
