import { Link, useNavigate } from "react-router-dom";
import ThemeToggle from "./ThemeToggle";

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const navigate = useNavigate();

  const logout = async () => {
    try {
      await fetch("/api/logout", {
        method: "POST",
        credentials: "same-origin",
      });
    } catch {
      // Still leave the app even if the network call fails.
    }
    navigate("/login");
  };

  return (
    <div className="app-shell bg-base-100 text-base-content">
      <header className="navbar app-navbar px-2 sm:px-4">
        <div className="navbar-start">
          <Link
            to="/projects"
            className="btn btn-ghost text-lg font-semibold tracking-tight normal-case min-h-[var(--touch-min)]"
          >
            superfolha
          </Link>
        </div>
        <div className="navbar-end gap-1">
          <button
            type="button"
            className="btn btn-ghost min-h-[var(--touch-min)]"
            onClick={logout}
          >
            Logout
          </button>
          <ThemeToggle />
        </div>
      </header>
      <main className="app-main">{children}</main>
    </div>
  );
}
