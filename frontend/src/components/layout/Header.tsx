// Link: normal navigation link (no "am I active?" info)
// NavLink: a Link that knows if it points to the current page, so you can style the active one differently
import { useState } from "react";
import { useModal } from "../../providers/ModalProvider";
import { Link, NavLink } from "react-router-dom";
import Avatar from "../objects/Avatar.tsx";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  isActive ? "text-foreground" : "text-muted hover:text-foreground";

export default function Header() {
  const [isLoggedIn] = useState(false);
  {
    /* const [isLoggedIn, setLoggedIn] = useState(false); */
    /* just a placeholder state until we pull from backend with auth */
  }
  const { openModal } = useModal();

  return (
    <header className="border-line bg-surface sticky top-0 z-10 border-b">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <Link to="/" className="text-accent text-lg font-bold">
          Forage Marketplace
        </Link>
        <nav className="flex items-center gap-6 text-sm">
          {/* {navLinkClass} passes the function itself and React Router calls it and supplies { isActive } */}
          <NavLink to="/" className={navLinkClass}>
            Home
          </NavLink>
          {isLoggedIn ? (
            <Link to="/profile">
              <Avatar size="sm" initials="OR" /> {/* OR just a placeholder for now */}
            </Link>
          ) : (
            <button onClick={() => openModal("login")}>
              <Avatar size="sm" initials="?" />
            </button>
          )}
          {/* TODO: add Listing (#20) and auth links (#46) when those pages exist */}
        </nav>
      </div>
    </header>
  );
}
