import { Link, useNavigate } from "react-router-dom";
import type { ReactNode } from "react";
import ThemeToggle from "./ThemeToggle";

interface LayoutProps {
  children: ReactNode;
  /** Left of brand (e.g. hamburger) */
  navStart?: ReactNode;
  /** Center of navbar */
  navCenter?: ReactNode;
  /** Before logout / theme (e.g. status + compile) */
  navEnd?: ReactNode;
  /** Brand link target */
  brandTo?: string;
}

export default function Layout({
  children,
  navStart,
  navCenter,
  navEnd,
  brandTo = "/projects",
}: LayoutProps) {
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
      <header className="navbar app-navbar px-1 sm:px-3 gap-1">
        <div className="navbar-start gap-0.5 min-w-0 shrink">
          {navStart}
          <Link
            to={brandTo}
            className="btn btn-ghost text-base sm:text-lg font-semibold tracking-tight normal-case min-h-[var(--touch-min)] px-2"
          >
            superfolha
          </Link>
        </div>
        {navCenter ? (
          <div className="navbar-center">{navCenter}</div>
        ) : null}
        <div className="navbar-end gap-1 flex-nowrap shrink-0">
          {navEnd}
          <button
            type="button"
            className="btn btn-ghost min-h-[var(--touch-min)] px-2 sm:px-4"
            onClick={logout}
          >
            Logout
          </button>
          <ThemeToggle />
        </div>
      </header>
      <main className="app-main flex flex-col min-h-0">{children}</main>
    </div>
  );
}
