import { Link } from "react-router-dom";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import UserMenu from "./UserMenu";
import { useAuthStatus } from "../hooks/useAuthStatus";

interface LayoutProps {
  children: ReactNode;
  navStart?: ReactNode;
  navCenter?: ReactNode;
  navEnd?: ReactNode;
  brandTo?: string;
}

export default function Layout({
  children,
  navStart,
  navCenter,
  navEnd,
  brandTo = "/projects",
}: LayoutProps) {
  const { t } = useTranslation("common");
  const { email } = useAuthStatus();

  return (
    <div className="app-shell bg-base-100 text-base-content">
      <header className="navbar app-navbar px-1 sm:px-3 gap-1">
        <div className="navbar-start gap-0.5 min-w-0 shrink">
          {navStart}
          <Link
            to={brandTo}
            className="btn btn-ghost text-base sm:text-lg font-semibold tracking-tight normal-case min-h-[var(--touch-min)] px-2"
          >
            {t("app_name")}
          </Link>
        </div>
        {navCenter ? <div className="navbar-center">{navCenter}</div> : null}
        <div className="navbar-end gap-1 flex-nowrap shrink-0 items-center">
          {navEnd}
          {/* Avatar is always rightmost */}
          <UserMenu email={email} showLogout />
        </div>
      </header>
      <main className="app-main">{children}</main>
    </div>
  );
}
