// Outlet is the placeholder where the matched child route renders
import { Outlet } from "react-router-dom";
import Header from "./Header";
import Footer from "./Footer";
import { ErrorBoundary } from "./ErrorBoundary";

export default function Layout() {
  return (
    <div className="bg-surface text-foreground flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        {/* A page that throws loses the page, not the app. Without this the
            only boundary is above the router (main.tsx), so one bad render
            takes the header and nav down with it. */}
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
      <Footer />
    </div>
  );
}
