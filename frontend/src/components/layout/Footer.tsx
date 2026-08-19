// footer links don't need active styling so only Link is needed
import { Link } from "react-router-dom";

export default function Footer() {
  return (
    <footer className="border-line bg-surface border-t">
      {/* sm is meant for wider screens */}
      <div className="text-muted mx-auto flex max-w-6xl flex-col gap-2 px-4 py-6 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p>© {new Date().getFullYear()} Metsätori</p>
        <nav className="flex gap-4">
          <Link to="/privacy" className="hover:text-foreground">
            Privacy Policy
          </Link>
          <Link to="/terms" className="hover:text-foreground">
            Terms of Service
          </Link>
        </nav>
      </div>
    </footer>
  );
}
