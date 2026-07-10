import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import ThemeToggle from "./ThemeToggle";

export default function Header() {
  const { t } = useTranslation();

  return (
    <header className="app-navbar">
      <div className="navbar px-2 sm:px-4 max-w-6xl mx-auto w-full min-h-[var(--shell-height)]">
        <div className="navbar-start">
          <Link
            to="/"
            className="btn btn-ghost text-lg font-semibold tracking-tight normal-case min-h-[var(--touch-min)]"
          >
            Superfolha
          </Link>
        </div>
        <div className="navbar-end gap-1">
          <Link
            to="/login"
            className="btn btn-ghost min-h-[var(--touch-min)]"
          >
            {t("login_button")}
          </Link>
          <Link
            to="/register"
            className="btn btn-primary min-h-[var(--touch-min)]"
          >
            {t("register_button")}
          </Link>
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
