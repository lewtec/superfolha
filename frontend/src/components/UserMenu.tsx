import { useEffect, useId, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { LogOut, Moon, Sun, User, Folder } from "feather-icons-react";
import { useTheme } from "../hooks/useTheme";
import { supportedLngs, type SupportedLng } from "../i18n";

const langLabels: Record<SupportedLng, string> = {
  en: "English",
  pt: "Português",
  es: "Español",
};

function initialsFromEmail(email: string | null | undefined): string {
  if (!email) return "";
  const local = email.split("@")[0] || email;
  const parts = local.split(/[._\s-]+/).filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return local.slice(0, 2).toUpperCase();
}

interface UserMenuProps {
  email?: string | null;
  /** Show logout (authenticated app shell) */
  showLogout?: boolean;
}

export default function UserMenu({
  email = null,
  showLogout = false,
}: UserMenuProps) {
  const { t, i18n } = useTranslation("common");
  const { isDark, toggle } = useTheme();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuId = useId();

  const currentLng = (i18n.resolvedLanguage || i18n.language || "en").slice(
    0,
    2,
  ) as SupportedLng;

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const goToProjects = () => {
    setOpen(false);
    navigate("/projects");
  };

  const leave = async () => {
    setOpen(false);
    try {
      await fetch("/api/logout", {
        method: "POST",
        credentials: "same-origin",
      });
    } catch {
      // still navigate
    }
    navigate("/login");
  };

  const initials = initialsFromEmail(email);

  return (
    <div className="relative shrink-0" ref={rootRef}>
      <button
        type="button"
        className="btn btn-ghost btn-circle avatar placeholder min-h-[var(--touch-min)] min-w-[var(--touch-min)]"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        aria-label={t("account_menu")}
        onClick={() => setOpen((v) => !v)}
      >
        <div className="bg-primary text-primary-content w-10 h-10 rounded-full flex items-center justify-center text-sm font-semibold">
          {initials ? <span>{initials}</span> : <User size={20} aria-hidden />}
        </div>
      </button>

      {open ? (
        <div
          id={menuId}
          role="menu"
          className="absolute right-0 mt-2 z-50 w-64 rounded-box border border-base-300 bg-base-100 shadow-lg p-2"
        >
          {email ? (
            <div className="px-3 py-2 border-b border-base-300 mb-2">
              <p className="text-xs text-base-content/60">
                {t("signed_in_as")}
              </p>
              <p className="text-sm font-medium truncate" title={email}>
                {email}
              </p>
            </div>
          ) : null}

          <div className="px-2 py-1.5">
            <p className="text-xs font-medium text-base-content/60 mb-1.5 px-1">
              {t("theme")}
            </p>
            <button
              type="button"
              role="menuitem"
              className="btn btn-ghost btn-sm w-full justify-start gap-2 min-h-[var(--touch-min)]"
              onClick={() => toggle()}
            >
              {isDark ? <Sun size={18} /> : <Moon size={18} />}
              {isDark ? t("theme_light") : t("theme_dark")}
            </button>
          </div>

          <div className="px-2 py-1.5">
            <label className="form-control w-full">
              <span className="text-xs font-medium text-base-content/60 mb-1.5 px-1">
                {t("language")}
              </span>
              <select
                className="select select-bordered select-sm w-full min-h-[var(--touch-min)]"
                value={supportedLngs.includes(currentLng) ? currentLng : "en"}
                onChange={(e) => {
                  void i18n.changeLanguage(e.target.value);
                }}
                onClick={(e) => e.stopPropagation()}
              >
                {supportedLngs.map((lng) => (
                  <option key={lng} value={lng}>
                    {langLabels[lng]}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {showLogout ? (
            <div className="border-t border-base-300 mt-2 pt-2 px-2 flex flex-col gap-0.5">
              <button
                type="button"
                role="menuitem"
                className="btn btn-ghost btn-sm w-full justify-start gap-2 min-h-[var(--touch-min)]"
                onClick={goToProjects}
              >
                <Folder size={18} />
                {t("my_projects")}
              </button>
              <button
                type="button"
                role="menuitem"
                className="btn btn-ghost btn-sm w-full justify-start gap-2 text-error min-h-[var(--touch-min)]"
                onClick={leave}
              >
                <LogOut size={18} />
                {t("leave")}
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
