import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { BrandMark } from "./Brand";
import UserMenu from "./UserMenu";

export default function Header() {
  const { t } = useTranslation(["common", "auth"]);

  return (
    <header className="app-navbar">
      <div className="navbar px-2 sm:px-4 max-w-6xl mx-auto w-full min-h-[var(--shell-height)]">
        <div className="navbar-start">
          <Link
            to="/"
            aria-label="Superfolha"
            className="btn btn-ghost gap-2 text-lg font-semibold tracking-tight normal-case min-h-[var(--touch-min)]"
          >
            <BrandMark className="h-8 w-8 shrink-0" />
            <span className="hidden sm:inline">Superfolha</span>
          </Link>
        </div>
        <div className="navbar-end gap-1 items-center">
          <Link to="/login" className="btn btn-ghost min-h-[var(--touch-min)]">
            {t("auth:login_button")}
          </Link>
          <Link
            to="/register"
            className="btn btn-primary min-h-[var(--touch-min)]"
          >
            {t("auth:register_button")}
          </Link>
          <UserMenu showLogout={false} />
        </div>
      </div>
    </header>
  );
}
