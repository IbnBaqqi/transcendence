import { Link } from "react-router-dom";

// rendered by "*" catch-all route for any unknown URL
export default function NotFound() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-16 text-center">
      <h1 className="text-foreground text-3xl font-bold">404 - Page not found</h1>
      <p className="text-muted mt-2">That page doesn't exist or has moved</p>
      <Link to="/" className="text-accent mt-4 inline-block hover:underline">
        Back to home
      </Link>
    </div>
  );
}
